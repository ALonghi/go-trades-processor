package cache

import "github.com/example/trades-aggregator/internal/domain"

// HoldingsKey: “all” means aggregate over all entities.
type HoldingsKey struct {
	Entity domain.Entity
}

func HoldingsAll() HoldingsKey {
	return HoldingsKey{Entity: domain.EntityAll}
}
func HoldingsByEntity(e domain.Entity) HoldingsKey {
	return HoldingsKey{Entity: e}
}

// TradesKey: identify a trades query. Entity=all means no filter.
type TradesKey struct {
	Entity domain.Entity
	Limit  uint16
}

func Trades(e domain.Entity, limit int) TradesKey {
	if limit < 0 {
		limit = 0
	}
	if limit > 65535 {
		limit = 65535
	}
	return TradesKey{Entity: e, Limit: uint16(limit)}
}
