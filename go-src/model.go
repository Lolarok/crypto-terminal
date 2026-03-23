package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─── Tab IDs ──────────────────────────────────────────
const (
	tabOverview = iota
	tabTrending
	tabDetail
	tabGlobal
)

var tabNames = []string{"📊 Overview", "🔥 Trending", "🔍 Detail", "🌍 Global"}

// ─── Messages ─────────────────────────────────────────
type coinsLoadedMsg struct {
	coins []CoinMarket
	err   error
}
type trendingLoadedMsg struct {
	coins []TrendingCoin
	err   error
}
type globalLoadedMsg struct {
	data  *GlobalData
	fg    *FearGreed
	err   error
}

// ─── Model ────────────────────────────────────────────
type model struct {
	width    int
	height   int
	tab      int
	loading  bool
	err      error

	// Data
	coins      []CoinMarket
	scored     []ScoredCoin
	trending   []TrendingCoin
	global     *GlobalData
	fearGreed  *FearGreed
	lastUpdate time.Time

	// Detail view
	selectedIdx int

	// UI
	spinner    spinner.Model
	autoRefresh bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = spinnerStyle

	return model{
		tab:         tabOverview,
		loading:     true,
		spinner:     s,
		autoRefresh: true,
		selectedIdx: 0,
	}
}

// ─── Init ─────────────────────────────────────────────
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadCoins(),
		loadTrending(),
		loadGlobal(),
	)
}

// ─── Commands ─────────────────────────────────────────
func loadCoins() tea.Cmd {
	return func() tea.Msg {
		coins, err := FetchTopCoins(50)
		return coinsLoadedMsg{coins: coins, err: err}
	}
}

func loadTrending() tea.Cmd {
	return func() tea.Msg {
		coins, err := FetchTrending()
		return trendingLoadedMsg{coins: coins, err: err}
	}
}

func loadGlobal() tea.Cmd {
	return func() tea.Msg {
		data, err := FetchGlobal()
		var fg *FearGreed
		if err == nil {
			fg, _ = FetchFearGreed()
		}
		return globalLoadedMsg{data: data, fg: fg, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(60*time.Second, func(t time.Time) tea.Msg {
		return coinsLoadedMsg{} // trigger reload
	})
}

// ─── Update ───────────────────────────────────────────
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % len(tabNames)
		case "shift+tab", "left", "h":
			m.tab = (m.tab - 1 + len(tabNames)) % len(tabNames)
		case "1":
			m.tab = tabOverview
		case "2":
			m.tab = tabTrending
		case "3":
			m.tab = tabDetail
		case "4":
			m.tab = tabGlobal
		case "r":
			m.loading = true
			return m, tea.Batch(loadCoins(), loadTrending(), loadGlobal())
		case "j", "down":
			if m.tab == tabOverview && m.selectedIdx < len(m.scored)-1 {
				m.selectedIdx++
			}
		case "k", "up":
			if m.tab == tabOverview && m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "enter":
			if m.tab == tabOverview && m.selectedIdx < len(m.scored) {
				m.tab = tabDetail
			}
		case "a":
			m.autoRefresh = !m.autoRefresh
		}

	case coinsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else if msg.coins != nil {
			m.coins = msg.coins
			m.scored = make([]ScoredCoin, len(msg.coins))
			for i, c := range msg.coins {
				m.scored[i] = computeScore(c)
			}
			m.lastUpdate = time.Now()
			m.loading = false
			m.err = nil
		}

	case trendingLoadedMsg:
		if msg.err == nil {
			m.trending = msg.coins
		}

	case globalLoadedMsg:
		if msg.err == nil {
			m.global = msg.data
			m.fearGreed = msg.fg
		}
	}

	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// ─── View ─────────────────────────────────────────────
