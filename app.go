package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"
)

// ── Tabs ──────────────────────────────────────────────────────

type Tab int

const (
	TabMarket Tab = iota
	TabOrderBook
	TabTrade
	TabPortfolio
	TabOrders
	TabTraders
)

var tabNames = []string{"Market", "Order Book", "Trade", "Portfolio", "Orders", "Traders"}

// ── Messages ──────────────────────────────────────────────────

type tickMsg time.Time
type stocksLoadedMsg []Stock
type orderBookMsg *OrderBook
type tradesMsg []Trade
type portfolioMsg struct {
	items  []Portfolio
	stocks map[string]Stock
}
type ordersMsg []Order
type tradersMsg []Trader
type errMsg error
type successMsg string
type seedSuccessMsg struct{}
type orderSubmittedMsg struct {
	order  *Order
	result *MatchResult
}

// ── Key map ───────────────────────────────────────────────────

type keyMap struct {
	Tab1, Tab2, Tab3, Tab4, Tab5, Tab6 key.Binding
	Up, Down                           key.Binding
	Enter                              key.Binding
	Esc                                key.Binding
	Refresh                            key.Binding
	Seed                               key.Binding
	Cancel                             key.Binding
	Buy, Sell                          key.Binding
	Quit                               key.Binding
}

var keys = keyMap{
	Tab1:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "Market")),
	Tab2:    key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Order Book")),
	Tab3:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "Trade")),
	Tab4:    key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "Portfolio")),
	Tab5:    key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "Orders")),
	Tab6:    key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "Traders")),
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "confirm")),
	Esc:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Seed:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "seed data")),
	Cancel:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "cancel order")),
	Buy:     key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "buy")),
	Sell:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sell")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

// ── Trade form state ──────────────────────────────────────────

type TradeStep int

const (
	StepSelectTicker TradeStep = iota
	StepSelectSide
	StepSelectType
	StepInputOrder
	StepResult
)

type TradeForm struct {
	Step       TradeStep
	Ticker     string
	Side       OrderSide
	OrderType  OrderType
	QtyInput   textinput.Model
	PriceInput textinput.Model
	Qty        int64
	Price      int64
	Cursor     int
}

// ── Login form ────────────────────────────────────────────────

type LoginStep int

const (
	LoginInput LoginStep = iota
	LoginDone
)

type LoginForm struct {
	Step    LoginStep
	Input   textinput.Model
	Traders []Trader
	Cursor  int
}

// ── App Model ─────────────────────────────────────────────────

type AppModel struct {
	repo   *Repo
	eng    *Engine
	width  int
	height int

	// Session
	trader *Trader

	// Navigation
	activeTab    Tab
	marketCursor int

	// Data
	stocks    []Stock
	orderBook *OrderBook
	trades    []Trade
	portfolio portfolioMsg
	orders    []Order
	traders   []Trader

	// Selected stock for order book / trade
	selectedTicker string

	// Forms
	loginForm *LoginForm
	tradeForm *TradeForm

	// Orders tab cursor
	ordersCursor int

	// UI state
	spinner     spinner.Model
	loading     bool
	statusMsg   string
	statusIsErr bool
	lastUpdate  time.Time

	// Result of last submitted order
	lastOrder  *Order
	lastResult *MatchResult
}

func NewApp(r *Repo, e *Engine) AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorAccent)

	return AppModel{
		repo:    r,
		eng:     e,
		spinner: sp,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadStocks(),
		tickEvery(),
	)
}

func tickEvery() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ── Update ────────────────────────────────────────────────────

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(m.loadStocks(), tickEvery())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stocksLoadedMsg:
		m.stocks = []Stock(msg)
		m.loading = false
		m.lastUpdate = time.Now()
		return m, nil

	case orderBookMsg:
		m.orderBook = msg
		m.loading = false
		return m, nil

	case tradesMsg:
		m.trades = []Trade(msg)
		return m, nil

	case portfolioMsg:
		m.portfolio = msg
		m.loading = false
		return m, nil

	case ordersMsg:
		m.orders = []Order(msg)
		m.loading = false
		return m, nil

	case tradersMsg:
		m.traders = []Trader(msg)
		m.loading = false
		return m, nil

	case errMsg:
		m.statusMsg = msg.Error()
		m.statusIsErr = true
		m.loading = false
		return m, nil

	case successMsg:
		m.statusMsg = string(msg)
		m.statusIsErr = false
		return m, nil

	case seedSuccessMsg:
		m.statusMsg = "✓ Seed data berhasil!"
		m.statusIsErr = false
		m.loading = false
		return m, tea.Batch(m.loadStocks(), m.loadTraders())

	case orderSubmittedMsg:
		m.lastOrder = msg.order
		m.lastResult = msg.result
		if m.tradeForm != nil {
			m.tradeForm.Step = StepResult
		}
		// refresh data
		return m, tea.Batch(m.loadStocks(), m.loadOrders())

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate to active form inputs
	if m.loginForm != nil && m.loginForm.Step == LoginInput {
		var cmd tea.Cmd
		m.loginForm.Input, cmd = m.loginForm.Input.Update(msg)
		return m, cmd
	}
	if m.tradeForm != nil {
		switch m.tradeForm.Step {
		case StepInputOrder:
			var cmds []tea.Cmd
			var cmd tea.Cmd
			if m.tradeForm.Cursor == 0 {
				m.tradeForm.PriceInput, cmd = m.tradeForm.PriceInput.Update(msg)
				cmds = append(cmds, cmd)
			} else if m.tradeForm.Cursor == 1 {
				m.tradeForm.QtyInput, cmd = m.tradeForm.QtyInput.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
	}

	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	isTyping := false
	if m.loginForm != nil && m.loginForm.Step == LoginInput {
		isTyping = true
	}
	if m.tradeForm != nil && m.tradeForm.Step == StepInputOrder {
		isTyping = true
	}

	// Always allow ctrl+c
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if !isTyping {
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}

		// Tab switching
		switch {
		case key.Matches(msg, keys.Tab1):
			m.loginForm = nil
			m.tradeForm = nil
			m.activeTab = TabMarket
			return m, m.loadStocks()
		case key.Matches(msg, keys.Tab2):
			m.loginForm = nil
			m.tradeForm = nil
			m.activeTab = TabOrderBook
			if m.selectedTicker == "" && len(m.stocks) > 0 {
				m.selectedTicker = m.stocks[m.marketCursor].Ticker
			}
			return m, tea.Batch(m.loadOrderBook(), m.loadTrades())
		case key.Matches(msg, keys.Tab3):
			m.loginForm = nil
			if m.trader == nil {
				m.statusMsg = "Login terlebih dahulu (tekan L)"
				m.statusIsErr = true
				return m, nil
			}
			if m.tradeForm == nil {
				m.activeTab = TabTrade
				m.openTradeForm()
			}
			return m, nil
		case key.Matches(msg, keys.Tab4):
			m.loginForm = nil
			m.tradeForm = nil
			m.activeTab = TabPortfolio
			return m, m.loadPortfolio()
		case key.Matches(msg, keys.Tab5):
			m.loginForm = nil
			m.tradeForm = nil
			m.activeTab = TabOrders
			return m, m.loadOrders()
		case key.Matches(msg, keys.Tab6):
			m.loginForm = nil
			m.tradeForm = nil
			m.activeTab = TabTraders
			return m, m.loadTraders()
		}

		// Common actions
		switch msg.String() {
		case "L", "l":
			m.tradeForm = nil
			m.openLogin()
			return m, m.loadTraders()
		case "o", "O":
			if m.trader != nil {
				m.trader = nil
				m.statusMsg = "Logout berhasil"
				m.statusIsErr = false
			}
			return m, nil
		case "S":
			m.loading = true
			return m, m.doSeed()
		case "r":
			return m, m.refreshCurrentTab()
		}
	}

	// Delegate to active forms
	if m.loginForm != nil {
		return m.handleLoginKey(msg)
	}

	if m.tradeForm != nil {
		return m.handleTradeKey(msg)
	}

	// Tab-specific keys
	switch m.activeTab {
	case TabMarket:
		return m.handleMarketKey(msg)
	case TabOrderBook:
		return m.handleOrderBookKey(msg)
	case TabOrders:
		return m.handleOrdersKey(msg)
	}

	return m, nil
}

