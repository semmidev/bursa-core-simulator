package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

var store = sessions.NewCookieStore([]byte("super-secret-key-for-bei-exchange-simulator"))

func init() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
}

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
	repo      *Repo
	engine    *Engine
	hub       *Hub
	upgrader  websocket.Upgrader
	templates map[string]*template.Template
}

func NewServer(r *Repo, e *Engine) *Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	funcMap := template.FuncMap{
		"fmtRupiah":   FmtRupiah,
		"fmtNumber":   FmtNumber,
		"fmtBillions": FmtBillions,
		"fmtPercent":  FmtPercent,
		"sub": func(a, b int64) int64 { return a - b },
	}

	base := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles("templates/base.html"))
	
	templates := make(map[string]*template.Template)
	pages := []string{"market.html", "orderbook.html", "portfolio.html", "orders.html", "traders.html"}
	for _, page := range pages {
		templates[page] = template.Must(template.Must(base.Clone()).ParseFiles("templates/" + page))
	}

	hub := NewHub()
	go hub.Run()

	srv := &Server{
		repo:      r,
		engine:    e,
		hub:       hub,
		upgrader:  upgrader,
		templates: templates,
	}

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
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/market", s.handleMarket)
	mux.HandleFunc("/orderbook", s.handleOrderBookPage)
	mux.HandleFunc("/portfolio", s.handlePortfolioPage)
	mux.HandleFunc("/orders", s.handleOrdersPage)
	mux.HandleFunc("/traders", s.handleTradersPage)
	
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/order/submit", s.handleSubmitOrder)
	mux.HandleFunc("/order/cancel", s.handleCancelOrder)
	
	mux.HandleFunc("/ws", s.handleWS)

	fmt.Printf("\n  ⚡ BEI Exchange Simulator (Web) started on http://localhost%s\n\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── Helpers ───────────────────────────────────────────────────

func (s *Server) getSessionUser(r *http.Request) *Trader {
	session, err := store.Get(r, "bei-session")
	if err != nil {
		fmt.Println("GetSessionUser error:", err)
	}
	traderID, ok := session.Values["trader_id"].(string)
	if !ok {
		return nil
	}
	t, _ := s.repo.GetTraderByID(traderID)
	return t
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["User"] = s.getSessionUser(r)
	data["Path"] = r.URL.Path
	
	tmpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		fmt.Println("Template Execution Error:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ── Page Handlers ──────────────────────────────────────────────

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/market", http.StatusSeeOther)
}

func (s *Server) handleMarket(w http.ResponseWriter, r *http.Request) {
	stocks, _ := s.repo.GetAllStocks()
	s.render(w, r, "market.html", map[string]interface{}{"Stocks": stocks})
}

func (s *Server) handleOrderBookPage(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	stocks, _ := s.repo.GetAllStocks()
	
	var stock *Stock
	var ob *OrderBook
	var trades []Trade
	var araPrice, arbPrice int64
	
	if ticker != "" {
		stock, _ = s.repo.GetStock(ticker)
		ob, _ = s.repo.GetOrderBook(ticker, 15)
		trades, _ = s.repo.GetRecentTrades(ticker, 20)

		if stock != nil {
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
			araPrice = refPrice + int64(float64(refPrice)*maxPercent)
			arbPrice = refPrice - int64(float64(refPrice)*maxPercent)
		}
	}
	
	s.render(w, r, "orderbook.html", map[string]interface{}{
		"Stocks": stocks,
		"CurrentStock": stock,
		"OrderBook": ob,
		"Trades": trades,
		"Error": r.URL.Query().Get("err"),
		"AraPrice": araPrice,
		"ArbPrice": arbPrice,
	})
}

func (s *Server) handlePortfolioPage(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(r)
	if user == nil {
		http.Redirect(w, r, "/traders", http.StatusSeeOther)
		return
	}
	items, _ := s.repo.GetPortfolio(user.ID)
	
	s.render(w, r, "portfolio.html", map[string]interface{}{
		"Items": items,
	})
}

func (s *Server) handleOrdersPage(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(r)
	if user == nil {
		http.Redirect(w, r, "/traders", http.StatusSeeOther)
		return
	}
	orders, _ := s.repo.GetTraderOrders(user.ID, 50)
	
	s.render(w, r, "orders.html", map[string]interface{}{
		"Orders": orders,
		"Error": r.URL.Query().Get("err"),
	})
}

func (s *Server) handleTradersPage(w http.ResponseWriter, r *http.Request) {
	traders, _ := s.repo.GetAllTraders()
	s.render(w, r, "traders.html", map[string]interface{}{
		"Traders": traders,
	})
}

// ── Actions ────────────────────────────────────────────────────

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/traders", http.StatusSeeOther)
		return
	}
	username := r.FormValue("username")
	t, err := s.repo.GetTraderByUsername(username)
	if err != nil {
		fmt.Println("Login Error DB:", err)
	}
	if t != nil {
		session, err := store.Get(r, "bei-session")
		if err != nil {
			fmt.Println("Login Session Get Error:", err)
		}
		session.Values["trader_id"] = t.ID
		if err := session.Save(r, w); err != nil {
			fmt.Println("Login Session Save Error:", err)
		} else {
			fmt.Println("Login Successful for:", t.Username, "ID:", t.ID)
		}
	} else {
		fmt.Println("Login Failed: user not found", username)
	}
	http.Redirect(w, r, "/portfolio", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		session, _ := store.Get(r, "bei-session")
		session.Options.MaxAge = -1
		session.Save(r, w)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleSubmitOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	user := s.getSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	ticker := r.FormValue("ticker")
	side := r.FormValue("side")
	orderType := r.FormValue("order_type")
	price, _ := strconv.ParseInt(r.FormValue("price"), 10, 64)
	qtyLot, _ := strconv.ParseInt(r.FormValue("qty_lot"), 10, 64)
	
	_, _, err := s.engine.SubmitOrder(
		user.ID, ticker,
		OrderSide(side), OrderType(orderType),
		price, qtyLot,
	)
	
	if err != nil {
		http.Redirect(w, r, "/orderbook?ticker="+ticker+"&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	// Broadcast updates
	if ob, err2 := s.repo.GetOrderBook(ticker, 10); err2 == nil {
		s.hub.Broadcast("orderbook:"+ticker, ob)
	}
	if trades, err2 := s.repo.GetRecentTrades(ticker, 20); err2 == nil {
		s.hub.Broadcast("trades:"+ticker, trades)
	}
	if stocks, err2 := s.repo.GetAllStocks(); err2 == nil {
		s.hub.Broadcast("stocks", stocks)
	}
	if t, err2 := s.repo.GetTraderByID(user.ID); err2 == nil {
		s.hub.Broadcast("trader:"+user.ID, t)
	}

	http.Redirect(w, r, "/orderbook?ticker="+ticker, http.StatusSeeOther)
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	user := s.getSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	orderID := r.FormValue("order_id")
	
	cancelled, err := s.repo.CancelOrder(orderID, user.ID)
	if err != nil {
		http.Redirect(w, r, "/orders?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	
	// Refund
	if cancelled.Side == SideBuy {
		refund := cancelled.Price * (cancelled.QtyLot - cancelled.FilledLot) * 100
		if refund > 0 {
			_ = s.repo.RefundCash(user.ID, refund)
		}
	} else {
		unfilled := cancelled.QtyLot - cancelled.FilledLot
		if unfilled > 0 {
			_ = s.repo.RefundShares(user.ID, cancelled.Ticker, unfilled)
		}
	}
	
	if ob, err2 := s.repo.GetOrderBook(cancelled.Ticker, 10); err2 == nil {
		s.hub.Broadcast("orderbook:"+cancelled.Ticker, ob)
	}
	if t, err2 := s.repo.GetTraderByID(user.ID); err2 == nil {
		s.hub.Broadcast("trader:"+user.ID, t)
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
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
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
