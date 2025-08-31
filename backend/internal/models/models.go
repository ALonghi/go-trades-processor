package models

import "time"

type Trade struct {
	TradeID        string    `json:"trade_id"`
	Entity         string    `json:"entity"`
	InstrumentType string    `json:"instrument_type"`
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	Price          *float64  `json:"price,omitempty"`
	TS             time.Time `json:"ts"`
}

type Holding struct {
	Entity         string  `json:"entity"`
	InstrumentType string  `json:"instrument_type"`
	Symbol         string  `json:"symbol"`
	Quantity       float64 `json:"quantity"`
}
