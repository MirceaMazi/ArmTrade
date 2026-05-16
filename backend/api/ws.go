package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ─── WebSocket Price Hub ───

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// PriceUpdate is sent to clients when a ticker price changes.
type PriceUpdate struct {
	Ticker       string  `json:"ticker"`
	Price        float64 `json:"price"`
	Change       float64 `json:"change"`
	ChangePct    float64 `json:"changePct"`
	PrevClose    float64 `json:"prevClose"`
}

type wsClient struct {
	conn    *websocket.Conn
	tickers map[string]bool
	send    chan []byte
	mu      sync.Mutex
}

type priceHub struct {
	clients    map[*wsClient]bool
	register   chan *wsClient
	unregister chan *wsClient
	mu         sync.RWMutex
}

var hub = &priceHub{
	clients:    make(map[*wsClient]bool),
	register:   make(chan *wsClient),
	unregister: make(chan *wsClient),
}

func init() {
	go hub.run()
}

func (h *priceHub) run() {
	// Manage client registrations
	go func() {
		for {
			select {
			case client := <-h.register:
				h.mu.Lock()
				h.clients[client] = true
				h.mu.Unlock()
			case client := <-h.unregister:
				h.mu.Lock()
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
				}
				h.mu.Unlock()
			}
		}
	}()

	// Price polling loop: every 15 seconds, fetch prices for all subscribed tickers
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.broadcastPrices()
	}
}

func (h *priceHub) subscribedTickers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	tickerSet := make(map[string]bool)
	for client := range h.clients {
		client.mu.Lock()
		for t := range client.tickers {
			tickerSet[t] = true
		}
		client.mu.Unlock()
	}

	tickers := make([]string, 0, len(tickerSet))
	for t := range tickerSet {
		tickers = append(tickers, t)
	}
	return tickers
}

func (h *priceHub) broadcastPrices() {
	tickers := h.subscribedTickers()
	if len(tickers) == 0 {
		return
	}

	// Fetch prices (reuses Yahoo service cache)
	updates := make([]PriceUpdate, 0, len(tickers))
	for _, t := range tickers {
		price, changePct := getTickerPrice(t)
		if price == 0 {
			continue
		}
		// Calculate previous close and absolute change from changePct
		prevClose := 0.0
		change := 0.0
		if changePct != 0 {
			prevClose = price / (1 + changePct/100)
			change = price - prevClose
		}
		updates = append(updates, PriceUpdate{
			Ticker:    t,
			Price:     price,
			Change:    change,
			ChangePct: changePct,
			PrevClose: prevClose,
		})
	}

	if len(updates) == 0 {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Filter updates to only tickers this client cares about
		client.mu.Lock()
		var clientUpdates []PriceUpdate
		for _, u := range updates {
			if client.tickers[u.Ticker] {
				clientUpdates = append(clientUpdates, u)
			}
		}
		client.mu.Unlock()

		if len(clientUpdates) == 0 {
			continue
		}

		msg, _ := json.Marshal(clientUpdates)
		select {
		case client.send <- msg:
		default:
			// Client buffer full, skip
		}
	}
}

// handleWebSocket upgrades the connection and manages per-client read/write.
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &wsClient{
		conn:    conn,
		tickers: make(map[string]bool),
		send:    make(chan []byte, 16),
	}

	hub.register <- client

	// Writer goroutine
	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Reader goroutine (handles subscribe/unsubscribe)
	defer func() {
		hub.unregister <- client
		conn.Close()
	}()

	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	// Start ping ticker
	go func() {
		pingTicker := time.NewTicker(30 * time.Second)
		defer pingTicker.Stop()
		for range pingTicker.C {
			client.mu.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			client.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Action  string   `json:"action"`
			Tickers []string `json:"tickers"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		client.mu.Lock()
		switch msg.Action {
		case "subscribe":
			for _, t := range msg.Tickers {
				t = strings.ToUpper(strings.TrimSpace(t))
				if validTicker.MatchString(t) {
					client.tickers[t] = true
				}
			}
		case "unsubscribe":
			for _, t := range msg.Tickers {
				delete(client.tickers, strings.ToUpper(strings.TrimSpace(t)))
			}
		}
		client.mu.Unlock()

		// Send immediate price snapshot for newly subscribed tickers
		if msg.Action == "subscribe" {
			go func(tickers []string) {
				var snapshot []PriceUpdate
				for _, t := range tickers {
					t = strings.ToUpper(strings.TrimSpace(t))
					price, changePct := getTickerPrice(t)
					if price == 0 {
						continue
					}
					prevClose := 0.0
					change := 0.0
					if changePct != 0 {
						prevClose = price / (1 + changePct/100)
						change = price - prevClose
					}
					snapshot = append(snapshot, PriceUpdate{
						Ticker:    t,
						Price:     price,
						Change:    change,
						ChangePct: changePct,
						PrevClose: prevClose,
					})
				}
				if len(snapshot) > 0 {
					data, _ := json.Marshal(snapshot)
					select {
					case client.send <- data:
					default:
					}
				}
			}(msg.Tickers)
		}
	}
}
