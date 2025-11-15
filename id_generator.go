// Package devicecheck provides a client for the Apple DeviceCheck API.
package devicecheck

import "github.com/google/uuid"

// Generator defines an interface for generating unique identifiers.
type Generator interface {
	Generate() string
}

// UUIDGenerator is an implementation of the Generator interface that generates UUIDs.
type UUIDGenerator struct{}

// Generate creates a new UUID string.
func (g UUIDGenerator) Generate() string {
	return uuid.NewString()
}
