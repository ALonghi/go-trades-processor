package domain

import "strings"

// Entity is a closed set of locations. Include "all" for aggregate queries.
type Entity string

const (
	EntityAll     Entity = "all"
	EntityZurich  Entity = "zurich"
	EntityNewYork Entity = "new_york"
)

func (e Entity) String() string { return string(e) }
func (e Entity) Valid() bool {
	switch e {
	case EntityAll, EntityZurich, EntityNewYork:
		return true
	default:
		return false
	}
}

func ParseEntity(s string) (Entity, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all":
		return EntityAll, true
	case "zurich":
		return EntityZurich, true
	case "new_york":
		return EntityNewYork, true
	default:
		return "", false
	}
}

// InstrumentType, if you want to type trades/holdings too (optional but recommended).
type InstrumentType string

const (
	InstrumentStock  InstrumentType = "stock"
	InstrumentCrypto InstrumentType = "crypto"
)

func (t InstrumentType) Valid() bool {
	return t == InstrumentStock || t == InstrumentCrypto
}
