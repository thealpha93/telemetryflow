package models

import "time"

// Device represents an entry in the device registry stored in PostgreSQL.
//
// The registry is the source of truth for device metadata (name, location,
// firmware). The stream processor enriches each metric with this data
// before computing aggregations (via the enricher cache — see processor/enricher.go).
type Device struct {
	DeviceID        string
	Name            string
	LocationID      string
	FirmwareVersion string
	RegisteredAt    time.Time
	Active          bool
}
