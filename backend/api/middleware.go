package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"armtrade-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.JWTSecret()), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Extract user ID (stored as hex string of ObjectID)
		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("username", claims["username"])
		c.Next()
	}
}

// RateLimitMiddleware limits requests per IP for expensive AI endpoints.
// Allows 10 requests per minute per IP.
func RateLimitMiddleware() gin.HandlerFunc {
	type client struct {
		count   int
		resetAt time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	const (
		maxRequests = 10
		window      = 1 * time.Minute
	)

	// Background cleanup every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.After(c.resetAt) {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()
		cl, exists := clients[ip]
		if !exists || time.Now().After(cl.resetAt) {
			clients[ip] = &client{count: 1, resetAt: time.Now().Add(window)}
			mu.Unlock()
			c.Next()
			return
		}

		cl.count++
		if cl.count > maxRequests {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded. Please wait before making more AI requests."})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