func (m *AppModel) handleMarketKey(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.marketCursor > 0 {
			m.marketCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.marketCursor < len(m.stocks)-1 {
			m.marketCursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(m.stocks) > 0 {
			m.selectedTicker = m.stocks[m.marketCursor].Ticker
			m.activeTab = TabOrderBook
			return *m, tea.Batch(m.loadOrderBook(), m.loadTrades())
		}
	case msg.String() == "b":
		if m.trader == nil {
			m.statusMsg = "Login dulu (tekan L)"
			m.statusIsErr = true
			return *m, nil
		}
		if len(m.stocks) > 0 {
			m.selectedTicker = m.stocks[m.marketCursor].Ticker
			m.activeTab = TabTrade
			m.openTradeFormWith(m.stocks[m.marketCursor].Ticker, SideBuy)
		}
	case msg.String() == "s":
		if m.trader == nil {
			m.statusMsg = "Login dulu (tekan L)"
			m.statusIsErr = true
			return *m, nil
		}
		if len(m.stocks) > 0 {
			m.selectedTicker = m.stocks[m.marketCursor].Ticker
			m.activeTab = TabTrade
			m.openTradeFormWith(m.stocks[m.marketCursor].Ticker, SideSell)
		}
	}
	return *m, nil
}

func (m *AppModel) handleOrderBookKey(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	if msg.String() == "b" && m.trader != nil {
		m.activeTab = TabTrade
		m.openTradeFormWith(m.selectedTicker, SideBuy)
	}
	if msg.String() == "s" && m.trader != nil {
		m.activeTab = TabTrade
		m.openTradeFormWith(m.selectedTicker, SideSell)
	}
	return *m, nil
}

func (m *AppModel) handleOrdersKey(msg tea.KeyMsg) (AppModel, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Up):
		if m.ordersCursor > 0 {
			m.ordersCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.ordersCursor < len(m.orders)-1 {
			m.ordersCursor++
		}
	case msg.String() == "c":
		if m.trader != nil && len(m.orders) > 0 {
			o := m.orders[m.ordersCursor]
			if o.Status == StatusOpen || o.Status == StatusPartial {
				return *m, m.cancelOrder(o.ID)
			}
		}
	}
	return *m, nil
}

