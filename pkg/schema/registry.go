// Package schema provides Confluent Schema Registry integration and
// Avro serialization/deserialization for TelemetryFlow's Kafka topics.
//
// Wire format: every Avro message on a Confluent topic uses a 5-byte prefix:
//
//	byte 0:    0x00  (magic byte — identifies Confluent wire format)
//	bytes 1-4: schema ID (big-endian uint32) — assigned by Schema Registry
//	bytes 5+:  Avro binary payload
//
// This package handles encoding/decoding that prefix transparently.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// contentType is required by the Confluent Schema Registry REST API.
// Using the wrong Content-Type returns a 415 Unsupported Media Type.
const contentType = "application/vnd.schemaregistry.v1+json"

// RegistryClient communicates with the Confluent Schema Registry to:
//   - Register new schemas on first use (returns a stable integer ID)
//   - Fetch schema definitions by ID for deserialization (cached locally)
//
// All methods are safe for concurrent use.
type RegistryClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client

	// cache stores schema JSON strings keyed by their integer ID.
	// Written once on first fetch, then read-only — safe with RWMutex.
	mu    sync.RWMutex
	cache map[int]string
}

// NewRegistryClient creates and returns an authenticated RegistryClient.
// baseURL is the Schema Registry endpoint, e.g. https://psrc-xxxx.region.confluent.cloud
func NewRegistryClient(baseURL, apiKey, apiSecret string) (*RegistryClient, error) {
	if baseURL == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("schema registry: baseURL, apiKey, and apiSecret are all required")
	}
	return &RegistryClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[int]string),
	}, nil
}

// GetOrRegisterSchemaID returns the integer schema ID for the given subject
// and schema JSON. If the schema is already registered the existing ID is
// returned. Otherwise the schema is registered and the new ID is returned.
//
// Subject naming follows TopicNameStrategy: "{topic}-value"
// e.g. "device.metrics-value", "device.alerts-value"
//
// Schema compatibility is enforced server-side. If the schema violates the
// configured compatibility mode (BACKWARD by default on Confluent Cloud),
// this call returns an error — catching breaking changes before they reach Kafka.
func (r *RegistryClient) GetOrRegisterSchemaID(subject, schemaJSON string) (int, error) {
	body, err := json.Marshal(map[string]string{"schema": schemaJSON})
	if err != nil {
		return 0, fmt.Errorf("schema registry: marshalling request body: %w", err)
	}

	url := fmt.Sprintf("%s/subjects/%s/versions", r.baseURL, subject)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("schema registry: building POST request for %s: %w", subject, err)
	}
	req.Header.Set("Content-Type", contentType)
	req.SetBasicAuth(r.apiKey, r.apiSecret)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("schema registry: POST /subjects/%s/versions: %w", subject, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("schema registry: reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("schema registry: POST /subjects/%s/versions returned HTTP %d: %s",
			subject, resp.StatusCode, data)
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("schema registry: parsing response JSON: %w", err)
	}

	return result.ID, nil
}

// GetSchemaByID fetches the Avro schema JSON for the given schema ID.
// Results are cached in memory after the first fetch — subsequent calls
// for the same ID make no network request.
func (r *RegistryClient) GetSchemaByID(id int) (string, error) {
	// Fast path: return from cache without acquiring a write lock.
	r.mu.RLock()
	if s, ok := r.cache[id]; ok {
		r.mu.RUnlock()
		return s, nil
	}
	r.mu.RUnlock()

	// Cache miss — fetch from the registry.
	url := fmt.Sprintf("%s/schemas/ids/%d", r.baseURL, id)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("schema registry: building GET request for id %d: %w", id, err)
	}
	req.Header.Set("Content-Type", contentType)
	req.SetBasicAuth(r.apiKey, r.apiSecret)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("schema registry: GET /schemas/ids/%d: %w", id, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("schema registry: reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("schema registry: GET /schemas/ids/%d returned HTTP %d: %s",
			id, resp.StatusCode, data)
	}

	var result struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("schema registry: parsing response JSON: %w", err)
	}

	// Store in cache under write lock.
	r.mu.Lock()
	r.cache[id] = result.Schema
	r.mu.Unlock()

	return result.Schema, nil
}
