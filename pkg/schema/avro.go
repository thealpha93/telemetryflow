package schema

// Serializer encodes Go structs into the Confluent Avro wire format:
//
//	[0x00][schema_id (4 bytes BE)][avro binary payload]
//
// The schema ID is fetched from the Registry on first use and cached.
// Subsequent calls for the same subject reuse the cached ID with no
// network round-trip.
type Serializer struct {
	// TODO: embed *RegistryClient, subject -> schema ID cache
}

// NewSerializer creates a Serializer for the given topic subject.
func NewSerializer() (*Serializer, error) {
	panic("not implemented — implement in Phase 1")
}

// Serialize encodes v into Confluent Avro wire format bytes.
// v must match the Avro schema registered for this serializer's subject.
func (s *Serializer) Serialize() ([]byte, error) {
	panic("not implemented — implement in Phase 1")
}

// Deserializer decodes Confluent Avro wire format bytes into Go structs.
//
// On each decode it:
//  1. Strips the 5-byte magic prefix
//  2. Extracts the schema ID
//  3. Fetches the schema from Registry (or local cache) using that ID
//  4. Decodes the Avro binary payload using the fetched schema
//
// This means the deserializer automatically handles multiple schema versions
// on the same topic — the schema ID in the message tells us exactly which
// version was used to encode it.
type Deserializer struct {
	// TODO: embed *RegistryClient
}

// NewDeserializer creates a Deserializer backed by the given RegistryClient.
func NewDeserializer() (*Deserializer, error) {
	panic("not implemented — implement in Phase 1")
}

// Deserialize decodes Confluent wire format bytes into v.
// v must be a pointer to a struct compatible with the encoded Avro schema.
func (d *Deserializer) Deserialize() error {
	panic("not implemented — implement in Phase 1")
}
