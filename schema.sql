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

-- ── Seed Data ────────────────────────────────────────────────
INSERT INTO stocks (ticker, company_name, sector, listing_date, total_shares, last_price, prev_close) VALUES
('BBCA', 'Bank Central Asia Tbk', 'Perbankan', '2000-01-01', 123306376986, 10300, 9785),
('BBRI', 'Bank Rakyat Indonesia (Persero) Tbk', 'Perbankan', '2000-01-01', 163714439506, 4840, 4598),
('BMRI', 'Bank Mandiri (Persero) Tbk', 'Perbankan', '2000-01-01', 92538238792, 7025, 6673),
('BBNI', 'Bank Negara Indonesia (Persero) Tbk', 'Perbankan', '2000-01-01', 18648656458, 5350, 5082),
('BRIS', 'Bank Syariah Indonesia Tbk', 'Perbankan', '2000-01-01', 28007809930, 2870, 2726),
('TLKM', 'Telkom Indonesia (Persero) Tbk', 'Telekomunikasi', '2000-01-01', 99062216600, 3180, 3021),
('EXCL', 'XL Axiata Tbk', 'Telekomunikasi', '2000-01-01', 10687960423, 2310, 2194),
('ISAT', 'Indosat Tbk', 'Telekomunikasi', '2000-01-01', 8062692952, 2810, 2669),
('ASII', 'Astra International Tbk', 'Industri Dasar', '2000-01-01', 40483553140, 4630, 4398),
('ADRO', 'Adaro Energy Indonesia Tbk', 'Pertambangan', '2000-01-01', 31985962000, 2650, 2517),
('PTBA', 'Bukit Asam (Persero) Tbk', 'Pertambangan', '2000-01-01', 11520659250, 3200, 3040),
('PGAS', 'Perusahaan Gas Negara Tbk', 'Energi', '2000-01-01', 24241508196, 1590, 1510),
('MEDC', 'Medco Energi Internasional Tbk', 'Energi', '2000-01-01', 9679447767, 1350, 1282),
('UNVR', 'Unilever Indonesia Tbk', 'Konsumer', '2000-01-01', 38150000000, 2460, 2337),
('ICBP', 'Indofood CBP Sukses Makmur Tbk', 'Konsumer', '2000-01-01', 11661908000, 10725, 10188),
('INDF', 'Indofood Sukses Makmur Tbk', 'Konsumer', '2000-01-01', 8780426500, 6800, 6460),
('HMSP', 'HM Sampoerna Tbk', 'Konsumer', '2000-01-01', 116318076900, 820, 779),
('MYOR', 'Mayora Indah Tbk', 'Konsumer', '2000-01-01', 22358699725, 2640, 2508),
('BSDE', 'Bumi Serpong Damai Tbk', 'Properti', '2000-01-01', 19246696192, 1185, 1125),
('SMRA', 'Summarecon Agung Tbk', 'Properti', '2000-01-01', 14426781680, 615, 584),
('JSMR', 'Jasa Marga (Persero) Tbk', 'Infrastruktur', '2000-01-01', 6800000000, 4400, 4180),
('GOTO', 'GoTo Gojek Tokopedia Tbk', 'Teknologi', '2000-01-01', 1190684447928, 73, 69),
('BUKA', 'Bukalapak.com Tbk', 'Teknologi', '2000-01-01', 104081765731, 132, 125),
('DMMX', 'Digital Mediatama Maxima Tbk', 'Teknologi', '2000-01-01', 4285600000, 710, 674),
('SMGR', 'Semen Indonesia (Persero) Tbk', 'Material', '2000-01-01', 5931520000, 4600, 4370),
('INTP', 'Indocement Tunggal Prakarsa Tbk', 'Material', '2000-01-01', 3681231699, 9275, 8811)
ON CONFLICT (ticker) DO NOTHING;

INSERT INTO traders (id, username, full_name, cash_balance) VALUES
('ffb8a910-8ff8-4bda-aac9-dd556119299c', 'budi', 'Budi Santoso', 500000000),
('2d9a6c90-9c1a-4c22-b91c-8e41a916a4e3', 'siti', 'Siti Rahayu', 250000000),
('e43c8b6b-4b15-4122-b57f-132b4b455c1e', 'agus', 'Agus Wijaya', 1000000000),
('707b6c8a-4d2a-4171-88c9-0a647d7a7b8e', 'dewi', 'Dewi Kusuma', 750000000)
ON CONFLICT (username) DO NOTHING;

INSERT INTO portfolios (trader_id, ticker, qty_lot, avg_price) VALUES
('ffb8a910-8ff8-4bda-aac9-dd556119299c', 'BBCA', 100, 10000),
('ffb8a910-8ff8-4bda-aac9-dd556119299c', 'TLKM', 500, 3100),
('ffb8a910-8ff8-4bda-aac9-dd556119299c', 'GOTO', 5000, 65),
('2d9a6c90-9c1a-4c22-b91c-8e41a916a4e3', 'ASII', 200, 4500),
('2d9a6c90-9c1a-4c22-b91c-8e41a916a4e3', 'BMRI', 150, 6800),
('e43c8b6b-4b15-4122-b57f-132b4b455c1e', 'BBRI', 1000, 4800),
('e43c8b6b-4b15-4122-b57f-132b4b455c1e', 'ICBP', 50, 10500),
('e43c8b6b-4b15-4122-b57f-132b4b455c1e', 'UNVR', 200, 2400),
('707b6c8a-4d2a-4171-88c9-0a647d7a7b8e', 'BBNI', 300, 5000),
('707b6c8a-4d2a-4171-88c9-0a647d7a7b8e', 'EXCL', 1000, 2200),
('707b6c8a-4d2a-4171-88c9-0a647d7a7b8e', 'JSMR', 50, 4000)
ON CONFLICT (trader_id, ticker) DO NOTHING;
