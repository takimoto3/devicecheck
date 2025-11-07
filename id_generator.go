package devicecheck

import "github.com/google/uuid"

type Generator interface {
	Generate() string
}

type UUIDGenerator struct{}

func (g UUIDGenerator) Generate() string {
	return uuid.NewString()
}