func (m AppModel) handleLoginKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	lf := m.loginForm
	switch {
	case key.Matches(msg, keys.Esc):
		m.loginForm = nil
		return m, nil

	case key.Matches(msg, keys.Up):
		if lf.Cursor > 0 {
			lf.Cursor--
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		if lf.Cursor < len(lf.Traders)-1 {
			lf.Cursor++
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		if len(lf.Traders) == 0 {
			m.loading = true
			return m, m.doSeed()
		}
		if len(lf.Traders) > 0 && lf.Cursor < len(lf.Traders) {
			t := lf.Traders[lf.Cursor]
			full, _ := m.repo.GetTraderByID(t.ID)
			if full != nil {
				m.trader = full
			}
			m.loginForm = nil
			m.statusMsg = fmt.Sprintf("Selamat datang, %s!", t.FullName)
			m.statusIsErr = false
		}
		return m, nil
	}

	var cmd tea.Cmd
	lf.Input, cmd = lf.Input.Update(msg)
	return m, cmd
}

func (m AppModel) handleTradeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	tf := m.tradeForm

	if key.Matches(msg, keys.Esc) {
		if tf.Step == StepResult || tf.Step == StepSelectTicker {
			m.tradeForm = nil
			m.activeTab = TabMarket
			return m, nil
		}
		tf.Step--
		return m, nil
	}

	switch tf.Step {
	case StepSelectTicker:
		switch {
		case key.Matches(msg, keys.Up):
			if tf.Cursor > 0 {
				tf.Cursor--
			}
		case key.Matches(msg, keys.Down):
			if tf.Cursor < len(m.stocks)-1 {
				tf.Cursor++
			}
		case key.Matches(msg, keys.Enter):
			if len(m.stocks) > 0 {
				tf.Ticker = m.stocks[tf.Cursor].Ticker
				tf.Step = StepSelectSide
				tf.Cursor = 0
			}
		}

	case StepSelectSide:
		switch {
		case key.Matches(msg, keys.Up) || key.Matches(msg, keys.Down):
			if tf.Cursor == 0 {
				tf.Cursor = 1
			} else {
				tf.Cursor = 0
			}
		case key.Matches(msg, keys.Enter):
			if tf.Cursor == 0 {
				tf.Side = SideBuy
			} else {
				tf.Side = SideSell
			}
			tf.Step = StepSelectType
			tf.Cursor = 0
		}

	case StepSelectType:
		switch {
		case key.Matches(msg, keys.Up) || key.Matches(msg, keys.Down):
			if tf.Cursor == 0 {
				tf.Cursor = 1
			} else {
				tf.Cursor = 0
			}
		case key.Matches(msg, keys.Enter):
			if tf.Cursor == 0 {
				tf.OrderType = TypeLimit
			} else {
				tf.OrderType = TypeMarket
			}
			tf.Step = StepInputOrder
			if tf.OrderType == TypeLimit {
				tf.Cursor = 0
				tf.PriceInput.Focus()
				tf.QtyInput.Blur()
				if tf.PriceInput.Value() == "" {
					for _, s := range m.stocks {
						if s.Ticker == tf.Ticker {
							tf.PriceInput.SetValue(fmt.Sprintf("%d", s.LastPrice))
							break
						}
					}
				}
			} else {
				tf.Cursor = 1
				tf.QtyInput.Focus()
				tf.PriceInput.Blur()
			}
			if tf.QtyInput.Value() == "" {
				tf.QtyInput.SetValue("1")
			}
		}

	case StepInputOrder:
		if key.Matches(msg, keys.Up) || msg.String() == "+" {
			if tf.Cursor == 0 {
				var p int64
				fmt.Sscanf(strings.ReplaceAll(tf.PriceInput.Value(), ".", ""), "%d", &p)
				tf.PriceInput.SetValue(fmt.Sprintf("%d", p+5))
			} else if tf.Cursor == 1 {
				var q int64
				fmt.Sscanf(strings.ReplaceAll(tf.QtyInput.Value(), ".", ""), "%d", &q)
				tf.QtyInput.SetValue(fmt.Sprintf("%d", q+1))
			} else if tf.Cursor == 2 {
				tf.Cursor = 1
				tf.QtyInput.Focus()
				tf.PriceInput.Blur()
			}
			return m, nil
		} else if key.Matches(msg, keys.Down) || msg.String() == "-" {
			if tf.Cursor == 0 {
				var p int64
				fmt.Sscanf(strings.ReplaceAll(tf.PriceInput.Value(), ".", ""), "%d", &p)
				if p > 5 {
					p -= 5
				}
				tf.PriceInput.SetValue(fmt.Sprintf("%d", p))
			} else if tf.Cursor == 1 {
				var q int64
				fmt.Sscanf(strings.ReplaceAll(tf.QtyInput.Value(), ".", ""), "%d", &q)
				if q > 1 {
					q -= 1
				}
				tf.QtyInput.SetValue(fmt.Sprintf("%d", q))
			}
			return m, nil
		} else if msg.String() == "tab" || key.Matches(msg, keys.Enter) {
			if tf.Cursor == 0 {
				var price int64
				fmt.Sscanf(strings.ReplaceAll(tf.PriceInput.Value(), ".", ""), "%d", &price)
				if price <= 0 {
					m.statusMsg = "Harga harus > 0"
					m.statusIsErr = true
					return m, nil
				}
				tf.Price = price
				tf.Cursor = 1
				tf.PriceInput.Blur()
				tf.QtyInput.Focus()
			} else if tf.Cursor == 1 {
				var qty int64
				fmt.Sscanf(strings.ReplaceAll(tf.QtyInput.Value(), ".", ""), "%d", &qty)
				if qty <= 0 {
					m.statusMsg = "Jumlah lot harus > 0"
					m.statusIsErr = true
					return m, nil
				}
				tf.Qty = qty
				tf.Cursor = 2
				tf.QtyInput.Blur()
				tf.PriceInput.Blur()
			} else if tf.Cursor == 2 {
				var price, qty int64
				if tf.OrderType == TypeLimit {
					price = tf.Price
				} else {
					for _, s := range m.stocks {
						if s.Ticker == tf.Ticker {
							price = s.LastPrice
							break
						}
					}
				}
				qty = tf.Qty
				investment := price * qty * 100
				fee := investment * 15 / 10000
				totalInvestment := investment + fee
				if tf.Side == SideBuy && m.trader != nil && totalInvestment > m.trader.CashBalance {
					m.statusMsg = "Saldo tidak mencukupi untuk order ini"
					m.statusIsErr = true
					return m, nil
				}
				return m, m.submitOrder()
			}
			return m, nil
		} else if key.Matches(msg, keys.Esc) {
			if tf.Cursor == 2 {
				tf.Cursor = 1
				tf.QtyInput.Focus()
			} else if tf.Cursor == 1 {
				if tf.OrderType == TypeLimit {
					tf.Cursor = 0
					tf.QtyInput.Blur()
					tf.PriceInput.Focus()
				} else {
					tf.Step = StepSelectType
					tf.Cursor = 0
				}
			} else if tf.Cursor == 0 {
				tf.Step = StepSelectType
				tf.Cursor = 0
			}
			return m, nil
		} else {
			var cmd tea.Cmd
			if tf.Cursor == 0 {
				tf.PriceInput, cmd = tf.PriceInput.Update(msg)
			} else if tf.Cursor == 1 {
				tf.QtyInput, cmd = tf.QtyInput.Update(msg)
			}
			return m, cmd
		}

	case StepResult:
		if key.Matches(msg, keys.Enter) || key.Matches(msg, keys.Esc) {
			m.tradeForm = nil
			m.activeTab = TabMarket
		}
	}

	return m, nil
}

func (m *AppModel) openLogin() {
	ti := textinput.New()
	ti.Placeholder = "username..."
	ti.CharLimit = 50
	m.loginForm = &LoginForm{
		Step:  LoginInput,
		Input: ti,
	}
}

func (m *AppModel) openTradeForm() {
	qtyIn := textinput.New()
	qtyIn.Placeholder = "contoh: 5"
	qtyIn.CharLimit = 10

	priceIn := textinput.New()
	priceIn.Placeholder = "contoh: 10300"
	priceIn.CharLimit = 15

	m.tradeForm = &TradeForm{
		Step:       StepSelectTicker,
		QtyInput:   qtyIn,
		PriceInput: priceIn,
	}
}

func (m *AppModel) openTradeFormWith(ticker string, side OrderSide) {
	m.openTradeForm()
	m.tradeForm.Ticker = ticker
	m.tradeForm.Side = side
	m.tradeForm.Step = StepSelectType
	m.tradeForm.Cursor = 0
}

// ── Commands ──────────────────────────────────────────────────

func (m AppModel) loadStocks() tea.Cmd {
	return func() tea.Msg {
		stocks, err := m.repo.GetAllStocks()
		if err != nil {
			return errMsg(err)
		}
		return stocksLoadedMsg(stocks)
	}
}

func (m AppModel) loadOrderBook() tea.Cmd {
	ticker := m.selectedTicker
	return func() tea.Msg {
		ob, err := m.repo.GetOrderBook(ticker, 10)
		if err != nil {
			return errMsg(err)
		}
		return orderBookMsg(ob)
	}
}

