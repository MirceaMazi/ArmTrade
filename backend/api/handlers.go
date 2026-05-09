package api

import (
	"net/http"

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
