package api

import (
	"context"
	"net/http"
	"regexp"
	"sort"
	"time"

	"armtrade-backend/db"
	"armtrade-backend/models"
	"armtrade-backend/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var yahooService = services.NewYahooFinanceService()
var armandService = services.NewArmandService(yahooService)

var validTicker = regexp.MustCompile(`^[A-Za-z0-9.\-^=]{1,20}$`)

func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Health check
		api.GET("/health", handleHealth)

		// WebSocket for live prices
		api.GET("/ws/prices", handleWebSocket)

		// Public routes
		api.GET("/search", handleSearch)
		api.GET("/chart/:ticker", validateTicker(), handleGetChart)
		api.GET("/fundamentals/:ticker", validateTicker(), handleGetFundamentals)
		api.GET("/news/:ticker", validateTicker(), handleGetNews)
		api.GET("/dividends/:ticker", validateTicker(), handleGetDividends)

		// Auth routes (public)
		api.POST("/auth/register", handleRegister)
		api.POST("/auth/login", handleLogin)

		// Armand AI endpoints (public, rate-limited)
		armand := api.Group("/armand")
		armand.Use(RateLimitMiddleware())
		{
			armand.POST("/analyze", handleArmandAnalysis)
			armand.POST("/screener", handleScreener)
			armand.POST("/compare", handleCompareStocks)
			armand.POST("/earnings", handleSummarizeEarnings)
			armand.POST("/sector-summary", handleSectorSummary)
		}

		// Market discovery (public)
		api.GET("/market/sectors", handleGetSectors)
		api.GET("/market/macro", handleGetMacro)
		api.GET("/market/movers", handleGetMovers)
		api.GET("/market/sectors-preview", handleGetSectorsPreview)
		api.GET("/market/sector-details/:sector", handleGetSectorDetails)
		api.GET("/market/ipos", handleGetIPOs)
		api.GET("/market/earnings", handleGetEarnings)

		// Protected routes (require JWT)
		protected := api.Group("/")
		protected.Use(AuthMiddleware())
		{
			protected.GET("/watchlist", handleGetWatchlist)
			protected.POST("/watchlist", handleAddWatchlist)
			protected.PUT("/watchlist/:id/portfolio", handleUpdatePortfolio)
			protected.DELETE("/watchlist/:id", handleDeleteWatchlist)

			// Saved analyses
			protected.GET("/analyses", handleGetAnalyses)
			protected.POST("/analyses", handleSaveAnalysis)
		}
	}
}

// validateTicker returns middleware that validates the :ticker URL param
func validateTicker() gin.HandlerFunc {
	return func(c *gin.Context) {
		ticker := c.Param("ticker")
		if !validTicker.MatchString(ticker) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticker format"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func handleHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.Client.Ping(ctx, nil); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": "database unreachable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
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

func handleCompareStocks(c *gin.Context) {
	var req struct {
		Ticker1 string `json:"ticker1" binding:"required"`
		Ticker2 string `json:"ticker2" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Two tickers are required"})
		return
	}

	result, err := armandService.CompareStocks(req.Ticker1, req.Ticker2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleSummarizeEarnings(c *gin.Context) {
	var req struct {
		Transcript string `json:"transcript" binding:"required"`
		Ticker     string `json:"ticker"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transcript text is required"})
		return
	}

	result, err := armandService.SummarizeEarnings(req.Transcript, req.Ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func handleSectorSummary(c *gin.Context) {
	var req services.SectorSummaryRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Sector == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sector is required"})
		return
	}

	result, err := armandService.GenerateSectorSummary(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
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

// --- Saved Analyses ---

func handleSaveAnalysis(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Ticker         string   `json:"ticker" binding:"required"`
		Recommendation string   `json:"recommendation" binding:"required"`
		Reasoning      []string `json:"reasoning" binding:"required"`
		Persona        string   `json:"persona"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	analysis := models.SavedAnalysis{
		UserID:         userID,
		Ticker:         req.Ticker,
		Recommendation: req.Recommendation,
		Reasoning:      req.Reasoning,
		Persona:        req.Persona,
		PriceAtSave:    func() float64 { p, _ := getTickerPrice(req.Ticker); return p }(),
		SavedAt:        time.Now(),
	}

	if _, err := db.Analyses().InsertOne(ctx, analysis); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save analysis"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Analysis saved"})
}

func handleGetAnalyses(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "saved_at", Value: -1}}).SetLimit(50)
	cursor, err := db.Analyses().Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analyses"})
		return
	}
	defer cursor.Close(ctx)

	var analyses []models.SavedAnalysis
	if err := cursor.All(ctx, &analyses); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read analyses"})
		return
	}

	if analyses == nil {
		analyses = []models.SavedAnalysis{}
	}

	c.JSON(http.StatusOK, analyses)
}
