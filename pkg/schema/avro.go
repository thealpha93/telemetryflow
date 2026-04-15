package schema

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/hamba/avro/v2"
)

// magicByte is the first byte of every Confluent wire-format message.
// Its presence signals that the next 4 bytes are a schema ID integer.
// If a message does not start with 0x00 it is not Confluent Avro.
const magicByte byte = 0x00

// Embedded .avsc files — compiled into the binary at build time.
// Services reference these when constructing a Serializer so there
// are no file path assumptions at runtime.
var (
	//go:embed schemas/device_metric.avsc
	DeviceMetricSchema string

	//go:embed schemas/device_alert.avsc
	DeviceAlertSchema string
)

// Serializer encodes Go structs into the Confluent Avro wire format:
//
//	[0x00][schema_id: 4 bytes big-endian][avro binary payload]
//
// The schema is registered with the Schema Registry on construction.
// The returned schema ID is cached — all subsequent Serialize calls
// prepend the same ID with no additional network round-trips.
type Serializer struct {
	subject  string
	schema   avro.Schema
	schemaID int
}

// NewSerializer registers the given schema with the registry under subject,
// parses it, and returns a Serializer ready to encode messages.
//
// subject follows TopicNameStrategy: "{topic}-value"
// e.g. NewSerializer(reg, "device.metrics-value", schema.DeviceMetricSchema)
func NewSerializer(registry *RegistryClient, subject, schemaJSON string) (*Serializer, error) {
	// Parse the schema locally first — catches malformed .avsc files before
	// making a network call to the registry.
	parsed, err := avro.Parse(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("schema: parsing avro schema for subject %q: %w", subject, err)
	}

	// Register (or retrieve existing) schema ID from the registry.
	id, err := registry.GetOrRegisterSchemaID(subject, schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("schema: registering schema for subject %q: %w", subject, err)
	}

	return &Serializer{
		subject:  subject,
		schema:   parsed,
		schemaID: id,
	}, nil
}

// Serialize encodes v into Confluent Avro wire format bytes.
// v must be a struct whose fields match the Avro schema — hamba/avro
// uses the `avro:""` struct tags to map Go field names to Avro field names.
func (s *Serializer) Serialize(v any) ([]byte, error) {
	// Encode v into raw Avro binary (no wire format prefix yet).
	payload, err := avro.Marshal(s.schema, v)
	if err != nil {
		return nil, fmt.Errorf("schema: avro marshal for subject %q: %w", s.subject, err)
	}

	// Prepend the 5-byte Confluent wire format header.
	//   buf[0]   = 0x00 (magic byte)
	//   buf[1:5] = schema ID as big-endian uint32
	//   buf[5:]  = avro binary payload
	buf := make([]byte, 5+len(payload))
	buf[0] = magicByte
	binary.BigEndian.PutUint32(buf[1:5], uint32(s.schemaID))
	copy(buf[5:], payload)

	return buf, nil
}

// Deserializer decodes Confluent Avro wire format bytes into Go structs.
//
// On each decode it:
//  1. Validates the magic byte (0x00)
//  2. Extracts the 4-byte schema ID
//  3. Fetches the schema from the Registry (or local cache) using that ID
//  4. Decodes the Avro binary payload using the fetched schema
//
// This means the Deserializer handles multiple schema versions on the same
// topic transparently — the schema ID in each message tells it exactly which
// version was used to encode it.
//
// All schema lookups are cached — repeated calls for the same schema ID
// make no network request.
type Deserializer struct {
	registry *RegistryClient

	// schemas caches parsed avro.Schema objects keyed by schema ID.
	mu      sync.RWMutex
	schemas map[int]avro.Schema
}

// NewDeserializer creates a Deserializer backed by the given RegistryClient.
func NewDeserializer(registry *RegistryClient) *Deserializer {
	return &Deserializer{
		registry: registry,
		schemas:  make(map[int]avro.Schema),
	}
}

// Deserialize decodes Confluent wire format bytes into v.
// v must be a pointer to a struct compatible with the encoded Avro schema.
func (d *Deserializer) Deserialize(data []byte, v any) error {
	if len(data) < 5 {
		return fmt.Errorf("schema: message too short (%d bytes) — not Confluent wire format", len(data))
	}
	if data[0] != magicByte {
		return fmt.Errorf("schema: invalid magic byte 0x%02x (expected 0x00) — not Confluent wire format", data[0])
	}

	schemaID := int(binary.BigEndian.Uint32(data[1:5]))

	schema, err := d.getOrParseSchema(schemaID)
	if err != nil {
		return fmt.Errorf("schema: fetching schema id %d from registry: %w", schemaID, err)
	}

	if err := avro.Unmarshal(schema, data[5:], v); err != nil {
		return fmt.Errorf("schema: avro unmarshal (schema id %d): %w", schemaID, err)
	}

	return nil
}

// getOrParseSchema returns the parsed avro.Schema for the given ID,
// fetching and parsing it on the first call then caching the result.
func (d *Deserializer) getOrParseSchema(id int) (avro.Schema, error) {
	// Fast path: already parsed and cached.
	d.mu.RLock()
	if s, ok := d.schemas[id]; ok {
		d.mu.RUnlock()
		return s, nil
	}
	d.mu.RUnlock()

	// Fetch schema JSON from registry (the registry handles its own cache).
	schemaJSON, err := d.registry.GetSchemaByID(id)
	if err != nil {
		return nil, err
	}

	parsed, err := avro.Parse(schemaJSON)
	if err != nil {
		return nil, fmt.Errorf("parsing schema id %d: %w", id, err)
	}

	d.mu.Lock()
	d.schemas[id] = parsed
	d.mu.Unlock()

	return parsed, nil
}
