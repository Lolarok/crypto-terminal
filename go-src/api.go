package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// ─── CoinGecko Models ─────────────────────────────────

type CoinMarket struct {
	ID                 string    `json:"id"`
	Symbol             string    `json:"symbol"`
	Name               string    `json:"name"`
	Image              string    `json:"image"`
	CurrentPrice       float64   `json:"current_price"`
	MarketCap          float64   `json:"market_cap"`
	MarketCapRank      int       `json:"market_cap_rank"`
	TotalVolume        float64   `json:"total_volume"`
	PriceChangePct1h   float64   `json:"price_change_percentage_1h_in_currency"`
	PriceChangePct24h  float64   `json:"price_change_percentage_24h"`
	PriceChangePct7d   float64   `json:"price_change_percentage_7d_in_currency"`
	PriceChangePct30d  float64   `json:"price_change_percentage_30d_in_currency"`
	ATH                float64   `json:"ath"`
	ATHChangePct       float64   `json:"ath_change_percentage"`
	SparklineIn7d      Sparkline `json:"sparkline_in_7d"`
}

type Sparkline struct {
	Price []float64 `json:"price"`
}

type TrendingCoin struct {
	Item struct {
		ID           string  `json:"id"`
		Name         string  `json:"name"`
		Symbol       string  `json:"symbol"`
		MarketCapRank int    `json:"market_cap_rank"`
		PriceBTC     float64 `json:"price_btc"`
		Score        int     `json:"score"`
	} `json:"item"`
}

type TrendingResponse struct {
	Coins []TrendingCoin `json:"coins"`
}

type GlobalData struct {
	Data struct {
		TotalMarketCap    float64            `json:"total_market_cap"`
		TotalVolume       float64            `json:"total_volume"`
		MarketCapPct      map[string]float64 `json:"market_cap_percentage"`
		ActiveCryptos     int                `json:"active_cryptocurrencies"`
		MarketCapChangePct float64           `json:"market_cap_change_percentage_24h_usd"`
	} `json:"data"`
}

type FearGreed struct {
	Data struct {
		Value string `json:"value"`
		Class string `json:"value_classification"`
	} `json:"data"`
}

// ─── Computed Score ────────────────────────────────────

type ScoredCoin struct {
	Coin          CoinMarket
	Score         float64
	Signals       map[string]float64
}

func computeScore(coin CoinMarket) ScoredCoin {
	signals := make(map[string]float64)

	// 24h momentum (0-15)
	m24h := math.Min(math.Max(coin.PriceChangePct24h+15, 0), 30) / 30 * 15
	signals["momentum_24h"] = math.Round(m24h*10) / 10

	// 7d momentum (0-20)
	m7d := math.Min(math.Max(coin.PriceChangePct7d+25, 0), 50) / 50 * 20
	signals["momentum_7d"] = math.Round(m7d*10) / 10

	// ATH discount (0-20): deeper discount = more points
	athDisc := math.Abs(coin.ATHChangePct)
	athScore := math.Min(athDisc/80*20, 20)
	signals["ath_discount"] = math.Round(athScore*10) / 10

	// Volume/MCap ratio (0-15)
	if coin.MarketCap > 0 {
		volRatio := coin.TotalVolume / coin.MarketCap
		volScore := math.Min(volRatio*100, 15)
		signals["volume_activity"] = math.Round(volScore*10) / 10
	}

	// MCap upside potential (0-15) — smaller = more potential
	if coin.MarketCap > 0 {
		mcapScore := 15.0
		if coin.MarketCap > 1e10 {
			mcapScore = 3
		} else if coin.MarketCap > 1e9 {
			mcapScore = 6
		} else if coin.MarketCap > 5e8 {
			mcapScore = 9
		} else if coin.MarketCap > 1e8 {
			mcapScore = 12
		}
		signals["mcap_upside"] = mcapScore
	}

	// 1h stability bonus (0-15)
	h1 := math.Abs(coin.PriceChangePct1h)
	stability := math.Max(0, 15-h1*2)
	signals["stability"] = math.Round(stability*10) / 10

	total := 0.0
	for _, v := range signals {
		total += v
	}

	return ScoredCoin{
		Coin:    coin,
		Score:   math.Round(total*10) / 10,
		Signals: signals,
	}
}

// ─── API Client ───────────────────────────────────────

var httpClient = &http.Client{Timeout: 15 * time.Second}

func fetchJSON(url string, target interface{}) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	return json.Unmarshal(body, target)
}

func FetchTopCoins(limit int) ([]CoinMarket, error) {
	url := fmt.Sprintf(
		"https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=%d&page=1&sparkline=true&price_change_percentage=1h,7d,30d",
		limit,
	)
	var coins []CoinMarket
	err := fetchJSON(url, &coins)
	return coins, err
}

func FetchTrending() ([]TrendingCoin, error) {
	var resp TrendingResponse
	err := fetchJSON("https://api.coingecko.com/api/v3/search/trending", &resp)
	return resp.Coins, err
}

func FetchGlobal() (*GlobalData, error) {
	var data GlobalData
	err := fetchJSON("https://api.coingecko.com/api/v3/global", &data)
	return &data, err
}

func FetchFearGreed() (*FearGreed, error) {
	var data FearGreed
	err := fetchJSON("https://api.alternative.me/fng/?limit=1", &data)
	return &data, err
}

// ─── Sparkline Renderer ───────────────────────────────

func renderSparkline(prices []float64, width int) string {
	if len(prices) < 2 {
		return ""
	}

	// Downsample to width
	step := len(prices) / width
	if step < 1 {
		step = 1
	}

	sampled := make([]float64, 0, width)
	for i := 0; i < len(prices) && len(sampled) < width; i += step {
		sampled = append(sampled, prices[i])
	}

	// Find min/max
	min, max := sampled[0], sampled[0]
	for _, p := range sampled {
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}

	// Braille blocks for resolution
	blocks := []rune{' ', '⣀', '⣄', '⣆', '⣇', '⣏', '⣟', '⣿'}

	result := make([]rune, 0, len(sampled))
	for _, p := range sampled {
		normalized := 0.0
		if max > min {
			normalized = (p - min) / (max - min)
		}
		idx := int(normalized * float64(len(blocks)-1))
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		result = append(result, blocks[idx])
	}

	return string(result)
}

// ─── Formatters ───────────────────────────────────────

func formatUSD(v float64) string {
	if v >= 1e12 {
		return fmt.Sprintf("$%.2fT", v/1e12)
	}
	if v >= 1e9 {
		return fmt.Sprintf("$%.2fB", v/1e9)
	}
	if v >= 1e6 {
		return fmt.Sprintf("$%.2fM", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("$%.1fK", v/1e3)
	}
	if v >= 1 {
		return fmt.Sprintf("$%.2f", v)
	}
	return fmt.Sprintf("$%.4f", v)
}

func formatPct(v float64) string {
	sign := ""
	if v >= 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.2f%%", sign, v)
}

func formatInt(v int) string {
	if v >= 1e6 {
		return fmt.Sprintf("%.1fM", float64(v)/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("%.1fK", float64(v)/1e3)
	}
	return fmt.Sprintf("%d", v)
}
