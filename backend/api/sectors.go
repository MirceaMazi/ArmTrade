package api

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// sectorDefinition describes a market sector and its representative companies.
// The first three symbols are used for the homepage preview; the full list is
// used for the sector detail page.
type sectorDefinition struct {
	Slug      string
	Name      string
	Companies []string
}

var sectorDefinitions = []sectorDefinition{
	{"technology", "Technology", []string{"AAPL", "MSFT", "NVDA", "GOOGL", "META", "AVGO", "ORCL", "ADBE"}},
	{"energy", "Energy", []string{"XOM", "CVX", "COP", "SLB", "EOG", "PSX", "MPC", "OXY"}},
	{"healthcare", "Healthcare", []string{"UNH", "JNJ", "LLY", "ABBV", "MRK", "PFE", "TMO", "ABT"}},
	{"financials", "Financials", []string{"JPM", "BAC", "WFC", "GS", "MS", "C", "BLK", "SCHW"}},
	{"consumer", "Consumer", []string{"AMZN", "TSLA", "HD", "MCD", "NKE", "SBUX", "LOW", "TGT"}},
	{"industrials", "Industrials", []string{"CAT", "GE", "UPS", "BA", "HON", "RTX", "UNP", "DE"}},
}

func findSector(slug string) (sectorDefinition, bool) {
	for _, s := range sectorDefinitions {
		if s.Slug == slug {
			return s, true
		}
	}
	return sectorDefinition{}, false
}

