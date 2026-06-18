package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleGetNetwork(c *gin.Context) {
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

	result, err := armandService.DiscoverNetwork(req.Ticker)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
