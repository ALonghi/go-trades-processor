package holdings

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/trades-aggregator/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct{ DB *pgxpool.Pool }

func New(db *pgxpool.Pool) *Service { return &Service{DB: db} }

func (s *Service) ApplyTrade(ctx context.Context, t models.Trade) error {
	_, err := s.DB.Exec(ctx, `
		WITH ins AS (
		  INSERT INTO trades (trade_id, entity, instrument_type, symbol, quantity, price, ts)
		  VALUES ($1, $2::entity, $3::instrument_type, $4, $5, $6, $7)
		  ON CONFLICT (trade_id) DO NOTHING
		  RETURNING 1
		)
		INSERT INTO holdings (entity, instrument_type, symbol, quantity)
		VALUES ($2::entity, $3::instrument_type, $4, $5)
		ON CONFLICT (entity, instrument_type, symbol)
		DO UPDATE SET quantity = holdings.quantity + EXCLUDED.quantity,
		              updated_at = now();
	`, t.TradeID, t.Entity, t.InstrumentType, t.Symbol, t.Quantity, t.Price, t.TS)
	return err
}

func (s *Service) GetAll(ctx context.Context) ([]models.Holding, error) {
	rows, err := s.DB.Query(ctx, `SELECT entity::text, instrument_type::text, symbol, quantity FROM holdings ORDER BY entity, instrument_type, symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Holding, 0)
	for rows.Next() {
		var h models.Holding
		if err := rows.Scan(&h.Entity, &h.InstrumentType, &h.Symbol, &h.Quantity); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) GetByEntity(ctx context.Context, entity string) ([]models.Holding, error) {
	rows, err := s.DB.Query(ctx, `SELECT entity::text, instrument_type::text, symbol, quantity FROM holdings WHERE entity=$1::entity ORDER BY instrument_type, symbol`, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Holding, 0)
	for rows.Next() {
		var h models.Holding
		if err := rows.Scan(&h.Entity, &h.InstrumentType, &h.Symbol, &h.Quantity); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	if len(out) == 0 {
		return nil, pgx.ErrNoRows
	}
	return out, rows.Err()
}

func (s *Service) GetTrades(ctx context.Context, limit int, entity *string) ([][]any, error) {
	q := `SELECT trade_id::text, entity::text, instrument_type::text, symbol, quantity, price, ts FROM trades`
	var args []any
	if entity != nil {
		q += ` WHERE entity = $1::entity`
		args = append(args, *entity)
	}
	q += ` ORDER BY ts DESC LIMIT ` + fmt.Sprint(limit)
	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([][]any, 0)
	for rows.Next() {
		var tid, ent, itype, sym string
		var qty float64
		var price *float64
		var ts any
		if err := rows.Scan(&tid, &ent, &itype, &sym, &qty, &price, &ts); err != nil {
			return nil, err
		}
		out = append(out, []any{tid, ent, itype, sym, qty, price, ts})
	}
	return out, rows.Err()
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
