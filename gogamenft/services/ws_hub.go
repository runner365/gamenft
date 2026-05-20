package services

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"

	"github.com/gamenft/gogamenft/logger"
)

const redisWsChannel = "ws:notifications"

// redisWsPayload is the message published to Redis for cross-instance delivery.
type redisWsPayload struct {
	Addr string    `json:"addr"`
	Msg  WsMessage `json:"msg"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WsMessage is a notification pushed to a client.
type WsMessage struct {
	Type     string `json:"type"`
	TaskID   string `json:"task_id,omitempty"`
	ItemType string `json:"item_type,omitempty"`
	Quantity int    `json:"quantity,omitempty"`
	TxHash   string `json:"tx_hash,omitempty"`
	Error    string `json:"error,omitempty"`
}

// WsClient represents a single WebSocket connection.
type WsClient struct {
	addr string
	conn *websocket.Conn
	send chan []byte
	hub  *WsHub
	done chan struct{}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func (c *WsClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WsClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				logger.LogWarnf("WsClient: send channel closed addr=%s", c.addr)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			logger.LogInfof("WsClient: sent type=%s to addr=%s", string(msg), c.addr)
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// WsHub manages all WebSocket connections, keyed by user address.
type WsHub struct {
	clients     map[string]*WsClient
	register    chan *WsClient
	unregister  chan *WsClient
	mu          sync.RWMutex
	redisClient *redis.Client
	pubsub      *redis.PubSub
}

// NewWsHub creates a WebSocket hub and starts its event loop.
// If redisClient is non-nil, it subscribes to Redis Pub/Sub for cross-instance message delivery.
func NewWsHub(redisClient *redis.Client) *WsHub {
	h := &WsHub{
		clients:     make(map[string]*WsClient),
		register:    make(chan *WsClient),
		unregister:  make(chan *WsClient),
		redisClient: redisClient,
	}
	go h.run()
	if redisClient != nil {
		go h.subscribeRedis()
	}
	return h
}

func (h *WsHub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			// Close existing connection for this address if any
			if old, ok := h.clients[c.addr]; ok {
				close(old.send)
				delete(h.clients, c.addr)
			}
			h.clients[c.addr] = c
			h.mu.Unlock()
			logger.LogInfof("WsHub: client connected addr=%s", c.addr)

		case c := <-h.unregister:
			h.mu.Lock()
			if existing, ok := h.clients[c.addr]; ok && existing == c {
				delete(h.clients, c.addr)
			}
			h.mu.Unlock()
			close(c.send)
			logger.LogInfof("WsHub: client disconnected addr=%s", c.addr)
		}
	}
}

// Push sends a message to a specific user's WebSocket connection.
func (h *WsHub) Push(addr string, msg WsMessage) {
	h.mu.RLock()
	c, ok := h.clients[addr]
	h.mu.RUnlock()
	if !ok {
		logger.LogWarnf("WsHub: no client for addr=%s type=%s", addr, msg.Type)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		logger.LogErrorf("WsHub: marshal failed addr=%s err=%v", addr, err)
		return
	}

	select {
	case c.send <- data:
		logger.LogInfof("WsHub: pushed type=%s to addr=%s", msg.Type, addr)
	default:
		logger.LogWarnf("WsHub: send buffer full addr=%s type=%s", addr, msg.Type)
	}
}

// Publish broadcasts a message to the target user across all instances via Redis Pub/Sub.
// It publishes to Redis for other instances AND delivers locally for clients connected to this instance.
// If Redis is not configured, it falls back to local-only Push.
func (h *WsHub) Publish(addr string, msg WsMessage) {
	if h.redisClient != nil {
		payload := redisWsPayload{Addr: addr, Msg: msg}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.LogErrorf("WsHub: marshal publish failed addr=%s err=%v", addr, err)
			return
		}
		if err := h.redisClient.Publish(context.Background(), redisWsChannel, data).Err(); err != nil {
			logger.LogErrorf("WsHub: redis publish failed addr=%s err=%v", addr, err)
		}
	}
	// Also deliver locally
	h.Push(addr, msg)
}

// subscribeRedis listens on the Redis Pub/Sub channel and forwards messages to local clients.
func (h *WsHub) subscribeRedis() {
	if h.redisClient == nil {
		return
	}
	ctx := context.Background()
	pubsub := h.redisClient.Subscribe(ctx, redisWsChannel)
	h.pubsub = pubsub
	defer pubsub.Close()

	ch := pubsub.Channel()
	logger.LogInfof("WsHub: subscribed to Redis channel %s", redisWsChannel)
	for msg := range ch {
		var payload redisWsPayload
		if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
			logger.LogErrorf("WsHub: redis unmarshal failed err=%v", err)
			continue
		}
		h.Push(payload.Addr, payload.Msg)
	}
	logger.LogInfof("WsHub: Redis subscription ended")
}

// HandleUpgrade is a Gin handler that upgrades HTTP to WebSocket.
// The caller must provide a JWT validator function to extract the user address.
func (h *WsHub) HandleUpgrade(validateToken func(string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		addr, err := validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.LogErrorf("WsHub: upgrade failed addr=%s err=%v", addr, err)
			return
		}

		client := &WsClient{
			addr: addr,
			conn: conn,
			send: make(chan []byte, 16),
			hub:  h,
			done: make(chan struct{}),
		}
		h.register <- client

		go client.writePump()
		go client.readPump()
	}
}
