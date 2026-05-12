package api

import (
	"net/http"
	"time"

	"armtrade-backend/db"
	"armtrade-backend/models"

	"github.com/gin-gonic/gin"
)

func handleGetWatchlist(c *gin.Context) {
	userID := c.GetUint("userID")

	var items []models.WatchlistItem
	db.DB.Where("user_id = ?", userID).Order("added_at DESC").Find(&items)

	// Enrich with live prices
	type WatchlistResponse struct {
		Ticker string  `json:"ticker"`
		Price  float64 `json:"price"`
		Change float64 `json:"change"`
	}

	var response []WatchlistResponse
	for _, item := range items {
		price, change := getTickerPrice(item.Ticker)
		response = append(response, WatchlistResponse{
			Ticker: item.Ticker,
			Price:  price,
			Change: change,
		})
	}

	if response == nil {
		response = []WatchlistResponse{}
	}

	c.JSON(http.StatusOK, response)
}

func handleAddWatchlist(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Ticker string `json:"ticker" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required"})
		return
	}

	// Check if already in watchlist
	var existing models.WatchlistItem
	if result := db.DB.Where("user_id = ? AND ticker = ?", userID, req.Ticker).First(&existing); result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Already in watchlist"})
		return
	}

	item := models.WatchlistItem{
		UserID:  userID,
		Ticker:  req.Ticker,
		AddedAt: time.Now(),
	}

	if result := db.DB.Create(&item); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watchlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Added to watchlist"})
}

func handleDeleteWatchlist(c *gin.Context) {
	userID := c.GetUint("userID")
	ticker := c.Param("ticker")

	result := db.DB.Where("user_id = ? AND ticker = ?", userID, ticker).Delete(&models.WatchlistItem{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found in watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from watchlist"})
}

// getTickerPrice fetches live price from Yahoo chart endpoint
func getTickerPrice(ticker string) (price float64, changePercent float64) {
	chartData, err := yahooService.GetChart(ticker, "1d", "5d")
	if err != nil {
		return 0, 0
	}

	chart, ok := chartData["chart"].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	result, ok := chart["result"].([]interface{})
	if !ok || len(result) == 0 {
		return 0, 0
	}
	meta, ok := result[0].(map[string]interface{})
	if !ok {
		return 0, 0
	}
	metaData, ok := meta["meta"].(map[string]interface{})
	if !ok {
		return 0, 0
	}

	regularPrice, _ := metaData["regularMarketPrice"].(float64)
	previousClose, _ := metaData["chartPreviousClose"].(float64)

	if previousClose > 0 {
		changePercent = ((regularPrice - previousClose) / previousClose) * 100
	}

	return regularPrice, changePercent
}
