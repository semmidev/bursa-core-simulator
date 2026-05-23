-- ============================================================
--  BEI EXCHANGE SIMULATOR — DATABASE SCHEMA
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Stocks ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS stocks (
    ticker          VARCHAR(10) PRIMARY KEY,
    company_name    VARCHAR(255) NOT NULL,
    sector          VARCHAR(100) NOT NULL,
    listing_date    DATE NOT NULL,
    total_shares    BIGINT NOT NULL,
    last_price      BIGINT NOT NULL DEFAULT 0,
    prev_close      BIGINT NOT NULL DEFAULT 0,
    open_price      BIGINT NOT NULL DEFAULT 0,
    high_price      BIGINT NOT NULL DEFAULT 0,
    low_price       BIGINT NOT NULL DEFAULT 999999999,
    volume_lot      BIGINT NOT NULL DEFAULT 0,
    value_idr       BIGINT NOT NULL DEFAULT 0,
    is_halted       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Traders ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS traders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(50) UNIQUE NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    cash_balance    BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Portfolio ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS portfolios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trader_id       UUID NOT NULL REFERENCES traders(id) ON DELETE CASCADE,
    ticker          VARCHAR(10) NOT NULL REFERENCES stocks(ticker),
    qty_lot         BIGINT NOT NULL DEFAULT 0,
    avg_price       BIGINT NOT NULL DEFAULT 0,
    UNIQUE (trader_id, ticker)
);

-- ── Orders ───────────────────────────────────────────────────
DO $$ BEGIN
  CREATE TYPE order_side AS ENUM ('BUY', 'SELL');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  CREATE TYPE order_type AS ENUM ('LIMIT', 'MARKET');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
  CREATE TYPE order_status AS ENUM ('OPEN', 'PARTIAL', 'FILLED', 'CANCELLED', 'REJECTED');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS orders (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trader_id       UUID NOT NULL REFERENCES traders(id),
    ticker          VARCHAR(10) NOT NULL REFERENCES stocks(ticker),
    side            order_side NOT NULL,
    order_type      order_type NOT NULL,
    price           BIGINT NOT NULL DEFAULT 0,
    qty_lot         BIGINT NOT NULL,
    filled_lot      BIGINT NOT NULL DEFAULT 0,
    status          order_status NOT NULL DEFAULT 'OPEN',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_ticker_side_status
    ON orders (ticker, side, status, price, created_at);

-- ── Trades ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS trades (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker          VARCHAR(10) NOT NULL REFERENCES stocks(ticker),
    buy_order_id    UUID NOT NULL REFERENCES orders(id),
    sell_order_id   UUID NOT NULL REFERENCES orders(id),
    price           BIGINT NOT NULL,
    qty_lot         BIGINT NOT NULL,
    traded_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_ticker_time ON trades (ticker, traded_at DESC);

-- ── Price History ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS price_history (
    id              BIGSERIAL PRIMARY KEY,
    ticker          VARCHAR(10) NOT NULL REFERENCES stocks(ticker),
    period_start    TIMESTAMPTZ NOT NULL,
    open_price      BIGINT NOT NULL,
    high_price      BIGINT NOT NULL,
    low_price       BIGINT NOT NULL,
    close_price     BIGINT NOT NULL,
    volume_lot      BIGINT NOT NULL DEFAULT 0,
    UNIQUE (ticker, period_start)
);

-- ── Triggers ─────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION touch_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;

DROP TRIGGER IF EXISTS stocks_updated_at ON stocks;
CREATE TRIGGER stocks_updated_at
    BEFORE UPDATE ON stocks
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

DROP TRIGGER IF EXISTS orders_updated_at ON orders;
CREATE TRIGGER orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
