package main

import (
	"database/sql"
	"fmt"
	"time"
)

type Repo struct{ DB *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{DB: db} }

// ── Stocks ───────────────────────────────────────────────────

func (r *Repo) GetAllStocks() ([]Stock, error) {
	rows, err := r.DB.Query(`
		SELECT ticker, company_name, sector, listing_date, total_shares,
		       last_price, prev_close, open_price, high_price, low_price,
		       volume_lot, value_idr, is_halted, created_at, updated_at
		FROM stocks ORDER BY ticker`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStocks(rows)
}

func (r *Repo) GetStock(ticker string) (*Stock, error) {
	row := r.DB.QueryRow(`
		SELECT ticker, company_name, sector, listing_date, total_shares,
		       last_price, prev_close, open_price, high_price, low_price,
		       volume_lot, value_idr, is_halted, created_at, updated_at
		FROM stocks WHERE ticker=$1`, ticker)
	return scanStock(row)
}

func (r *Repo) UpsertStock(s Stock) error {
	_, err := r.DB.Exec(`
		INSERT INTO stocks (ticker, company_name, sector, listing_date, total_shares,
		                    last_price, prev_close, open_price, high_price, low_price)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$6,$6,$6)
		ON CONFLICT (ticker) DO UPDATE SET
		    company_name = EXCLUDED.company_name,
		    sector       = EXCLUDED.sector,
		    updated_at   = NOW()`,
		s.Ticker, s.CompanyName, s.Sector, s.ListingDate.Format("2006-01-02"),
		s.TotalShares, s.LastPrice, s.PrevClose)
	return err
}

func (r *Repo) HaltStock(ticker string, halted bool) error {
	_, err := r.DB.Exec(`UPDATE stocks SET is_halted=$2, updated_at=NOW() WHERE ticker=$1`, ticker, halted)
	return err
}

func (r *Repo) DeleteStock(ticker string) error {
	_, err := r.DB.Exec(`DELETE FROM stocks WHERE ticker=$1`, ticker)
	return err
}

// ── Traders ──────────────────────────────────────────────────

func (r *Repo) CreateTrader(t *Trader) error {
	return r.DB.QueryRow(`
		INSERT INTO traders (username, full_name, cash_balance)
		VALUES ($1, $2, $3) RETURNING id, created_at`,
		t.Username, t.FullName, t.CashBalance,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *Repo) GetTraderByUsername(username string) (*Trader, error) {
	t := &Trader{}
	err := r.DB.QueryRow(`
		SELECT id, username, full_name, cash_balance, created_at
		FROM traders WHERE username=$1`, username,
	).Scan(&t.ID, &t.Username, &t.FullName, &t.CashBalance, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *Repo) GetTraderByID(id string) (*Trader, error) {
	t := &Trader{}
	err := r.DB.QueryRow(`
		SELECT id, username, full_name, cash_balance, created_at
		FROM traders WHERE id=$1`, id,
	).Scan(&t.ID, &t.Username, &t.FullName, &t.CashBalance, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *Repo) GetAllTraders() ([]Trader, error) {
	rows, err := r.DB.Query(`SELECT id, username, full_name, cash_balance, created_at FROM traders ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var traders []Trader
	for rows.Next() {
		var t Trader
		if err := rows.Scan(&t.ID, &t.Username, &t.FullName, &t.CashBalance, &t.CreatedAt); err != nil {
			return nil, err
		}
		traders = append(traders, t)
	}
	return traders, rows.Err()
}

func (r *Repo) UpdateTraderCash(tx *sql.Tx, traderID string, delta int64) error {
	res, err := tx.Exec(`UPDATE traders SET cash_balance = cash_balance + $2 WHERE id=$1`, traderID, delta)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("trader %s not found", traderID)
	}
	var bal int64
	if err := tx.QueryRow(`SELECT cash_balance FROM traders WHERE id=$1`, traderID).Scan(&bal); err != nil {
		return err
	}
	if bal < 0 {
		return fmt.Errorf("insufficient cash balance")
	}
	return nil
}

// ── Portfolio ─────────────────────────────────────────────────

func (r *Repo) GetPortfolio(traderID string) ([]Portfolio, error) {
	rows, err := r.DB.Query(`
		SELECT id, trader_id, ticker, qty_lot, avg_price
		FROM portfolios WHERE trader_id=$1 AND qty_lot > 0 ORDER BY ticker`, traderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Portfolio
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.ID, &p.TraderID, &p.Ticker, &p.QtyLot, &p.AvgPrice); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *Repo) UpsertPortfolioAdd(tx *sql.Tx, traderID, ticker string, qtyLot, price int64) error {
	_, err := tx.Exec(`
		INSERT INTO portfolios (trader_id, ticker, qty_lot, avg_price)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (trader_id, ticker) DO UPDATE SET
		    avg_price = ((portfolios.avg_price * portfolios.qty_lot) + ($4 * $3))
		              / (portfolios.qty_lot + $3),
		    qty_lot   = portfolios.qty_lot + $3`,
		traderID, ticker, qtyLot, price)
	return err
}

func (r *Repo) ReducePortfolio(tx *sql.Tx, traderID, ticker string, qtyLot int64) error {
	res, err := tx.Exec(`
		UPDATE portfolios SET qty_lot = qty_lot - $3
		WHERE trader_id=$1 AND ticker=$2 AND qty_lot >= $3`,
		traderID, ticker, qtyLot)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("insufficient stock holdings for %s", ticker)
	}
	return nil
}

// ── Orders ────────────────────────────────────────────────────

func (r *Repo) InsertOrder(tx *sql.Tx, o *Order) error {
	return tx.QueryRow(`
		INSERT INTO orders (trader_id, ticker, side, order_type, price, qty_lot, status)
		VALUES ($1,$2,$3,$4,$5,$6,'OPEN')
		RETURNING id, created_at`,
		o.TraderID, o.Ticker, string(o.Side), string(o.OrderType), o.Price, o.QtyLot,
	).Scan(&o.ID, &o.CreatedAt)
}

func (r *Repo) GetOpenOrdersForMatching(tx *sql.Tx, ticker string, side OrderSide) ([]Order, error) {
	var query string
	if side == SideBuy {
		query = `
			SELECT id, trader_id, ticker, side, order_type, price, qty_lot, filled_lot, status, created_at, updated_at
			FROM orders
			WHERE ticker=$1 AND side='SELL' AND status IN ('OPEN','PARTIAL')
			ORDER BY price ASC, created_at ASC`
	} else {
		query = `
			SELECT id, trader_id, ticker, side, order_type, price, qty_lot, filled_lot, status, created_at, updated_at
			FROM orders
			WHERE ticker=$1 AND side='BUY' AND status IN ('OPEN','PARTIAL')
			ORDER BY price DESC, created_at ASC`
	}
	rows, err := tx.Query(query, ticker)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrderRows(rows)
}

func (r *Repo) UpdateOrderFilled(tx *sql.Tx, orderID string, filledDelta int64) error {
	_, err := tx.Exec(`
		UPDATE orders SET
		    filled_lot = filled_lot + $2,
		    status     = CASE
		                   WHEN filled_lot + $2 >= qty_lot THEN 'FILLED'::order_status
		                   ELSE 'PARTIAL'::order_status
		                 END,
		    updated_at = NOW()
		WHERE id=$1`, orderID, filledDelta)
	return err
}

func (r *Repo) CancelOrder(orderID, traderID string) (*Order, error) {
	o := &Order{}
	err := r.DB.QueryRow(`
		UPDATE orders SET status='CANCELLED', updated_at=NOW()
		WHERE id=$1 AND trader_id=$2 AND status IN ('OPEN','PARTIAL')
		RETURNING id, trader_id, ticker, side, order_type, price, qty_lot, filled_lot, status, created_at, updated_at`,
		orderID, traderID,
	).Scan(&o.ID, &o.TraderID, &o.Ticker, &o.Side, &o.OrderType,
		&o.Price, &o.QtyLot, &o.FilledLot, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found or already completed")
	}
	return o, err
}

func (r *Repo) GetTraderOrders(traderID string, limit int) ([]Order, error) {
	rows, err := r.DB.Query(`
		SELECT id, trader_id, ticker, side, order_type, price, qty_lot, filled_lot, status, created_at, updated_at
		FROM orders WHERE trader_id=$1
		ORDER BY created_at DESC LIMIT $2`, traderID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrderRows(rows)
}

// ── Trades ────────────────────────────────────────────────────

func (r *Repo) InsertTrade(tx *sql.Tx, t *Trade) error {
	return tx.QueryRow(`
		INSERT INTO trades (ticker, buy_order_id, sell_order_id, price, qty_lot)
		VALUES ($1,$2,$3,$4,$5) RETURNING id, traded_at`,
		t.Ticker, t.BuyOrderID, t.SellOrderID, t.Price, t.QtyLot,
	).Scan(&t.ID, &t.TradedAt)
}

func (r *Repo) GetRecentTrades(ticker string, limit int) ([]Trade, error) {
	rows, err := r.DB.Query(`
		SELECT id, ticker, buy_order_id, sell_order_id, price, qty_lot, traded_at
		FROM trades WHERE ticker=$1
		ORDER BY traded_at DESC LIMIT $2`, ticker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trades []Trade
	for rows.Next() {
		var t Trade
		if err := rows.Scan(&t.ID, &t.Ticker, &t.BuyOrderID, &t.SellOrderID, &t.Price, &t.QtyLot, &t.TradedAt); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

// ── Order Book ────────────────────────────────────────────────

func (r *Repo) GetOrderBook(ticker string, depth int) (*OrderBook, error) {
	ob := &OrderBook{Ticker: ticker}

	bidRows, err := r.DB.Query(`
		SELECT price, SUM(qty_lot - filled_lot) AS total_lot, COUNT(*) AS order_count
		FROM orders
		WHERE ticker=$1 AND side='BUY' AND status IN ('OPEN','PARTIAL')
		GROUP BY price ORDER BY price DESC LIMIT $2`, ticker, depth)
	if err != nil {
		return nil, err
	}
	defer bidRows.Close()
	for bidRows.Next() {
		var b BookLevel
		if err := bidRows.Scan(&b.Price, &b.TotalLot, &b.Orders); err != nil {
			return nil, err
		}
		ob.Bids = append(ob.Bids, b)
	}

	askRows, err := r.DB.Query(`
		SELECT price, SUM(qty_lot - filled_lot) AS total_lot, COUNT(*) AS order_count
		FROM orders
		WHERE ticker=$1 AND side='SELL' AND status IN ('OPEN','PARTIAL')
		GROUP BY price ORDER BY price ASC LIMIT $2`, ticker, depth)
	if err != nil {
		return nil, err
	}
	defer askRows.Close()
	for askRows.Next() {
		var a BookLevel
		if err := askRows.Scan(&a.Price, &a.TotalLot, &a.Orders); err != nil {
			return nil, err
		}
		ob.Asks = append(ob.Asks, a)
	}

	return ob, nil
}

// ── Candles ───────────────────────────────────────────────────

func (r *Repo) UpsertCandle(ticker string, t time.Time, price, qty int64) error {
	period := t.Truncate(time.Minute)
	_, err := r.DB.Exec(`
		INSERT INTO price_history (ticker, period_start, open_price, high_price, low_price, close_price, volume_lot)
		VALUES ($1, $2, $3, $3, $3, $3, $4)
		ON CONFLICT (ticker, period_start) DO UPDATE SET
		    high_price  = GREATEST(price_history.high_price, $3),
		    low_price   = LEAST(price_history.low_price, $3),
		    close_price = $3,
		    volume_lot  = price_history.volume_lot + $4`,
		ticker, period, price, qty)
	return err
}

func (r *Repo) GetCandles(ticker string, limit int) ([]Candle, error) {
	rows, err := r.DB.Query(`
		SELECT period_start, open_price, high_price, low_price, close_price, volume_lot
		FROM price_history WHERE ticker=$1
		ORDER BY period_start DESC LIMIT $2`, ticker, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var candles []Candle
	for rows.Next() {
		var c Candle
		if err := rows.Scan(&c.PeriodStart, &c.Open, &c.High, &c.Low, &c.Close, &c.VolumeLot); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

// ── Refund helpers ────────────────────────────────────────────

func (r *Repo) RefundCash(traderID string, amount int64) error {
	_, err := r.DB.Exec(`UPDATE traders SET cash_balance=cash_balance+$1 WHERE id=$2`, amount, traderID)
	return err
}

func (r *Repo) RefundShares(traderID, ticker string, qty int64) error {
	_, err := r.DB.Exec(`
		INSERT INTO portfolios (trader_id, ticker, qty_lot, avg_price)
		VALUES ($1,$2,$3,0)
		ON CONFLICT (trader_id, ticker) DO UPDATE SET qty_lot=portfolios.qty_lot+$3`,
		traderID, ticker, qty)
	return err
}

// ── Scan helpers ─────────────────────────────────────────────

func scanStock(row *sql.Row) (*Stock, error) {
	s := &Stock{}
	err := row.Scan(
		&s.Ticker, &s.CompanyName, &s.Sector, &s.ListingDate, &s.TotalShares,
		&s.LastPrice, &s.PrevClose, &s.OpenPrice, &s.HighPrice, &s.LowPrice,
		&s.VolumeLot, &s.ValueIDR, &s.IsHalted, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func scanStocks(rows *sql.Rows) ([]Stock, error) {
	var stocks []Stock
	for rows.Next() {
		s := Stock{}
		if err := rows.Scan(
			&s.Ticker, &s.CompanyName, &s.Sector, &s.ListingDate, &s.TotalShares,
			&s.LastPrice, &s.PrevClose, &s.OpenPrice, &s.HighPrice, &s.LowPrice,
			&s.VolumeLot, &s.ValueIDR, &s.IsHalted, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		stocks = append(stocks, s)
	}
	return stocks, rows.Err()
}

func scanOrderRows(rows *sql.Rows) ([]Order, error) {
	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.TraderID, &o.Ticker, &o.Side, &o.OrderType,
			&o.Price, &o.QtyLot, &o.FilledLot, &o.Status, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
