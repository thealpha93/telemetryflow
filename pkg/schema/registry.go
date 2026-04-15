// Package schema provides Confluent Schema Registry integration and
// Avro serialization/deserialization for TelemetryFlow's Kafka topics.
//
// Wire format: every Avro message on a Confluent topic uses the Confluent
// wire format — a 5-byte magic prefix:
//
//	byte 0:   0x00  (magic byte — identifies Confluent wire format)
//	bytes 1-4: schema ID (big-endian int32) — fetched from Schema Registry
//	bytes 5+:  Avro binary payload
//
// This package handles encoding/decoding that prefix transparently.
package schema

// RegistryClient communicates with the Confluent Schema Registry to:
//   - Register new schemas on first use
//   - Fetch schema IDs for serialization
//   - Fetch schema definitions for deserialization (cached locally)
//
// Authentication uses HTTP Basic auth with the Schema Registry API key/secret.
//
// Schema compatibility is enforced server-side. If a schema change violates
// the configured compatibility mode (BACKWARD by default on Confluent Cloud),
// registration will fail — catching breaking changes before they reach Kafka.
type RegistryClient struct {
	// TODO: http client, base URL, credentials, local schema cache
}

// NewRegistryClient creates and returns an authenticated RegistryClient.
func NewRegistryClient() (*RegistryClient, error) {
	panic("not implemented — implement in Phase 1")
}

// GetOrRegisterSchemaID returns the integer schema ID for the given subject
// and schema definition. If the schema is already registered, the existing
// ID is returned. Otherwise the schema is registered and the new ID is returned.
//
// Subject naming follows TopicNameStrategy: "{topic}-value"
// e.g. "device.metrics-value", "device.alerts-value"
func (r *RegistryClient) GetOrRegisterSchemaID() (int, error) {
	panic("not implemented — implement in Phase 1")
}

// GetSchemaByID fetches and returns the Avro schema definition for a given
// schema ID. Results are cached in memory after the first fetch.
func (r *RegistryClient) GetSchemaByID() (string, error) {
	panic("not implemented — implement in Phase 1")
}
