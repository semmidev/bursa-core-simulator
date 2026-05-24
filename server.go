package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ── WebSocket Hub ─────────────────────────────────────────────

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
		case conn := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(event string, data interface{}) {
	payload := map[string]interface{}{"event": event, "data": data}
	b, _ := json.Marshal(payload)
	select {
	case h.broadcast <- b:
	default:
	}
}

// ── Server ────────────────────────────────────────────────────

type Server struct {
	repo     *Repo
	engine   *Engine
	hub      *Hub
	upgrader websocket.Upgrader
	tmpl     *template.Template
}

func NewServer(r *Repo, e *Engine) *Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Parse embedded templates
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"fmtRupiah":  FmtRupiah,
		"fmtNumber":  FmtNumber,
		"fmtBillions": FmtBillions,
		"fmtPercent": FmtPercent,
	}).ParseGlob("templates/*.html"))

	hub := NewHub()
	go hub.Run()

	srv := &Server{
		repo:     r,
		engine:   e,
		hub:      hub,
		upgrader: upgrader,
		tmpl:     tmpl,
	}

	// Start auto-broadcaster
	go srv.ticker()

	return srv
}

func (s *Server) ticker() {
	t := time.NewTicker(2 * time.Second)
	for range t.C {
		stocks, err := s.repo.GetAllStocks()
		if err == nil {
			s.hub.Broadcast("stocks", stocks)
		}
	}
}

func (s *Server) Run() {
	addr := envOr("HTTP_ADDR", ":8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/stocks", s.handleStocks)
	mux.HandleFunc("/api/orderbook", s.handleOrderBook)
	mux.HandleFunc("/api/trades", s.handleTrades)
	mux.HandleFunc("/api/portfolio", s.handlePortfolio)
	mux.HandleFunc("/api/orders", s.handleOrders)
	mux.HandleFunc("/api/traders", s.handleTraders)
	mux.HandleFunc("/api/order/submit", s.handleSubmitOrder)
	mux.HandleFunc("/api/order/cancel", s.handleCancelOrder)
	mux.HandleFunc("/api/seed", s.handleSeed)

	fmt.Fprintf(os.Stdout, "\n  ██████╗ ███████╗██╗    ███████╗██╗  ██╗ ██████╗██╗  ██╗ █████╗ ███╗   ██╗ ██████╗ ███████╗\n")
	fmt.Fprintf(os.Stdout, "  ██╔══██╗██╔════╝██║    ██╔════╝╚██╗██╔╝██╔════╝██║  ██║██╔══██╗████╗  ██║██╔════╝ ██╔════╝\n")
	fmt.Fprintf(os.Stdout, "  ██████╔╝█████╗  ██║    █████╗   ╚███╔╝ ██║     ███████║███████║██╔██╗ ██║██║  ███╗█████╗\n")
	fmt.Fprintf(os.Stdout, "  ██╔══██╗██╔══╝  ██║    ██╔══╝   ██╔██╗ ██║     ██╔══██║██╔══██║██║╚██╗██║██║   ██║██╔══╝\n")
	fmt.Fprintf(os.Stdout, "  ██████╔╝███████╗██║    ███████╗██╔╝ ██╗╚██████╗██║  ██║██║  ██║██║ ╚████║╚██████╔╝███████╗\n")
	fmt.Fprintf(os.Stdout, "  ╚═════╝ ╚══════╝╚═╝    ╚══════╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚══════╝\n\n")
	fmt.Fprintf(os.Stdout, "  ⚡ BEI Exchange Simulator (Web)\n")
	fmt.Fprintf(os.Stdout, "  ─────────────────────────────────\n")
	fmt.Fprintf(os.Stdout, "  http://localhost%s\n\n", addr)

	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── Handlers ──────────────────────────────────────────────────

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	stocks, _ := s.repo.GetAllStocks()
	traders, _ := s.repo.GetAllTraders()
	data := map[string]interface{}{
		"Stocks":  stocks,
		"Traders": traders,
	}
	if err := s.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.register <- conn
	defer func() {
		s.hub.unregister <- conn
		conn.Close()
	}()
	// Keep alive — read and discard
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (s *Server) handleStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := s.repo.GetAllStocks()
	jsonResp(w, stocks, err)
}

