package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

var (
	stocks    = []string{"AAPL", "MSFT", "GOOGL", "AMZN", "TSLA", "NVDA", "NFLX"}
	stockBase = map[string]float64{"AAPL": 190, "MSFT": 420, "GOOGL": 145, "AMZN": 180, "TSLA": 220, "NVDA": 800, "NFLX": 550}

	cryptos    = []string{"BTC", "ETH", "SOL", "ADA", "XRP"}
	cryptoBase = map[string]float64{"BTC": 60000, "ETH": 3200, "SOL": 150, "ADA": 0.45, "XRP": 0.6}

	entities   = []string{"zurich", "new_york"}
	instrTypes = []string{"stock", "crypto"}
)

func round(x float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(x*scale) / scale
}

func pick[T any](xs []T) T { return xs[rng.Intn(len(xs))] }

func genTrade() Trade {
	ent := pick(entities)
	itype := pick(instrTypes)

	var (
		sym string
		qty float64
		px  float64
	)

	if itype == "stock" {
		sym = pick(stocks)
		base := stockBase[sym]
		px = round(base*(1+(rng.Float64()-0.5)*0.03), 2) // ±1.5%
		q := float64(rng.Intn(50) + 1)
		if rng.Intn(2) == 0 {
			q = -q
		}
		qty = q
	} else {
		sym = pick(cryptos)
		base := cryptoBase[sym]
		px = round(base*(1+(rng.Float64()-0.5)*0.05), 2) // ±2.5%
		q := 0.001 + rng.Float64()*(0.5-0.001)
		q = round(q, 4)
		if rng.Intn(2) == 0 {
			q = -q
		}
		qty = q
	}

	price := px
	return Trade{
		TradeID:        uuid.NewString(),
		Entity:         ent,
		InstrumentType: itype,
		Symbol:         sym,
		Quantity:       qty,
		Price:          &price,
		TS:             time.Now().UTC(),
	}
}
