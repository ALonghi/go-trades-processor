CREATE TYPE entity AS ENUM ('zurich', 'new_york');
CREATE TYPE instrument_type AS ENUM ('stock', 'crypto');

CREATE TABLE IF NOT EXISTS trades (
  id BIGSERIAL PRIMARY KEY,
  trade_id UUID UNIQUE NOT NULL,
  entity entity NOT NULL,
  instrument_type instrument_type NOT NULL,
  symbol TEXT NOT NULL,
  quantity NUMERIC(20,8) NOT NULL,
  price NUMERIC(20,8),
  ts TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS holdings (
  entity entity NOT NULL,
  instrument_type instrument_type NOT NULL,
  symbol TEXT NOT NULL,
  quantity NUMERIC(20,8) NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (entity, instrument_type, symbol)
);
