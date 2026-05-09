package api

import (
	"net/http"
	"sort"

	"armtrade-backend/services"

	"github.com/gin-gonic/gin"
)

var yahooService = services.NewYahooFinanceService()
var armandService = services.NewArmandService(yahooService)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/search", handleSearch)
		api.GET("/chart/:ticker", handleGetChart)
		api.GET("/fundamentals/:ticker", handleGetFundamentals)
		api.GET("/news/:ticker", handleGetNews)
		api.GET("/dividends/:ticker", handleGetDividends)

		// Armand AI endpoints
		api.POST("/armand/analyze", handleArmandAnalysis)
		api.POST("/armand/screener", handleScreener)
	}
}

func handleSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	results, err := yahooService.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

func handleGetChart(c *gin.Context) {
	ticker := c.Param("ticker")
	interval := c.DefaultQuery("interval", "1d")
	trange := c.DefaultQuery("range", "1mo")

	results, err := yahooService.GetChart(ticker, interval, trange)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

func handleGetFundamentals(c *gin.Context) {
	ticker := c.Param("ticker")

	results, err := yahooService.GetFundamentals(ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

func handleArmandAnalysis(c *gin.Context) {
	var req services.AnalyzeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	analysis, err := armandService.Analyze(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

func handleScreener(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	results, err := armandService.Screen(req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}

type NewsResponseItem struct {
	services.NewsItem
	Sentiment string `json:"sentiment"`
	Reason    string `json:"reason"`
}

func handleGetNews(c *gin.Context) {
	ticker := c.Param("ticker")

	// 1. Fetch news from Yahoo
	newsData, err := yahooService.GetNews(ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract news array
	newsArray, ok := newsData["news"].([]interface{})
	if !ok || len(newsArray) == 0 {
		c.JSON(http.StatusOK, []NewsResponseItem{})
		return
	}

	var newsItems []services.NewsItem
	for _, itemInterface := range newsArray {
		item, ok := itemInterface.(map[string]interface{})
		if !ok {
			continue
		}
		title, _ := item["title"].(string)
		link, _ := item["link"].(string)
		if title != "" && link != "" {
			newsItems = append(newsItems, services.NewsItem{Title: title, Link: link})
		}
		if len(newsItems) >= 3 { // Limit to 5 for speed and layout
			break
		}
	}

	if len(newsItems) == 0 {
		c.JSON(http.StatusOK, []NewsResponseItem{})
		return
	}

	// 2. Analyze sentiment with Armand
	analysis, err := armandService.AnalyzeNewsSentiment(newsItems)
	
	// 3. Combine results
	var response []NewsResponseItem
	for i, item := range newsItems {
		sentiment := "Neutral"
		reason := "Could not analyze"
		if err == nil && i < len(analysis) {
			sentiment = analysis[i].Sentiment
			reason = analysis[i].Reason
		}
		
		response = append(response, NewsResponseItem{
			NewsItem:  item,
			Sentiment: sentiment,
			Reason:    reason,
		})
	}

	c.JSON(http.StatusOK, response)
}

func handleGetDividends(c *gin.Context) {
	ticker := c.Param("ticker")

	data, err := yahooService.GetDividends(ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Navigate the response JSON: chart -> result -> [0] -> events -> dividends
	var dividendsList []map[string]interface{}
	
	chart, ok := data["chart"].(map[string]interface{})
	if ok {
		result, ok := chart["result"].([]interface{})
		if ok && len(result) > 0 {
			resObj, ok := result[0].(map[string]interface{})
			if ok {
				events, ok := resObj["events"].(map[string]interface{})
				if ok {
					dividends, ok := events["dividends"].(map[string]interface{})
					if ok {
						for _, div := range dividends {
							divObj, ok := div.(map[string]interface{})
							if ok {
								dividendsList = append(dividendsList, divObj)
							}
						}
					}
				}
			}
		}
	}

	if dividendsList == nil {
		dividendsList = []map[string]interface{}{}
	}

	// Sort dividends by date descending
	sort.Slice(dividendsList, func(i, j int) bool {
		dateI, _ := dividendsList[i]["date"].(float64)
		dateJ, _ := dividendsList[j]["date"].(float64)
		return dateI > dateJ
	})

	c.JSON(http.StatusOK, dividendsList)
}