func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Header
	header := m.renderHeader()

	// Tabs
	tabs := m.renderTabs()

	// Content
	var content string
	switch m.tab {
	case tabOverview:
		content = m.renderOverview()
	case tabTrending:
		content = m.renderTrending()
	case tabDetail:
		content = m.renderDetail()
	case tabGlobal:
		content = m.renderGlobal()
	}

	// Footer
	footer := m.renderFooter()

	// Compose
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		content,
		footer,
	)
}

// ─── Header ───────────────────────────────────────────
func (m model) renderHeader() string {
	logo := titleStyle.Render("⚡ CRYPTO TERMINAL")
	status := ""
	if m.loading {
		status = " " + m.spinner.View() + " loading..."
	} else if !m.lastUpdate.IsZero() {
		status = fmt.Sprintf(" • updated %s", m.lastUpdate.Format("15:04:05"))
	}
	if m.autoRefresh {
		status += " • auto-refresh ON"
	}

	line := lipgloss.JoinHorizontal(lipgloss.Center,
		logo,
		subtitleStyle.Render(status),
	)

	width := lipgloss.Width(line)
	padding := m.width - width
	if padding > 0 {
		line += strings.Repeat(" ", padding)
	}

	return headerStyle.
		Width(m.width).
		Render(line)
}

