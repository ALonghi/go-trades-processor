package main

import "time"

// Trade is the JSON schema the backend expects.
type Trade struct {
	TradeID        string    `json:"trade_id"`
	Entity         string    `json:"entity"`          // "zurich" | "new_york"
	InstrumentType string    `json:"instrument_type"` // "stock" | "crypto"
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`        // +buy / -sell
	Price          *float64  `json:"price,omitempty"` // quoted currency (USD here)
	TS             time.Time `json:"ts"`              // RFC3339
}