func (m AppModel) loadTrades() tea.Cmd {
	ticker := m.selectedTicker
	return func() tea.Msg {
		trades, err := m.repo.GetRecentTrades(ticker, 15)
		if err != nil {
			return errMsg(err)
		}
		return tradesMsg(trades)
	}
}

func (m AppModel) loadPortfolio() tea.Cmd {
	if m.trader == nil {
		return nil
	}
	traderID := m.trader.ID
	return func() tea.Msg {
		items, err := m.repo.GetPortfolio(traderID)
		if err != nil {
			return errMsg(err)
		}
		allStocks, _ := m.repo.GetAllStocks()
		sm := make(map[string]Stock)
		for _, s := range allStocks {
			sm[s.Ticker] = s
		}
		return portfolioMsg{items: items, stocks: sm}
	}
}

func (m AppModel) loadOrders() tea.Cmd {
	if m.trader == nil {
		return nil
	}
	traderID := m.trader.ID
	return func() tea.Msg {
		orders, err := m.repo.GetTraderOrders(traderID, 30)
		if err != nil {
			return errMsg(err)
		}
		return ordersMsg(orders)
	}
}

func (m AppModel) loadTraders() tea.Cmd {
	return func() tea.Msg {
		traders, err := m.repo.GetAllTraders()
		if err != nil {
			return errMsg(err)
		}
		if m.loginForm != nil {
			m.loginForm.Traders = traders
		}
		return tradersMsg(traders)
	}
}

func (m AppModel) doSeed() tea.Cmd {
	r := m.repo
	return func() tea.Msg {
		if err := SeedStocks(r); err != nil {
			return errMsg(err)
		}
		if err := SeedTraders(r); err != nil {
			return errMsg(err)
		}
		return seedSuccessMsg{}
	}
}

func (m AppModel) submitOrder() tea.Cmd {
	tf := m.tradeForm
	traderID := m.trader.ID
	eng := m.eng
	return func() tea.Msg {
		order, result, err := eng.SubmitOrder(traderID, tf.Ticker, tf.Side, tf.OrderType, tf.Price, tf.Qty)
		if err != nil {
			return errMsg(err)
		}
		return orderSubmittedMsg{order: order, result: result}
	}
}

func (m AppModel) cancelOrder(orderID string) tea.Cmd {
	traderID := m.trader.ID
	r := m.repo
	return func() tea.Msg {
		cancelled, err := r.CancelOrder(orderID, traderID)
		if err != nil {
			return errMsg(err)
		}
		// Refund
		if cancelled.Side == SideBuy {
			refund := cancelled.Price * (cancelled.QtyLot - cancelled.FilledLot) * 100
			if refund > 0 {
				_ = r.RefundCash(traderID, refund)
			}
		} else {
			unfilled := cancelled.QtyLot - cancelled.FilledLot
			if unfilled > 0 {
				_ = r.RefundShares(traderID, cancelled.Ticker, unfilled)
			}
		}
		return successMsg(fmt.Sprintf("✓ Order %s dibatalkan", orderID[:8]))
	}
}

func (m AppModel) refreshCurrentTab() tea.Cmd {
	switch m.activeTab {
	case TabMarket:
		return m.loadStocks()
	case TabOrderBook:
		return tea.Batch(m.loadOrderBook(), m.loadTrades())
	case TabPortfolio:
		return m.loadPortfolio()
	case TabOrders:
		return m.loadOrders()
	case TabTraders:
		return m.loadTraders()
	}
	return nil
}

// ── View ──────────────────────────────────────────────────────

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Overlay: login form
	if m.loginForm != nil {
		return m.renderLoginOverlay()
	}

	var parts []string

	parts = append(parts, m.renderHeader())
	parts = append(parts, m.renderTabs())

	// Main content
	contentHeight := m.height - 5 // header+tabs+footer
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content string
	switch m.activeTab {
	case TabMarket:
		content = m.renderMarket(contentHeight)
	case TabOrderBook:
		content = m.renderOrderBook(contentHeight)
	case TabTrade:
		content = m.renderTradeForm(contentHeight)
	case TabPortfolio:
		content = m.renderPortfolio(contentHeight)
	case TabOrders:
		content = m.renderOrders(contentHeight)
	case TabTraders:
		content = m.renderTraders(contentHeight)
	}

	parts = append(parts, content)
	parts = append(parts, m.renderFooter())

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// ── Header ────────────────────────────────────────────────────

func (m AppModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Render("⚡ BEI Exchange Simulator")

	var session string
	if m.trader != nil {
		cash := lipgloss.NewStyle().Foreground(ColorGreen).Render(FmtRupiah(m.trader.CashBalance))
		session = lipgloss.NewStyle().Foreground(ColorMuted).Render("  │  ") +
			lipgloss.NewStyle().Foreground(ColorCyan).Bold(true).Render(m.trader.Username) +
			lipgloss.NewStyle().Foreground(ColorMuted).Render("  Kas: ") + cash
	} else {
		session = lipgloss.NewStyle().Foreground(ColorMuted).Render("  │  tekan L untuk login")
	}

	ts := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		m.lastUpdate.Format("15:04:05"))

	left := title + session
	right := ts
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	line := lipgloss.NewStyle().
		Background(ColorSurface).
		Width(m.width).
		Render(left + strings.Repeat(" ", gap) + right)

	return line
}

// ── Tabs ──────────────────────────────────────────────────────

func (m AppModel) renderTabs() string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%d:%s", i+1, name)
		if Tab(i) == m.activeTab {
			tabs = append(tabs, StyleActiveTab.Render(label))
		} else {
			tabs = append(tabs, StyleInactiveTab.Render(label))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	filler := lipgloss.NewStyle().
		Background(ColorSurface).
		Width(m.width - lipgloss.Width(row)).
		Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, row, filler)
}

// ── Footer ────────────────────────────────────────────────────

func (m AppModel) renderFooter() string {
	hints := " 1-6:tabs  ↑↓:nav  ⏎:select  b:buy  s:sell  L:login  o:logout  S:seed  r:refresh  q:quit"

	var statusPart string
	if m.statusMsg != "" {
		if m.statusIsErr {
			statusPart = lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render(" ✗ " + m.statusMsg)
		} else {
			statusPart = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render(" ✓ " + m.statusMsg)
		}
	}

	hintStyle := lipgloss.NewStyle().Foreground(ColorMuted).Background(ColorSurface)
	line := hintStyle.Width(m.width).Render(hints + statusPart)
	return line
}