func (s *Server) handleOrderBook(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		http.Error(w, "ticker required", 400)
		return
	}
	ob, err := s.repo.GetOrderBook(ticker, 10)
	jsonResp(w, ob, err)
}

func (s *Server) handleTrades(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		http.Error(w, "ticker required", 400)
		return
	}
	trades, err := s.repo.GetRecentTrades(ticker, 20)
	jsonResp(w, trades, err)
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	traderID := r.URL.Query().Get("trader_id")
	if traderID == "" {
		http.Error(w, "trader_id required", 400)
		return
	}
	items, err := s.repo.GetPortfolio(traderID)
	if err != nil {
		jsonResp(w, nil, err)
		return
	}
	allStocks, _ := s.repo.GetAllStocks()
	sm := make(map[string]Stock)
	for _, st := range allStocks {
		sm[st.Ticker] = st
	}
	jsonResp(w, map[string]interface{}{"items": items, "stocks": sm}, nil)
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	traderID := r.URL.Query().Get("trader_id")
	if traderID == "" {
		http.Error(w, "trader_id required", 400)
		return
	}
	orders, err := s.repo.GetTraderOrders(traderID, 50)
	jsonResp(w, orders, err)
}

func (s *Server) handleTraders(w http.ResponseWriter, r *http.Request) {
	traders, err := s.repo.GetAllTraders()
	jsonResp(w, traders, err)
}

type SubmitOrderReq struct {
	TraderID  string `json:"trader_id"`
	Ticker    string `json:"ticker"`
	Side      string `json:"side"`
	OrderType string `json:"order_type"`
	Price     int64  `json:"price"`
	QtyLot    int64  `json:"qty_lot"`
}

func (s *Server) handleSubmitOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req SubmitOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	order, result, err := s.engine.SubmitOrder(
		req.TraderID, req.Ticker,
		OrderSide(req.Side), OrderType(req.OrderType),
		req.Price, req.QtyLot,
	)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}

	// Broadcast updated stocks & trades
	if stocks, err2 := s.repo.GetAllStocks(); err2 == nil {
		s.hub.Broadcast("stocks", stocks)
	}
	if trades, err2 := s.repo.GetRecentTrades(req.Ticker, 20); err2 == nil {
		s.hub.Broadcast("trades:"+req.Ticker, trades)
	}
	if ob, err2 := s.repo.GetOrderBook(req.Ticker, 10); err2 == nil {
		s.hub.Broadcast("orderbook:"+req.Ticker, ob)
	}
	// Broadcast trader cash update
	if trader, err2 := s.repo.GetTraderByID(req.TraderID); err2 == nil && trader != nil {
		s.hub.Broadcast("trader:"+req.TraderID, trader)
	}

	jsonResp(w, map[string]interface{}{"order": order, "result": result}, nil)
}

type CancelOrderReq struct {
	OrderID  string `json:"order_id"`
	TraderID string `json:"trader_id"`
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var req CancelOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	cancelled, err := s.repo.CancelOrder(req.OrderID, req.TraderID)
	if err != nil {
		jsonErr(w, err.Error(), 400)
		return
	}
	// Refund
	if cancelled.Side == SideBuy {
		refund := cancelled.Price * (cancelled.QtyLot - cancelled.FilledLot) * 100
		if refund > 0 {
			_ = s.repo.RefundCash(req.TraderID, refund)
		}
	} else {
		unfilled := cancelled.QtyLot - cancelled.FilledLot
		if unfilled > 0 {
			_ = s.repo.RefundShares(req.TraderID, cancelled.Ticker, unfilled)
		}
	}
	if trader, err2 := s.repo.GetTraderByID(req.TraderID); err2 == nil && trader != nil {
		s.hub.Broadcast("trader:"+req.TraderID, trader)
	}
	jsonResp(w, cancelled, nil)
}

func (s *Server) handleSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if err := SeedStocks(s.repo); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	if err := SeedTraders(s.repo); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	stocks, _ := s.repo.GetAllStocks()
	s.hub.Broadcast("stocks", stocks)
	jsonResp(w, map[string]string{"status": "ok"}, nil)
}

// ── JSON helpers ──────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
