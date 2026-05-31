package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const dateLayout = "2006-01-02"

// IpoItem is a normalized upcoming IPO entry.
type IpoItem struct {
	Company    string  `json:"company"`
	Ticker     string  `json:"ticker"`
	Date       string  `json:"date"`
	Exchange   string  `json:"exchange"`
	PriceFrom  float64 `json:"priceFrom"`
	PriceTo    float64 `json:"priceTo"`
	PriceRange string  `json:"priceRange"`
}

// EarningsItem is a normalized upcoming earnings report entry.
type EarningsItem struct {
	Company     string  `json:"company"`
	Ticker      string  `json:"ticker"`
	Date        string  `json:"date"`
	Time        string  `json:"time"` // pre-market / after-hours / during-market
	EpsEstimate float64 `json:"epsEstimate"`
	HasEps      bool    `json:"hasEps"`
	EpsForward  bool    `json:"epsForward"` // true when EpsEstimate is the forward annual EPS, not the quarterly consensus
}

// calendarRows extracts the rows from a Yahoo visualization response. Each row is
// returned as a map keyed by column id.
func calendarRows(data map[string]interface{}) ([]map[string]interface{}, []string) {
	doc := firstDocument(data)
	if doc == nil {
		return nil, nil
	}

	columns := documentColumns(doc)
	rows := documentRows(doc, columns)
	return rows, columns
}

// firstDocument navigates finance.result[0].documents[0] of a visualization payload.
func firstDocument(data map[string]interface{}) map[string]interface{} {
	finance, ok := data["finance"].(map[string]interface{})
	if !ok {
		return nil
	}
	resultArr, ok := finance["result"].([]interface{})
	if !ok || len(resultArr) == 0 {
		return nil
	}
	result0, ok := resultArr[0].(map[string]interface{})
	if !ok {
		return nil
	}
	documents, ok := result0["documents"].([]interface{})
	if !ok || len(documents) == 0 {
		return nil
	}
	doc0, _ := documents[0].(map[string]interface{})
	return doc0
}

func documentColumns(doc map[string]interface{}) []string {
	cols, ok := doc["columns"].([]interface{})
	if !ok {
		return nil
	}
	columns := make([]string, 0, len(cols))
	for _, col := range cols {
		cm, ok := col.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := cm["id"].(string)
		columns = append(columns, id)
	}
	return columns
}

func documentRows(doc map[string]interface{}, columns []string) []map[string]interface{} {
	rawRows, ok := doc["rows"].([]interface{})
	if !ok {
		return nil
	}
	rows := make([]map[string]interface{}, 0, len(rawRows))
	for _, r := range rawRows {
		cells, ok := r.([]interface{})
		if !ok {
			continue
		}
		row := map[string]interface{}{}
		for i, cell := range cells {
			if i < len(columns) {
				row[columns[i]] = cell
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func cellString(row map[string]interface{}, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	return ""
}

func cellFloat(row map[string]interface{}, key string) (float64, bool) {
	if v, ok := row[key].(float64); ok {
		return v, true
	}
	return 0, false
}

// dateOnly converts a Yahoo datetime string (e.g. "2024-05-01T09:00:00.000Z")
// to a YYYY-MM-DD date.
func dateOnly(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func mapEarningsTime(t string) string {
	switch t {
	case "BMO":
		return "pre-market"
	case "AMC":
		return "after-hours"
	case "TAS", "TNS":
		return "during-market"
	default:
		return "unspecified"
	}
}

func handleGetIPOs(c *gin.Context) {
	now := time.Now()
	from := now.Format(dateLayout)
	to := now.AddDate(0, 1, 0).Format(dateLayout)

	data, err := yahooService.GetIPOCalendar(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch IPO calendar"})
		return
	}

	rows, _ := calendarRows(data)
	ipos := make([]IpoItem, 0, len(rows))
	for _, row := range rows {
		priceFrom, _ := cellFloat(row, "pricefrom")
		priceTo, _ := cellFloat(row, "priceto")
		offer, hasOffer := cellFloat(row, "offerprice")
		if priceFrom == 0 && hasOffer {
			priceFrom = offer
			priceTo = offer
		}

		ipos = append(ipos, IpoItem{
			Company:    cellString(row, "companyshortname"),
			Ticker:     cellString(row, "ticker"),
			Date:       dateOnly(cellString(row, "startdatetime")),
			Exchange:   cellString(row, "exchange"),
			PriceFrom:  priceFrom,
			PriceTo:    priceTo,
			PriceRange: formatPriceRange(priceFrom, priceTo),
		})
	}

	c.JSON(http.StatusOK, ipos)
}

func handleGetEarnings(c *gin.Context) {
	now := time.Now()
	from := now.Format(dateLayout)
	to := now.AddDate(0, 3, 0).Format(dateLayout)

	data, err := yahooService.GetEarningsCalendar(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch earnings calendar"})
		return
	}

	rows, _ := calendarRows(data)
	earnings := make([]EarningsItem, 0, len(rows))
	tickers := make([]string, 0, len(rows))
	for _, row := range rows {
		eps, hasEps := cellFloat(row, "epsestimate")
		ticker := cellString(row, "ticker")
		tickers = append(tickers, ticker)
		earnings = append(earnings, EarningsItem{
			Company:     cellString(row, "companyshortname"),
			Ticker:      ticker,
			Date:        dateOnly(cellString(row, "startdatetime")),
			Time:        mapEarningsTime(cellString(row, "startdatetimetype")),
			EpsEstimate: eps,
			HasEps:      hasEps,
		})
	}

	// The visualization calendar rarely populates a quarterly EPS estimate for
	// future-dated reports, so fall back to the forward annual EPS from the
	// batched quote endpoint to give users a meaningful number.
	quotes := fetchQuotes(tickers)
	for i := range earnings {
		if earnings[i].HasEps {
			continue
		}
		q, ok := quotes[earnings[i].Ticker]
		if !ok {
			continue
		}
		if fwd, ok := q["epsForward"].(float64); ok && fwd != 0 {
			earnings[i].EpsEstimate = fwd
			earnings[i].HasEps = true
			earnings[i].EpsForward = true
		}
	}

	c.JSON(http.StatusOK, earnings)
}

func formatPriceRange(from, to float64) string {
	if from == 0 && to == 0 {
		return ""
	}
	if from == to || to == 0 {
		return formatDollar(from)
	}
	return formatDollar(from) + " – " + formatDollar(to)
}

func formatDollar(v float64) string {
	if v == 0 {
		return ""
	}
	return "$" + strconv.FormatFloat(v, 'f', 2, 64)
}