// ── Market view ───────────────────────────────────────────────

func (m AppModel) renderMarket(height int) string {
	if len(m.stocks) == 0 {
		return StyleMuted.Render("\n  Belum ada data. Tekan S untuk seed data demo.\n")
	}

	// Column widths
	col := []int{6, 30, 16, 14, 12, 10, 12, 9}

	header := StyleTableHeader.Render(
		fmt.Sprintf("  %-*s %-*s %-*s %*s %*s %*s %*s %*s",
			col[0], "KODE",
			col[1], "PERUSAHAAN",
			col[2], "SEKTOR",
			col[3], "HARGA",
			col[4], "PERUBAHAN",
			col[5], "PCT",
			col[6], "VOLUME (L)",
			col[7], "HALTED",
		),
	)

	var rows []string
	rows = append(rows, header)
	rows = append(rows, StyleMuted.Render("  "+strings.Repeat("─", m.width-4)))

	for i, s := range m.stocks {
		change := s.Change()
		pct := s.ChangePercent()

		ticker := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(fmt.Sprintf("%-6s", s.Ticker))
		name := TruncStr(s.CompanyName, col[1])
		sector := TruncStr(s.Sector, col[2])

		priceStr := PriceStyle(s.LastPrice, s.PrevClose).Render(fmt.Sprintf("%14s", FmtRupiah(s.LastPrice)))

		chgStr := ChangeStyle(change).Render(fmt.Sprintf("%+d", change))
		if change >= 0 {
			chgStr = ChangeStyle(change).Render("+" + FmtNumber(change))
		} else {
			chgStr = ChangeStyle(change).Render(FmtNumber(change))
		}
		chgStr = Pad(chgStr, col[4]+8) // +8 for ansi escape

		arrow := ChangeArrow(change)
		pctStr := ChangeStyle(change).Render(fmt.Sprintf("%.2f%%", pct))

		volStr := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("%10s", FmtNumber(s.VolumeLot)))

		haltStr := "      "
		if s.IsHalted {
			haltStr = lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render(" HALT ")
		}

		row := fmt.Sprintf("  %s %-*s %-*s %s  %s %s %-8s %s  %s",
			ticker,
			col[1], name,
			col[2], sector,
			priceStr,
			arrow, chgStr,
			pctStr,
			volStr,
			haltStr,
		)

		if i == m.marketCursor {
			row = StyleSelectedRow.Width(m.width).Render(
				fmt.Sprintf("  %-6s %-*s %-*s %14s  %s %s%-8s  %10s  %s",
					s.Ticker,
					col[1], name,
					col[2], sector,
					FmtRupiah(s.LastPrice),
					arrow, chgStr,
					fmt.Sprintf("%.2f%%", pct),
					FmtNumber(s.VolumeLot),
					haltStr,
				),
			)
		}

		rows = append(rows, row)
	}

	rows = append(rows, StyleMuted.Render("  "+strings.Repeat("─", m.width-4)))
	rows = append(rows, StyleMuted.Render(fmt.Sprintf("  %d saham  │  auto-refresh setiap 3 detik  │  ↑↓ pilih  ⏎ order book  b/s beli/jual", len(m.stocks))))

	return strings.Join(rows, "\n")
}

// ── Order Book view ───────────────────────────────────────────

func (m AppModel) renderOrderBook(height int) string {
	if m.selectedTicker == "" {
		return StyleMuted.Render("\n  Pilih saham di tab Market (⏎) terlebih dahulu.\n")
	}

	// Get stock info
	var stock *Stock
	for i := range m.stocks {
		if m.stocks[i].Ticker == m.selectedTicker {
			stock = &m.stocks[i]
			break
		}
	}

	halfW := (m.width - 4) / 2

	// Header
	title := StyleTitle.Bold(true).Render(fmt.Sprintf("Order Book — %s", m.selectedTicker))
	if stock != nil {
		priceStr := PriceStyle(stock.LastPrice, stock.PrevClose).Render(FmtRupiah(stock.LastPrice))
		chgStr := ChangeStyle(stock.Change()).Render(FmtPercent(stock.ChangePercent()))
		title += "  " + priceStr + "  " + chgStr
	}

	var lines []string
	lines = append(lines, "\n  "+title)
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	// Column headers
	bidHdr := BidStyle().Bold(true).Render(fmt.Sprintf("  %-10s  %8s  %6s", "HARGA BELI", "LOT", "ORDER"))
	askHdr := AskStyle().Bold(true).Render(fmt.Sprintf("  %-10s  %8s  %6s", "HARGA JUAL", "LOT", "ORDER"))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(halfW).Render(bidHdr),
		StyleMuted.Render("│"),
		lipgloss.NewStyle().Width(halfW).Render(askHdr),
	))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	ob := m.orderBook
	maxRows := 10
	if ob != nil {
		if len(ob.Bids) > maxRows {
			maxRows = len(ob.Bids)
		}
		if len(ob.Asks) > maxRows {
			maxRows = len(ob.Asks)
		}
	}
	if maxRows > 12 {
		maxRows = 12
	}

	var maxBidLot, maxAskLot int64 = 1, 1
	if ob != nil {
		for _, b := range ob.Bids {
			if b.TotalLot > maxBidLot {
				maxBidLot = b.TotalLot
			}
		}
		for _, a := range ob.Asks {
			if a.TotalLot > maxAskLot {
				maxAskLot = a.TotalLot
			}
		}
	}

	barWidth := 8

	for i := 0; i < maxRows; i++ {
		var bidCell, askCell string

		if ob != nil && i < len(ob.Bids) {
			b := ob.Bids[i]
			barLen := int(float64(b.TotalLot) / float64(maxBidLot) * float64(barWidth))
			if barLen < 1 {
				barLen = 1
			}
			bar := BidBgStyle().Render(strings.Repeat("█", barLen) + strings.Repeat("░", barWidth-barLen))
			bidCell = fmt.Sprintf("  %s  %s  %s",
				BidStyle().Bold(true).Render(fmt.Sprintf("%-10s", FmtRupiah(b.Price))),
				bar,
				BidStyle().Render(fmt.Sprintf("%6d", b.Orders)),
			)
		} else {
			bidCell = strings.Repeat(" ", halfW)
		}

		if ob != nil && i < len(ob.Asks) {
			a := ob.Asks[i]
			barLen := int(float64(a.TotalLot) / float64(maxAskLot) * float64(barWidth))
			if barLen < 1 {
				barLen = 1
			}
			bar := AskBgStyle().Render(strings.Repeat("█", barLen) + strings.Repeat("░", barWidth-barLen))
			askCell = fmt.Sprintf("  %s  %s  %s",
				AskStyle().Bold(true).Render(fmt.Sprintf("%-10s", FmtRupiah(a.Price))),
				bar,
				AskStyle().Render(fmt.Sprintf("%6d", a.Orders)),
			)
		} else {
			askCell = strings.Repeat(" ", halfW)
		}

		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(halfW).Render(bidCell),
			StyleMuted.Render("│"),
			lipgloss.NewStyle().Render(askCell),
		))
	}

	if ob != nil && len(ob.Asks) > 0 && len(ob.Bids) > 0 {
		spread := ob.Asks[0].Price - ob.Bids[0].Price
		lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))
		lines = append(lines, fmt.Sprintf("  %s %s  %s %s",
			StyleMuted.Render("Spread:"),
			StyleWarning.Render(FmtRupiah(spread)),
			StyleMuted.Render("Last:"),
			func() string {
				if stock != nil {
					return PriceStyle(stock.LastPrice, stock.PrevClose).Render(FmtRupiah(stock.LastPrice))
				}
				return "-"
			}(),
		))
	}

	// Recent trades
	if len(m.trades) > 0 {
		lines = append(lines, "\n  "+StyleTitle.Render("Recent Trades"))
		lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", 50)))
		lines = append(lines, "  "+StyleTableHeader.Render(fmt.Sprintf("%-20s  %14s  %8s", "WAKTU", "HARGA", "LOT")))
		limit := 5
		if len(m.trades) < limit {
			limit = len(m.trades)
		}
		for _, t := range m.trades[:limit] {
			lines = append(lines, fmt.Sprintf("  %-20s  %s  %s",
				StyleMuted.Render(t.TradedAt.Format("02 Jan 15:04:05")),
				StyleSuccess.Render(fmt.Sprintf("%14s", FmtRupiah(t.Price))),
				StyleAccent.Render(fmt.Sprintf("%8s", FmtNumber(t.QtyLot))),
			))
		}
	}

	lines = append(lines, "\n  "+StyleMuted.Render("b:beli  s:jual  r:refresh"))

	return strings.Join(lines, "\n")
}

