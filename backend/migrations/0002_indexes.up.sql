CREATE INDEX IF NOT EXISTS idx_trades_entity ON trades(entity);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
CREATE INDEX IF NOT EXISTS idx_trades_ts ON trades(ts);
