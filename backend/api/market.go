package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// Sector ETF tickers and their display names
var sectorETFs = []struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
}{
	{"XLK", "Technology"},
	{"XLF", "Financials"},
	{"XLV", "Healthcare"},
	{"XLE", "Energy"},
	{"XLI", "Industrials"},
	{"XLC", "Communication"},
	{"XLY", "Consumer Disc."},
	{"XLP", "Consumer Stap."},
	{"XLRE", "Real Estate"},
	{"XLU", "Utilities"},
	{"XLB", "Materials"},
}

// Macro indicator symbols
var macroSymbols = []struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}{
	{"^GSPC", "S&P 500"},
	{"^DJI", "Dow Jones"},
	{"^IXIC", "Nasdaq"},
	{"^VIX", "VIX"},
	{"GC=F", "Gold"},
	{"CL=F", "Crude Oil"},
	{"^TNX", "10Y Treasury"},
}

type SectorData struct {
	Ticker string  `json:"ticker"`
	Name   string  `json:"name"`
	Change float64 `json:"change"`
}

type MacroData struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

type MoverItem struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Change float64 `json:"change"`
}

type MoversResponse struct {
	Gainers []MoverItem `json:"gainers"`
	Losers  []MoverItem `json:"losers"`
	Active  []MoverItem `json:"active"`
}

func handleGetSectors(c *gin.Context) {
	var sectors []SectorData
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, etf := range sectorETFs {
		wg.Add(1)
		go func(ticker, name string) {
			defer wg.Done()
			_, change := getTickerPrice(ticker)
			mu.Lock()
			sectors = append(sectors, SectorData{
				Ticker: ticker,
				Name:   name,
				Change: change,
			})
			mu.Unlock()
		}(etf.Ticker, etf.Name)
	}

	wg.Wait()
	c.JSON(http.StatusOK, sectors)
}

func handleGetMacro(c *gin.Context) {
	var macros []MacroData
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, m := range macroSymbols {
		wg.Add(1)
		go func(symbol, name string) {
			defer wg.Done()
			price, change := getTickerPrice(symbol)
			mu.Lock()
			macros = append(macros, MacroData{
				Symbol: symbol,
				Name:   name,
				Price:  price,
				Change: change,
			})
			mu.Unlock()
		}(m.Symbol, m.Name)
	}

	wg.Wait()
	c.JSON(http.StatusOK, macros)
}

func handleGetMovers(c *gin.Context) {
	var response MoversResponse
	var wg sync.WaitGroup

	scrIDs := []struct {
		id     string
		target *[]MoverItem
	}{
		{"day_gainers", &response.Gainers},
		{"day_losers", &response.Losers},
		{"most_actives", &response.Active},
	}

	for _, scr := range scrIDs {
		wg.Add(1)
		go func(scrID string, target *[]MoverItem) {
			defer wg.Done()
			items := fetchScreener(scrID, 5)
			*target = items
		}(scr.id, scr.target)
	}

	wg.Wait()

	if response.Gainers == nil {
		response.Gainers = []MoverItem{}
	}
	if response.Losers == nil {
		response.Losers = []MoverItem{}
	}
	if response.Active == nil {
		response.Active = []MoverItem{}
	}

	c.JSON(http.StatusOK, response)
}

func fetchScreener(scrID string, count int) []MoverItem {
	url := "https://query1.finance.yahoo.com/v1/finance/screener/predefined/saved?formatted=true&lang=en-US&region=US&scrIds=" + scrID + "&count=5"
	data, err := yahooService.MakeRawRequest(url)
	if err != nil {
		return []MoverItem{}
	}

	finance, ok := data["finance"].(map[string]interface{})
	if !ok {
		return []MoverItem{}
	}
	results, ok := finance["result"].([]interface{})
	if !ok || len(results) == 0 {
		return []MoverItem{}
	}
	resultObj, ok := results[0].(map[string]interface{})
	if !ok {
		return []MoverItem{}
	}
	quotes, ok := resultObj["quotes"].([]interface{})
	if !ok {
		return []MoverItem{}
	}

	var items []MoverItem
	for _, q := range quotes {
		quote, ok := q.(map[string]interface{})
		if !ok {
			continue
		}
		symbol, _ := quote["symbol"].(string)
		name, _ := quote["shortName"].(string)

		var price float64
		var change float64
		if priceObj, ok := quote["regularMarketPrice"].(map[string]interface{}); ok {
			price, _ = priceObj["raw"].(float64)
		}
		if changeObj, ok := quote["regularMarketChangePercent"].(map[string]interface{}); ok {
			change, _ = changeObj["raw"].(float64)
		}

		items = append(items, MoverItem{
			Symbol: symbol,
			Name:   name,
			Price:  price,
			Change: change,
		})
	}

	return items
}