// ── Trade Form view ───────────────────────────────────────────

func (m AppModel) renderTradeForm(height int) string {
	tf := m.tradeForm
	if tf == nil {
		return StyleMuted.Render("\n  Membuka form order...\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+StyleTitle.Bold(true).Render("⚡ Order Baru"))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", 60)))

	// Progress indicator
	steps := []string{"Saham", "Sisi", "Tipe", "Input Order", "Selesai"}
	stepIdx := int(tf.Step)
	if stepIdx >= len(steps) {
		stepIdx = len(steps) - 1
	}
	var stepBar []string
	for i, s := range steps {
		if i < stepIdx {
			stepBar = append(stepBar, StyleSuccess.Render("✓ "+s))
		} else if i == stepIdx {
			stepBar = append(stepBar, StyleAccent.Bold(true).Render("▶ "+s))
		} else {
			stepBar = append(stepBar, StyleMuted.Render("○ "+s))
		}
	}
	lines = append(lines, "  "+strings.Join(stepBar, StyleMuted.Render(" → ")))
	lines = append(lines, "")

	var stock *Stock
	if tf.Ticker != "" {
		for i := range m.stocks {
			if m.stocks[i].Ticker == tf.Ticker {
				stock = &m.stocks[i]
				break
			}
		}
	}

	var tickerInfo string
	if stock != nil {
		priceStr := PriceStyle(stock.LastPrice, stock.PrevClose).Render(FmtRupiah(stock.LastPrice))
		chgStr := ChangeStyle(stock.Change()).Render(FmtPercent(stock.ChangePercent()))
		tickerInfo = fmt.Sprintf("%s ( %s  %s )", StyleAccent.Bold(true).Render(tf.Ticker), priceStr, chgStr)
	} else {
		tickerInfo = StyleAccent.Bold(true).Render(tf.Ticker)
	}

	sideLabel := ""
	if tf.Side != "" {
		if tf.Side == SideBuy {
			sideLabel = StyleSuccess.Render(string(tf.Side))
		} else {
			sideLabel = StyleError.Render(string(tf.Side))
		}
	}

	switch tf.Step {
	case StepSelectTicker:
		lines = append(lines, "  "+StyleBold.Render("Pilih Saham (↑↓ untuk navigasi, ⏎ untuk memilih):"))
		lines = append(lines, "")
		limit := 10
		for i, s := range m.stocks {
			if i >= limit {
				break
			}
			row := fmt.Sprintf("  %-6s  %-30s  %s",
				s.Ticker, TruncStr(s.CompanyName, 30), FmtRupiah(s.LastPrice))
			if i == tf.Cursor {
				lines = append(lines, StyleSelectedRow.Render(row))
			} else {
				lines = append(lines, row)
			}
		}

	case StepSelectSide:
		lines = append(lines, "  "+StyleBold.Render("Saham: ")+tickerInfo)
		lines = append(lines, "  "+StyleBold.Render("Pilih Sisi:"))
		lines = append(lines, "")
		options := []struct {
			label string
			style lipgloss.Style
		}{
			{"  BUY  — Beli Saham", lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)},
			{"  SELL — Jual Saham", lipgloss.NewStyle().Foreground(ColorRed).Bold(true)},
		}
		for i, opt := range options {
			if i == tf.Cursor {
				lines = append(lines, StyleSelectedRow.Render("  ▶ "+opt.label))
			} else {
				lines = append(lines, opt.style.Render("    "+opt.label))
			}
		}

	case StepSelectType:
		lines = append(lines, fmt.Sprintf("  %s: %s — %s", StyleBold.Render("Order"), tickerInfo, sideLabel))
		lines = append(lines, "  "+StyleBold.Render("Pilih Tipe Order:"))
		lines = append(lines, "")
		options := []struct{ label, desc string }{
			{"LIMIT", "Pasang di harga tertentu, tunggu match"},
			{"MARKET", "Eksekusi instan di harga terbaik"},
		}
		for i, opt := range options {
			row := fmt.Sprintf("    %-8s — %s", opt.label, opt.desc)
			if i == tf.Cursor {
				lines = append(lines, StyleSelectedRow.Render("  ▶"+row))
			} else {
				lines = append(lines, StyleMuted.Render("    "+row))
			}
		}

	case StepInputOrder:
		balance := int64(0)
		if m.trader != nil {
			balance = m.trader.CashBalance
		}
		lines = append(lines, "  "+StyleMuted.Render("Trading Balance")+"  "+StyleSuccess.Bold(true).Render(FmtRupiah(balance)))
		lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", 40)))
		lines = append(lines, "")

		var price, qty int64
		if tf.OrderType == TypeLimit {
			fmt.Sscanf(strings.ReplaceAll(tf.PriceInput.Value(), ".", ""), "%d", &price)
		} else {
			if stock != nil {
				price = stock.LastPrice
			}
		}
		fmt.Sscanf(strings.ReplaceAll(tf.QtyInput.Value(), ".", ""), "%d", &qty)

		investment := price * qty * 100
		fee := investment * 15 / 10000
		totalInvestment := investment + fee

		invStyle := StyleAccent
		if tf.Side == SideBuy && totalInvestment > balance {
			invStyle = StyleError
		}

		lines = append(lines, "  "+StyleMuted.Render("Investment (Inc. Fee)")+"  "+invStyle.Bold(true).Render(FmtRupiah(totalInvestment)))
		if tf.Side == SideBuy && totalInvestment > balance {
			lines = append(lines, "  "+StyleError.Render("⚠ Saldo tidak mencukupi!"))
		}
		lines = append(lines, "")

		lines = append(lines, fmt.Sprintf("  %s  %s  %s", tickerInfo, sideLabel, StyleCyan.Render(string(tf.OrderType))))
		lines = append(lines, "")

		if tf.OrderType == TypeLimit {
			priceLabel := StyleBold.Render("Harga per lembar :")
			if tf.Cursor == 0 {
				lines = append(lines, "  "+priceLabel+" "+tf.PriceInput.View())
			} else {
				lines = append(lines, "  "+priceLabel+"  "+tf.PriceInput.Value())
			}
		} else {
			lines = append(lines, "  "+StyleBold.Render("Harga per lembar :")+" "+StyleWarning.Render("MARKET"))
		}

		lotLabel := StyleBold.Render("Jumlah Lot       :")
		if tf.Cursor == 1 {
			lines = append(lines, "  "+lotLabel+" "+tf.QtyInput.View())
		} else {
			lines = append(lines, "  "+lotLabel+"  "+tf.QtyInput.Value())
		}
		lines = append(lines, "  "+StyleMuted.Render("(1 lot = 100 lembar)"))
		lines = append(lines, "")

		if tf.Cursor == 2 {
			lines = append(lines, "  "+StyleAccent.Bold(true).Render("▶ [ KONFIRMASI & KIRIM ORDER ]"))
		} else {
			lines = append(lines, "  "+StyleMuted.Render("  [ KONFIRMASI & KIRIM ORDER ]"))
		}
		lines = append(lines, "")
		lines = append(lines, "  "+StyleMuted.Render("↑/↓: +/- nilai   Tab/⏎: Lanjut   Esc: Kembali"))

	case StepResult:
		if m.lastOrder == nil {
			lines = append(lines, StyleMuted.Render("  Loading..."))
			break
		}
		lines = append(lines, "  "+StyleSuccess.Bold(true).Render("✓ Order Berhasil Dikirim!"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Order ID : %s", StyleMuted.Render(m.lastOrder.ID[:8]+"...")))
		statusStyle := StyleAccent
		switch m.lastOrder.Status {
		case StatusFilled:
			statusStyle = StyleSuccess
		case StatusPartial:
			statusStyle = StyleWarning
		}
		lines = append(lines, fmt.Sprintf("  Status   : %s", statusStyle.Bold(true).Render(string(m.lastOrder.Status))))

		if m.lastResult != nil && len(m.lastResult.Trades) > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("  %s %d transaksi terjadi:", StyleSuccess.Bold(true).Render("⚡ Matched!"), len(m.lastResult.Trades)))
			for _, t := range m.lastResult.Trades {
				lines = append(lines, fmt.Sprintf("    → %s Lot @ %s",
					StyleAccent.Render(FmtNumber(t.QtyLot)),
					StyleSuccess.Render(FmtRupiah(t.Price)),
				))
			}
			lines = append(lines, fmt.Sprintf("  Total terisi: %s / %s Lot",
				StyleSuccess.Render(FmtNumber(m.lastResult.TotalFill)),
				StyleBold.Render(FmtNumber(m.lastOrder.QtyLot)),
			))
		} else {
			lines = append(lines, "  "+StyleMuted.Render("Order masuk antrean — menunggu counterpart."))
		}
		lines = append(lines, "")
		lines = append(lines, "  "+StyleAccent.Render("⏎ atau Esc — Kembali ke Market"))
	}

	if m.statusMsg != "" && m.statusIsErr {
		lines = append(lines, "")
		lines = append(lines, "  "+StyleError.Render("✗ "+m.statusMsg))
	}

	return strings.Join(lines, "\n")
}

