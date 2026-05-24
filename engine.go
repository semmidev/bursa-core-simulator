package main

import (
	"database/sql"
	"fmt"
	"time"
)

type MatchResult struct {
	Trades    []Trade
	TotalFill int64
}

type Engine struct{ R *Repo }

func NewEngine(r *Repo) *Engine { return &Engine{R: r} }

func (e *Engine) SubmitOrder(traderID, ticker string, side OrderSide, orderType OrderType, price, qtyLot int64) (*Order, *MatchResult, error) {
	if qtyLot <= 0 {
		return nil, nil, fmt.Errorf("qty_lot must be positive")
	}

	stock, err := e.R.GetStock(ticker)
	if err != nil {
		return nil, nil, fmt.Errorf("get stock: %w", err)
	}
	if stock == nil {
		return nil, nil, fmt.Errorf("stock %s not found", ticker)
	}
	if stock.IsHalted {
		return nil, nil, fmt.Errorf("trading for %s is currently HALTED", ticker)
	}

	if orderType == TypeLimit {
		refPrice := stock.PrevClose
		if refPrice <= 0 {
			refPrice = stock.LastPrice
		}

		var maxPercent float64
		if refPrice >= 50 && refPrice <= 200 {
			maxPercent = 0.35
		} else if refPrice > 200 && refPrice <= 5000 {
			maxPercent = 0.25
		} else {
			maxPercent = 0.20
		}

		araPrice := refPrice + int64(float64(refPrice)*maxPercent)
		arbPrice := refPrice - int64(float64(refPrice)*maxPercent)

		if price > araPrice {
			return nil, nil, fmt.Errorf("Order ditolak (Auto Reject): Harga %d melebihi batas ARA (%d)", price, araPrice)
		}
		if price < arbPrice {
			return nil, nil, fmt.Errorf("Order ditolak (Auto Reject): Harga %d di bawah batas ARB (%d)", price, arbPrice)
		}
	}

	tx, err := e.R.DB.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	order := &Order{
		TraderID:  traderID,
		Ticker:    ticker,
		Side:      side,
		OrderType: orderType,
		Price:     price,
		QtyLot:    qtyLot,
	}

	var reservePrice int64

	if side == SideBuy {
		if orderType == TypeMarket {
			reservePrice = stock.LastPrice + stock.LastPrice/10
			if reservePrice == 0 {
				reservePrice = 99_999_999
			}
			order.Price = 0
		} else {
			reservePrice = price
		}
		cost := reservePrice * qtyLot * 100
		if err = e.R.UpdateTraderCash(tx, traderID, -cost); err != nil {
			return nil, nil, fmt.Errorf("insufficient cash: need Rp %s", fmtRupiah(cost))
		}
	} else {
		if err = e.R.ReducePortfolio(tx, traderID, ticker, qtyLot); err != nil {
			return nil, nil, fmt.Errorf("insufficient shares: %w", err)
		}
		if orderType == TypeMarket {
			order.Price = 0
		}
	}

	if err = e.R.InsertOrder(tx, order); err != nil {
		return nil, nil, fmt.Errorf("insert order: %w", err)
	}

	result, err := e.matchOrder(tx, order, stock)
	if err != nil {
		return nil, nil, fmt.Errorf("matching: %w", err)
	}

	// Refund unused reservation
	if side == SideBuy {
		if orderType == TypeLimit {
			var actualCost int64
			for _, t := range result.Trades {
				actualCost += t.Price * t.QtyLot * 100
			}
			refund := (order.Price * order.QtyLot * 100) - actualCost
			if refund > 0 {
				if err = e.R.UpdateTraderCash(tx, traderID, refund); err != nil {
					return nil, nil, err
				}
			}
		} else {
			var actualCost int64
			for _, t := range result.Trades {
				actualCost += t.Price * t.QtyLot * 100
			}
			rp := stock.LastPrice + stock.LastPrice/10
			if rp == 0 {
				rp = 99_999_999
			}
			refund := rp*qtyLot*100 - actualCost
			if refund > 0 {
				if err = e.R.UpdateTraderCash(tx, traderID, refund); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	if side == SideSell && orderType == TypeMarket {
		unfilled := order.QtyLot - result.TotalFill
		if unfilled > 0 {
			if err = e.R.UpsertPortfolioAdd(tx, traderID, ticker, unfilled, 0); err != nil {
				return nil, nil, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, err
	}
	return order, result, nil
}

func (e *Engine) matchOrder(tx *sql.Tx, incoming *Order, stock *Stock) (*MatchResult, error) {
	result := &MatchResult{}

	counterOrders, err := e.R.GetOpenOrdersForMatching(tx, incoming.Ticker, incoming.Side)
	if err != nil {
		return nil, err
	}

	remaining := incoming.QtyLot

	for _, counter := range counterOrders {
		if remaining == 0 {
			break
		}
		if incoming.OrderType == TypeLimit {
			if incoming.Side == SideBuy && incoming.Price < counter.Price {
				break
			}
			if incoming.Side == SideSell && incoming.Price > counter.Price {
				break
			}
		}

		tradePrice := counter.Price
		if counter.OrderType == TypeMarket {
			tradePrice = incoming.Price
		}

		fillQty := min64(remaining, counter.RemainingLot())
		if fillQty == 0 {
			continue
		}

		var buyID, sellID, buyTrader, sellTrader string
		if incoming.Side == SideBuy {
			buyID, sellID = incoming.ID, counter.ID
			buyTrader, sellTrader = incoming.TraderID, counter.TraderID
		} else {
			buyID, sellID = counter.ID, incoming.ID
			buyTrader, sellTrader = counter.TraderID, incoming.TraderID
		}

		trade := &Trade{
			Ticker:      incoming.Ticker,
			BuyOrderID:  buyID,
			SellOrderID: sellID,
			Price:       tradePrice,
			QtyLot:      fillQty,
		}
		if err := e.R.InsertTrade(tx, trade); err != nil {
			return nil, err
		}
		if err := e.R.UpdateOrderFilled(tx, incoming.ID, fillQty); err != nil {
			return nil, err
		}
		if err := e.R.UpdateOrderFilled(tx, counter.ID, fillQty); err != nil {
			return nil, err
		}
		if err := e.R.UpsertPortfolioAdd(tx, buyTrader, incoming.Ticker, fillQty, tradePrice); err != nil {
			return nil, err
		}
		proceeds := tradePrice * fillQty * 100
		if err := e.R.UpdateTraderCash(tx, sellTrader, proceeds); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`
			UPDATE stocks SET
			    last_price = $2,
			    high_price = GREATEST(high_price, $2),
			    low_price  = LEAST(low_price, $2),
			    volume_lot = volume_lot + $3,
			    value_idr  = value_idr  + ($2 * $3 * 100),
			    open_price = CASE WHEN open_price = 0 THEN $2 ELSE open_price END,
			    updated_at = NOW()
			WHERE ticker=$1`, incoming.Ticker, tradePrice, fillQty); err != nil {
			return nil, err
		}
		if err := e.R.UpsertCandle(incoming.Ticker, time.Now(), tradePrice, fillQty); err != nil {
			return nil, err
		}

		result.Trades = append(result.Trades, *trade)
		result.TotalFill += fillQty
		remaining -= fillQty
	}

	return result, nil
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func fmtRupiah(v int64) string {
	return fmt.Sprintf("%d", v)
}
