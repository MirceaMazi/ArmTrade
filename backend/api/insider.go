package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleGetInsider(c *gin.Context) {
	var req struct {
		Ticker string `json:"ticker" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticker is required"})
		return
	}

	if !validTicker.MatchString(req.Ticker) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticker format"})
		return
	}

	// 1. Fetch real insider data from Yahoo Finance (SEC Form 4 filings)
	insiderData, err := yahooService.GetInsiderData(req.Ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch insider data: " + err.Error()})
		return
	}

	// 2. Have Armand analyze the insider activity
	analysis, err := armandService.AnalyzeInsiderActivity(req.Ticker, insiderData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze insider activity: " + err.Error()})
		return
	}

	// 3. Return combined response: AI analysis + raw data for source citations
	c.JSON(http.StatusOK, gin.H{
		"ticker":   req.Ticker,
		"analysis": analysis,
		"rawData":  insiderData,
		"source":   "SEC Form 4 filings via Yahoo Finance",
		"sourceUrl": "https://finance.yahoo.com/quote/" + req.Ticker + "/insider-transactions/",
	})
}