// ── Portfolio view ────────────────────────────────────────────

func (m AppModel) renderPortfolio(height int) string {
	if m.trader == nil {
		return StyleMuted.Render("\n  Login terlebih dahulu (tekan L).\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+StyleTitle.Bold(true).Render(fmt.Sprintf("Portofolio — %s", m.trader.Username)))

	// Refresh trader cash
	if t, _ := m.repo.GetTraderByID(m.trader.ID); t != nil {
		m.trader = t
	}

	lines = append(lines, fmt.Sprintf("  %s %s  │  %s %s",
		StyleMuted.Render("Kas:"),
		StyleSuccess.Bold(true).Render(FmtRupiah(m.trader.CashBalance)),
		StyleMuted.Render("Nama:"),
		StyleBold.Render(m.trader.FullName),
	))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	if len(m.portfolio.items) == 0 {
		lines = append(lines, "\n  "+StyleMuted.Render("Portofolio saham kosong."))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "  "+StyleTableHeader.Render(fmt.Sprintf(
		"%-6s  %14s  %14s  %10s  %8s  %16s",
		"KODE", "HARGA PASAR", "RATA2 BELI", "P/L", "LOT", "NILAI",
	)))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	var totalMarket, totalCost int64
	for _, p := range m.portfolio.items {
		s, ok := m.portfolio.stocks[p.Ticker]
		if !ok {
			continue
		}
		marketVal := s.LastPrice * p.QtyLot * 100
		costVal := p.AvgPrice * p.QtyLot * 100
		pl := marketVal - costVal
		plPct := 0.0
		if costVal != 0 {
			plPct = float64(pl) / float64(costVal) * 100
		}
		totalMarket += marketVal
		totalCost += costVal

		plStyle := StyleSuccess
		if pl < 0 {
			plStyle = StyleError
		}

		lines = append(lines, fmt.Sprintf("  %s  %14s  %14s  %s  %8s  %16s",
			StyleAccent.Bold(true).Render(fmt.Sprintf("%-6s", p.Ticker)),
			PriceStyle(s.LastPrice, s.PrevClose).Render(FmtRupiah(s.LastPrice)),
			StyleMuted.Render(FmtRupiah(p.AvgPrice)),
			plStyle.Render(fmt.Sprintf("%+.2f%%", plPct)),
			StyleBold.Render(FmtNumber(p.QtyLot)),
			StyleCyan.Render(FmtBillions(marketVal)),
		))
	}

	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))
	totalPL := totalMarket - totalCost
	totalPct := 0.0
	if totalCost > 0 {
		totalPct = float64(totalPL) / float64(totalCost) * 100
	}
	plStyle := StyleSuccess
	if totalPL < 0 {
		plStyle = StyleError
	}
	lines = append(lines, fmt.Sprintf("  %s %s  │  %s %s  │  %s %s",
		StyleMuted.Render("Total Aset Saham:"),
		StyleCyan.Bold(true).Render(FmtBillions(totalMarket)),
		StyleMuted.Render("Total P/L:"),
		plStyle.Bold(true).Render(fmt.Sprintf("%+.2f%%", totalPct)),
		StyleMuted.Render("Total Kas+Aset:"),
		StyleSuccess.Bold(true).Render(FmtBillions(totalMarket+m.trader.CashBalance)),
	))

	return strings.Join(lines, "\n")
}