// ─── Tabs ─────────────────────────────────────────────
func (m model) renderTabs() string {
	rendered := make([]string, len(tabNames))
	for i, name := range tabNames {
		if i == m.tab {
			rendered[i] = tabActiveStyle.Render(name)
		} else {
			rendered[i] = tabInactiveStyle.Render(name)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

// ─── Footer ───────────────────────────────────────────
func (m model) renderFooter() string {
	keys := []string{
		"tab/→ next tab",
		"← prev tab",
		"↑↓ navigate",
		"enter select",
		"r refresh",
		"a auto-refresh",
		"q quit",
	}
	return helpStyle.Render(strings.Join(keys, " • "))
}

// ─── Overview ─────────────────────────────────────────
func (m model) renderOverview() string {
	if len(m.scored) == 0 {
		return "\n  " + m.spinner.View() + " Loading market data...\n"
	}

	// Table header
	cols := []string{
		padRight("#", 4),
		padRight("Coin", 18),
		padRight("Price", 14),
		padRight("1h", 9),
		padRight("24h", 9),
		padRight("7d", 9),
		padRight("MCap", 12),
		padRight("Score", 7),
		"7d Chart",
	}

	headerLine := lipgloss.JoinHorizontal(lipgloss.Top,
		cellDimStyle.Render(cols[0]),
		cellDimStyle.Render(cols[1]),
		cellDimStyle.Render(cols[2]),
		cellDimStyle.Render(cols[3]),
		cellDimStyle.Render(cols[4]),
		cellDimStyle.Render(cols[5]),
		cellDimStyle.Render(cols[6]),
		cellDimStyle.Render(cols[7]),
		cellDimStyle.Render(cols[8]),
	)

	rows := []string{headerLine}

	// Calculate visible rows
	maxRows := m.height - 10
	if maxRows < 5 {
		maxRows = 5
	}

	sparkWidth := m.width - 90
	if sparkWidth < 10 {
		sparkWidth = 10
	}

	for i, sc := range m.scored {
		if i >= maxRows {
			break
		}

		c := sc.Coin
		selected := i == m.selectedIdx

		rank := fmt.Sprintf("%d", c.MarketCapRank)
		name := c.Name
		if len(name) > 16 {
			name = name[:14] + ".."
		}
		symbol := strings.ToUpper(c.Symbol)

		price := formatUSD(c.CurrentPrice)
		h1 := formatPct(c.PriceChangePct1h)
		h24 := formatPct(c.PriceChangePct24h)
		d7 := formatPct(c.PriceChangePct7d)
		mcap := formatUSD(c.MarketCap)
		score := fmt.Sprintf("%.0f", sc.Score)

		spark := renderSparkline(c.SparklineIn7d.Price, sparkWidth)

		line := lipgloss.JoinHorizontal(lipgloss.Top,
			cellDimStyle.Render(padRight(rank, 4)),
			coinNameStyle.Render(padRight(name, 14))+
				coinSymbolStyle.Render(padRight(symbol, 4)),
			cellStyle.Render(padRight(price, 14)),
			changeStyle(c.PriceChangePct1h).Render(padRight(h1, 9)),
			changeStyle(c.PriceChangePct24h).Render(padRight(h24, 9)),
			changeStyle(c.PriceChangePct7d).Render(padRight(d7, 9)),
			cellDimStyle.Render(padRight(mcap, 12)),
			scoreStyle(sc.Score).Render(padRight(score, 7)),
			sparklineColor(c.SparklineIn7d).Render(spark),
		)

		if selected {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#1a1a2e")).
				Render("▸ " + line)
		} else {
			line = "  " + line
		}

		rows = append(rows, line)
	}

	return "\n" + strings.Join(rows, "\n") + "\n"
}

func sparklineColor(sl Sparkline) lipgloss.Style {
	if len(sl.Price) < 2 {
		return lipgloss.NewStyle().Foreground(dim)
	}
	if sl.Price[len(sl.Price)-1] >= sl.Price[0] {
		return lipgloss.NewStyle().Foreground(green)
	}
	return lipgloss.NewStyle().Foreground(red)
}

// ─── Trending ─────────────────────────────────────────
func (m model) renderTrending() string {
	if len(m.trending) == 0 {
		return "\n  " + m.spinner.View() + " Loading trending...\n"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(sectionTitle.Render("🔥 Top Trending on CoinGecko"))
	b.WriteString("\n\n")

	for i, tc := range m.trending {
		item := tc.Item
		rank := fmt.Sprintf("#%d", i+1)

		line := fmt.Sprintf("  %s  %s %s",
			scoreStyle(float64(70+item.Score)).Render(padRight(rank, 4)),
			coinNameStyle.Render(padRight(item.Name, 25)),
			coinSymbolStyle.Render(strings.ToUpper(item.Symbol)),
		)

		if item.MarketCapRank > 0 {
			line += cellDimStyle.Render(fmt.Sprintf("  MCap Rank: %d", item.MarketCapRank))
		}

		b.WriteString(line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(cellDimStyle.Render("  Data refreshes every 60s • Source: CoinGecko"))
	return b.String()
}

// ─── Detail ───────────────────────────────────────────
func (m model) renderDetail() string {
	if len(m.scored) == 0 {
		return "\n  No data loaded. Press 'r' to refresh.\n"
	}

	idx := m.selectedIdx
	if idx >= len(m.scored) {
		idx = 0
	}

	sc := m.scored[idx]
	c := sc.Coin

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(sectionTitle.Render(fmt.Sprintf("🔍 %s (%s) — Detailed Analysis", c.Name, strings.ToUpper(c.Symbol))))
	b.WriteString("\n\n")

	// Price info box
	priceInfo := fmt.Sprintf(
		"Price: %s\nMCap: %s\nVolume 24h: %s\nRank: #%d",
		formatUSD(c.CurrentPrice),
		formatUSD(c.MarketCap),
		formatUSD(c.TotalVolume),
		c.MarketCapRank,
	)

	// Change info box
	changeInfo := fmt.Sprintf(
		"1h:  %s\n24h: %s\n7d:  %s\n30d: %s",
		formatPct(c.PriceChangePct1h),
		formatPct(c.PriceChangePct24h),
		formatPct(c.PriceChangePct7d),
		formatPct(c.PriceChangePct30d),
	)

	left := borderBox.Render(priceInfo)
	right := borderBox.Render(changeInfo)

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right))
	b.WriteString("\n\n")

	// ATH
	athInfo := fmt.Sprintf("ATH: %s (%.1f%% from ATH)", formatUSD(c.ATH), math.Abs(c.ATHChangePct))
	b.WriteString(borderBox.Render(athInfo))
	b.WriteString("\n\n")

	// Score breakdown
	b.WriteString(sectionTitle.Render("📊 Score Breakdown"))
	b.WriteString("\n")

	for signal, value := range sc.Signals {
		label := padRight(signal, 20)
		bar := renderBar(value, 20, 20)
		b.WriteString(fmt.Sprintf("  %s %s %.1f\n",
			cellDimStyle.Render(label),
			bar,
			value,
		))
	}

	b.WriteString(fmt.Sprintf("\n  %s %s\n",
		coinNameStyle.Render(padRight("TOTAL SCORE", 20)),
		scoreStyle(sc.Score).Render(fmt.Sprintf("%.1f / 100", sc.Score)),
	))

	// Sparkline
	if len(c.SparklineIn7d.Price) > 0 {
		b.WriteString("\n")
		b.WriteString(sectionTitle.Render("📈 7-Day Price"))
		b.WriteString("\n")
		spark := renderSparkline(c.SparklineIn7d.Price, m.width-10)
		b.WriteString("  " + sparklineColor(c.SparklineIn7d).Render(spark))
		b.WriteString("\n")
	}

	// Navigation hint
	b.WriteString("\n")
	b.WriteString(cellDimStyle.Render("  ↑/↓ to browse • tab to switch views"))

	return b.String()
}

func renderBar(value float64, max float64, width int) string {
	if max <= 0 {
		max = 1
	}
	filled := int(value / max * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	if float64(value)/float64(max) > 0.7 {
		return lipgloss.NewStyle().Foreground(green).Render(bar)
	}
	if float64(value)/float64(max) > 0.4 {
		return lipgloss.NewStyle().Foreground(yellow).Render(bar)
	}
	return lipgloss.NewStyle().Foreground(red).Render(bar)
}

// ─── Global ───────────────────────────────────────────
func (m model) renderGlobal() string {
	if m.global == nil {
		return "\n  " + m.spinner.View() + " Loading global data...\n"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(sectionTitle.Render("🌍 Global Market Overview"))
	b.WriteString("\n\n")

	g := m.global.Data

	// Market overview
	overview := fmt.Sprintf(
		"Total Market Cap: %s\n24h Change: %s\nActive Cryptos: %s\nTotal 24h Volume: %s",
		formatUSD(g.TotalMarketCap),
		formatPct(g.MarketCapChangePct),
		formatInt(g.ActiveCryptos),
		formatUSD(g.TotalVolume),
	)
	b.WriteString(borderBox.Render(overview))
	b.WriteString("\n\n")

	// Fear & Greed
	if m.fearGreed != nil {
		fgValue := m.fearGreed.Data.Value
		fgClass := m.fearGreed.Data.Class

		fgStyle := scoreHighStyle
		switch strings.ToLower(fgClass) {
		case "fear", "extreme fear":
			fgStyle = priceDownStyle
		case "greed", "extreme greed":
			fgStyle = priceUpStyle
		case "neutral":
			fgStyle = scoreMedStyle
		}

		fgBox := fmt.Sprintf("Fear & Greed Index: %s — %s", fgValue, fgClass)
		b.WriteString(fgStyle.Render("  ⚡ " + fgBox))
		b.WriteString("\n\n")
	}

	// Dominance
	b.WriteString(sectionTitle.Render("📊 Market Dominance"))
	b.WriteString("\n")

	dominance := []string{"btc", "eth", "bnb", "sol", "xrp"}
	for _, coin := range dominance {
		if pct, ok := g.MarketCapPct[coin]; ok {
			bar := renderBar(pct, 100, 30)
			label := strings.ToUpper(coin)
			b.WriteString(fmt.Sprintf("  %s %s %.1f%%\n",
				coinNameStyle.Render(padRight(label, 5)),
				bar,
				pct,
			))
		}
	}

	return b.String()
}

// ─── Helpers ──────────────────────────────────────────
func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}
