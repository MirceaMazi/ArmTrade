package api

import (
	"context"
	"net/http"
	"time"

	"armtrade-backend/db"
	"armtrade-backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func handleGetWatchlist(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "added_at", Value: -1}})
	cursor, err := db.Watchlist().Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch watchlist"})
		return
	}
	defer cursor.Close(ctx)

	var items []models.WatchlistItem
	if err := cursor.All(ctx, &items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read watchlist"})
		return
	}

	// Enrich with live prices
	type WatchlistResponse struct {
		ID       string   `json:"id"`
		Ticker   string   `json:"ticker"`
		Price    float64  `json:"price"`
		Change   float64  `json:"change"`
		BuyPrice *float64 `json:"buyPrice,omitempty"`
		Quantity *float64 `json:"quantity,omitempty"`
		BuyDate  *string  `json:"buyDate,omitempty"`
	}

	var response []WatchlistResponse
	for _, item := range items {
		price, change := getTickerPrice(item.Ticker)
		wr := WatchlistResponse{
			ID:       item.ID.Hex(),
			Ticker:   item.Ticker,
			Price:    price,
			Change:   change,
			BuyPrice: item.BuyPrice,
			Quantity: item.Quantity,
		}
		if item.BuyDate != nil {
			d := item.BuyDate.Format("2006-01-02")
			wr.BuyDate = &d
		}
		response = append(response, wr)
	}

	if response == nil {
		response = []WatchlistResponse{}
	}

	c.JSON(http.StatusOK, response)
}

func handleAddWatchlist(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Ticker string `json:"ticker" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticker is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	item := models.WatchlistItem{
		UserID:  userID,
		Ticker:  req.Ticker,
		AddedAt: time.Now(),
	}

	if _, err := db.Watchlist().InsertOne(ctx, item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watchlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Added to watchlist"})
}

func handleDeleteWatchlist(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	idParam := c.Param("id")
	oid, err := bson.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := db.Watchlist().DeleteOne(ctx, bson.M{"_id": oid, "user_id": userID})
	if err != nil || result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found in watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from watchlist"})
}

// getUserObjectID extracts the user's ObjectID from the gin context (set by AuthMiddleware).
func getUserObjectID(c *gin.Context) (bson.ObjectID, error) {
	userIDHex := c.GetString("userID")
	return bson.ObjectIDFromHex(userIDHex)
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

func handleUpdatePortfolio(c *gin.Context) {
	userID, err := getUserObjectID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	idParam := c.Param("id")
	oid, err := bson.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req struct {
		BuyPrice *float64 `json:"buyPrice"`
		Quantity *float64 `json:"quantity"`
		BuyDate  *string  `json:"buyDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{}
	if req.BuyPrice != nil {
		update["buy_price"] = *req.BuyPrice
	}
	if req.Quantity != nil {
		update["quantity"] = *req.Quantity
	}
	if req.BuyDate != nil {
		t, err := time.Parse("2006-01-02", *req.BuyDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format, use YYYY-MM-DD"})
			return
		}
		update["buy_date"] = t
	}

	if len(update) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	result, err := db.Watchlist().UpdateOne(
		ctx,
		bson.M{"_id": oid, "user_id": userID},
		bson.M{"$set": update},
	)
	if err != nil || result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Watchlist item not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Portfolio data updated"})
}