// ── Orders view ───────────────────────────────────────────────

func (m AppModel) renderOrders(height int) string {
	if m.trader == nil {
		return StyleMuted.Render("\n  Login terlebih dahulu (tekan L).\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+StyleTitle.Bold(true).Render("Riwayat Order"))
	lines = append(lines, "  "+StyleTableHeader.Render(fmt.Sprintf(
		"%-8s  %-6s  %-5s  %-7s  %14s  %8s  %8s  %10s",
		"ID", "KODE", "SISI", "TIPE", "HARGA", "QTY", "TERISI", "STATUS",
	)))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	if len(m.orders) == 0 {
		lines = append(lines, "\n  "+StyleMuted.Render("Belum ada order."))
		return strings.Join(lines, "\n")
	}

	for i, o := range m.orders {
		priceStr := FmtRupiah(o.Price)
		if o.OrderType == TypeMarket {
			priceStr = "MARKET"
		}

		sideStyle := StyleSuccess
		if o.Side == SideSell {
			sideStyle = StyleError
		}

		var statusStyle lipgloss.Style
		switch o.Status {
		case StatusFilled:
			statusStyle = StyleSuccess
		case StatusPartial:
			statusStyle = StyleWarning
		case StatusOpen:
			statusStyle = StyleAccent
		case StatusCancelled:
			statusStyle = StyleMuted
		default:
			statusStyle = StyleError
		}

		row := fmt.Sprintf("  %s  %-6s  %s  %-7s  %14s  %8s  %8s  %s",
			StyleMuted.Render(o.ID[:8]),
			StyleAccent.Render(o.Ticker),
			sideStyle.Render(fmt.Sprintf("%-5s", string(o.Side))),
			string(o.OrderType),
			priceStr,
			FmtNumber(o.QtyLot),
			FmtNumber(o.FilledLot),
			statusStyle.Render(string(o.Status)),
		)

		if i == m.ordersCursor {
			row = StyleSelectedRow.Width(m.width).Render(
				fmt.Sprintf("  %-8s  %-6s  %-5s  %-7s  %14s  %8s  %8s  %-10s",
					o.ID[:8], o.Ticker, string(o.Side), string(o.OrderType),
					priceStr, FmtNumber(o.QtyLot), FmtNumber(o.FilledLot), string(o.Status),
				),
			)
		}

		lines = append(lines, row)
	}

	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))
	lines = append(lines, "  "+StyleMuted.Render("↑↓ navigasi  c batalkan order terpilih"))

	return strings.Join(lines, "\n")
}

// ── Traders view ──────────────────────────────────────────────

func (m AppModel) renderTraders(height int) string {
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+StyleTitle.Bold(true).Render("Daftar Trader"))
	lines = append(lines, "  "+StyleTableHeader.Render(fmt.Sprintf("%-14s  %-28s  %20s  %s", "USERNAME", "NAMA LENGKAP", "KAS", "ID")))
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", m.width-4)))

	for _, t := range m.traders {
		active := t.ID == func() string {
			if m.trader != nil {
				return m.trader.ID
			}
			return ""
		}()

		userStyle := StyleMuted
		if active {
			userStyle = StyleAccent.Bold(true)
		}

		lines = append(lines, fmt.Sprintf("  %s  %-28s  %20s  %s",
			userStyle.Render(fmt.Sprintf("%-14s", t.Username)),
			TruncStr(t.FullName, 28),
			StyleSuccess.Render(FmtRupiah(t.CashBalance)),
			StyleMuted.Render(t.ID[:8]+"..."),
		))
	}

	return strings.Join(lines, "\n")
}

// ── Login overlay ─────────────────────────────────────────────

func (m AppModel) renderLoginOverlay() string {
	lf := m.loginForm

	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  "+StyleTitle.Bold(true).Render("🔐 Login"))
	lines = append(lines, "  "+StyleMuted.Render("Pilih trader atau ketik username:"))
	lines = append(lines, "")

	if len(lf.Traders) > 0 {
		lines = append(lines, "  "+StyleTableHeader.Render(fmt.Sprintf("%-14s  %-25s  %s", "USERNAME", "NAMA", "KAS")))
		lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", 60)))
		for i, t := range lf.Traders {
			row := fmt.Sprintf("  %-14s  %-25s  %s",
				t.Username, TruncStr(t.FullName, 25), FmtRupiah(t.CashBalance))
			if i == lf.Cursor {
				lines = append(lines, StyleSelectedRow.Render(row))
			} else {
				lines = append(lines, row)
			}
		}
	} else {
		lines = append(lines, "  "+StyleMuted.Render("Belum ada trader. Tekan ⏎ (Enter) untuk seed data demo."))
	}

	lines = append(lines, "")
	lines = append(lines, "  "+StyleMuted.Render("↑↓ navigasi  ⏎ pilih  Esc tutup"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(1, 2).
		Width(m.width - 10).
		Render(strings.Join(lines, "\n"))

	// Center vertically
	boxHeight := strings.Count(box, "\n") + 1
	topPad := (m.height - boxHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	return strings.Repeat("\n", topPad) + "     " + box
}