// CompanyStat is the normalized price snapshot returned to the frontend.
type CompanyStat struct {
	Symbol     string  `json:"symbol"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	DayChange  float64 `json:"dayChange"`
	WeekChange float64 `json:"weekChange"`
	MarketCap  float64 `json:"marketCap"`
}

// SectorNewsItem is a normalized news headline for a sector.
type SectorNewsItem struct {
	Title     string `json:"title"`
	Source    string `json:"source"`
	Published int64  `json:"published"`
	URL       string `json:"url"`
}

// SectorPreview is one card on the homepage Market Sectors section.
type SectorPreview struct {
	Slug      string        `json:"slug"`
	Name      string        `json:"name"`
	AvgChange float64       `json:"avgChange"`
	Sentiment string        `json:"sentiment"`
	Companies []CompanyStat `json:"companies"`
}

// SectorDetail is the full payload for a single sector page.
type SectorDetail struct {
	Slug      string           `json:"slug"`
	Name      string           `json:"name"`
	AvgChange float64          `json:"avgChange"`
	Sentiment string           `json:"sentiment"`
	Companies []CompanyStat    `json:"companies"`
	News      []SectorNewsItem `json:"news"`
}

func sentimentFor(avgChange float64) string {
	switch {
	case avgChange > 0.1:
		return "up"
	case avgChange < -0.1:
		return "down"
	default:
		return "flat"
	}
}

// getCompanyStats fetches normalized price stats for the given symbols. Price,
// day change, market cap and name come from a single batched quote request; the
// 1-week performance is computed from each symbol's 5-day chart concurrently.
func getCompanyStats(symbols []string) []CompanyStat {
	if len(symbols) == 0 {
		return []CompanyStat{}
	}

	quotes := fetchQuotes(symbols)
	weekChanges := fetchWeekChanges(symbols)

	stats := make([]CompanyStat, 0, len(symbols))
	for i, sym := range symbols {
		stats = append(stats, buildCompanyStat(sym, quotes[sym], weekChanges[i]))
	}
	return stats
}

// fetchQuotes returns a map of symbol -> raw quote object from the batched quote endpoint.
func fetchQuotes(symbols []string) map[string]map[string]interface{} {
	quotes := map[string]map[string]interface{}{}
	quoteData, _ := yahooService.GetQuotes(strings.Join(symbols, ","))
	if quoteData == nil {
		return quotes
	}
	qr, ok := quoteData["quoteResponse"].(map[string]interface{})
	if !ok {
		return quotes
	}
	results, ok := qr["result"].([]interface{})
	if !ok {
		return quotes
	}
	for _, r := range results {
		q, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if sym, ok := q["symbol"].(string); ok {
			quotes[sym] = q
		}
	}
	return quotes
}

// fetchWeekChanges computes the trailing-week change for each symbol concurrently.
func fetchWeekChanges(symbols []string) []float64 {
	weekChanges := make([]float64, len(symbols))
	var wg sync.WaitGroup
	for i, sym := range symbols {
		wg.Add(1)
		go func(idx int, ticker string) {
			defer wg.Done()
			weekChanges[idx] = getTickerWeekChange(ticker)
		}(i, sym)
	}
	wg.Wait()
	return weekChanges
}

// buildCompanyStat normalizes a single company's quote into a CompanyStat,
// falling back to the chart endpoint when the quote endpoint is unavailable.
func buildCompanyStat(symbol string, quote map[string]interface{}, weekChange float64) CompanyStat {
	stat := CompanyStat{Symbol: symbol, WeekChange: weekChange}
	if quote != nil {
		stat.Name, _ = quote["shortName"].(string)
		if stat.Name == "" {
			stat.Name, _ = quote["longName"].(string)
		}
		stat.Price, _ = quote["regularMarketPrice"].(float64)
		stat.DayChange, _ = quote["regularMarketChangePercent"].(float64)
		stat.MarketCap, _ = quote["marketCap"].(float64)
	}
	if stat.Name == "" {
		stat.Name = symbol
	}
	if stat.Price == 0 {
		price, change := getTickerPrice(symbol)
		stat.Price = price
		stat.DayChange = change
	}
	return stat
}

func averageDayChange(stats []CompanyStat) float64 {
	if len(stats) == 0 {
		return 0
	}
	var sum float64
	for _, s := range stats {
		sum += s.DayChange
	}
	return sum / float64(len(stats))
}

func handleGetSectorsPreview(c *gin.Context) {
	previews := make([]SectorPreview, len(sectorDefinitions))
	var wg sync.WaitGroup

	for i, def := range sectorDefinitions {
		wg.Add(1)
		go func(idx int, d sectorDefinition) {
			defer wg.Done()
			top3 := d.Companies
			if len(top3) > 3 {
				top3 = top3[:3]
			}
			stats := getCompanyStats(top3)
			avg := averageDayChange(stats)
			previews[idx] = SectorPreview{
				Slug:      d.Slug,
				Name:      d.Name,
				AvgChange: avg,
				Sentiment: sentimentFor(avg),
				Companies: stats,
			}
		}(i, def)
	}

	wg.Wait()
	c.JSON(http.StatusOK, previews)
}

func handleGetSectorDetails(c *gin.Context) {
	slug := strings.ToLower(c.Param("sector"))
	def, ok := findSector(slug)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unknown sector"})
		return
	}

	var stats []CompanyStat
	var news []SectorNewsItem
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		stats = getCompanyStats(def.Companies)
	}()
	go func() {
		defer wg.Done()
		news = getSectorNews(def.Name)
	}()

	wg.Wait()

	avg := averageDayChange(stats)
	c.JSON(http.StatusOK, SectorDetail{
		Slug:      def.Slug,
		Name:      def.Name,
		AvgChange: avg,
		Sentiment: sentimentFor(avg),
		Companies: stats,
		News:      news,
	})
}

// getSectorNews fetches and normalizes recent news headlines for a sector.
func getSectorNews(sectorName string) []SectorNewsItem {
	query := sectorName + " stocks"
	data, err := yahooService.SearchNews(query, 8)
	if err != nil || data == nil {
		return []SectorNewsItem{}
	}

	newsArray, ok := data["news"].([]interface{})
	if !ok {
		return []SectorNewsItem{}
	}

	items := make([]SectorNewsItem, 0, len(newsArray))
	for _, raw := range newsArray {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := item["title"].(string)
		link, _ := item["link"].(string)
		if title == "" || link == "" {
			continue
		}
		source, _ := item["publisher"].(string)
		published, _ := item["providerPublishTime"].(float64)
		items = append(items, SectorNewsItem{
			Title:     title,
			Source:    source,
			Published: int64(published),
			URL:       link,
		})
	}
	return items
}

// getTickerWeekChange computes the percentage change over the trailing ~1 week
// using the 5-day daily chart (first available close vs. latest close).
func getTickerWeekChange(ticker string) float64 {
	chartData, err := yahooService.GetChart(ticker, "1d", "5d")
	if err != nil {
		return 0
	}

	closes := extractChartCloses(chartData)
	var first, last float64
	for _, v := range closes {
		if f, ok := v.(float64); ok {
			if first == 0 {
				first = f
			}
			last = f
		}
	}
	if first == 0 {
		return 0
	}
	return ((last - first) / first) * 100
}

// extractChartCloses navigates chart.result[0].indicators.quote[0].close.
func extractChartCloses(chartData map[string]interface{}) []interface{} {
	chart, ok := chartData["chart"].(map[string]interface{})
	if !ok {
		return nil
	}
	result, ok := chart["result"].([]interface{})
	if !ok || len(result) == 0 {
		return nil
	}
	res0, ok := result[0].(map[string]interface{})
	if !ok {
		return nil
	}
	indicators, ok := res0["indicators"].(map[string]interface{})
	if !ok {
		return nil
	}
	quoteArr, ok := indicators["quote"].([]interface{})
	if !ok || len(quoteArr) == 0 {
		return nil
	}
	quote, ok := quoteArr[0].(map[string]interface{})
	if !ok {
		return nil
	}
	closes, _ := quote["close"].([]interface{})
	return closes
}
