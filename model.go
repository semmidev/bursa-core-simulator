package main

import "time"

type OrderSide string
type OrderType string
type OrderStatus string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"

	TypeLimit  OrderType = "LIMIT"
	TypeMarket OrderType = "MARKET"

	StatusOpen      OrderStatus = "OPEN"
	StatusPartial   OrderStatus = "PARTIAL"
	StatusFilled    OrderStatus = "FILLED"
	StatusCancelled OrderStatus = "CANCELLED"
	StatusRejected  OrderStatus = "REJECTED"
)

type Stock struct {
	Ticker      string
	CompanyName string
	Sector      string
	ListingDate time.Time
	TotalShares int64
	LastPrice   int64
	PrevClose   int64
	OpenPrice   int64
	HighPrice   int64
	LowPrice    int64
	VolumeLot   int64
	ValueIDR    int64
	IsHalted    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Stock) ChangePercent() float64 {
	if s.PrevClose == 0 {
		return 0
	}
	return float64(s.LastPrice-s.PrevClose) / float64(s.PrevClose) * 100
}

func (s *Stock) Change() int64 { return s.LastPrice - s.PrevClose }

type Trader struct {
	ID          string
	Username    string
	FullName    string
	CashBalance int64
	CreatedAt   time.Time
}

type Portfolio struct {
	ID       string
	TraderID string
	Ticker   string
	QtyLot   int64
	AvgPrice int64
}

type Order struct {
	ID        string
	TraderID  string
	Ticker    string
	Side      OrderSide
	OrderType OrderType
	Price     int64
	QtyLot    int64
	FilledLot int64
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (o *Order) RemainingLot() int64 { return o.QtyLot - o.FilledLot }

type Trade struct {
	ID          string
	Ticker      string
	BuyOrderID  string
	SellOrderID string
	Price       int64
	QtyLot      int64
	TradedAt    time.Time
}

type BookLevel struct {
	Price    int64
	TotalLot int64
	Orders   int
}

type OrderBook struct {
	Ticker string
	Bids   []BookLevel
	Asks   []BookLevel
}

type Candle struct {
	PeriodStart time.Time
	Open        int64
	High        int64
	Low         int64
	Close       int64
	VolumeLot   int64
}
