package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nenav/nenav/backend/internal/events"
)

type client struct {
	id       string
	channels map[string]bool
	send     chan []byte
}

// RealtimeHandler manages server-sent event (SSE) connections across channels.
type RealtimeHandler struct {
	mu         sync.RWMutex
	clients    map[*client]bool
	logger     *slog.Logger
	register   chan *client
	unregister chan *client
	broadcast  chan channelMessage
}

type channelMessage struct {
	channel string
	data    []byte
}

func NewRealtimeHandler(eventBus *events.Bus, logger *slog.Logger) *RealtimeHandler {
	h := &RealtimeHandler{
		clients:    make(map[*client]bool),
		logger:     logger,
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan channelMessage, 256),
	}

	go h.run()

	// Connect to domain event bus
	if eventBus != nil {
		eventBus.SubscribeAll(func(e events.Event) {
			h.HandleDomainEvent(e)
		})
	}

	return h
}

func (h *RealtimeHandler) run() {
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("sse_client_connected", "id", c.id)
			}

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			if h.logger != nil {
				h.logger.Debug("sse_client_disconnected", "id", c.id)
			}

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				// Deliver if client subscribed to this channel or to wildcard
				if c.channels[msg.channel] || c.channels["*"] || strings.HasPrefix(msg.channel, "admin:") && c.channels["admin:marketplace"] {
					select {
					case c.send <- msg.data:
					default:
						// Buffer full; skip or drop
					}
				}
			}
			h.mu.RUnlock()

		case <-heartbeat.C:
			h.mu.RLock()
			ping := []byte(": keepalive\n\n")
			for c := range h.clients {
				select {
				case c.send <- ping:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleDomainEvent routes domain events into SSE channels.
func (h *RealtimeHandler) HandleDomainEvent(e events.Event) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return
	}

	formatted := fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", e.Type, e.ID, string(bytes))
	msg := []byte(formatted)

	// Broadcast to admin stream
	h.broadcast <- channelMessage{channel: "admin:marketplace", data: msg}

	// Route to specific entity channels
	if e.EntityType == "booking" && e.EntityID != "" {
		h.broadcast <- channelMessage{channel: "booking:" + e.EntityID, data: msg}
	}
	if e.ActorID != "" {
		h.broadcast <- channelMessage{channel: "customer:" + e.ActorID, data: msg}
		h.broadcast <- channelMessage{channel: "prof:" + e.ActorID, data: msg}
	}
}

// BroadcastManual allows direct notification dispatch to a specific channel.
func (h *RealtimeHandler) BroadcastManual(channel, eventType string, payload any) {
	evt := events.New(eventType, "system", "realtime", channel, payload)
	bytes, err := json.Marshal(evt)
	if err != nil {
		return
	}
	formatted := fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", eventType, evt.ID, string(bytes))
	h.broadcast <- channelMessage{channel: channel, data: []byte(formatted)}
}

// HandleStream handles SSE streaming requests: GET /api/v1/realtime/stream?channel=...
func (h *RealtimeHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	channelsParam := r.URL.Query().Get("channel")
	if channelsParam == "" {
		channelsParam = "admin:marketplace"
	}

	channelsList := strings.Split(channelsParam, ",")
	channelMap := make(map[string]bool)
	for _, ch := range channelsList {
		ch = strings.TrimSpace(ch)
		if ch != "" {
			channelMap[ch] = true
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	// Flush initial connection confirmation
	fmt.Fprintf(w, ": connected to channels %s\n\n", channelsParam)
	flusher.Flush()

	client := &client{
		id:       fmt.Sprintf("%d", time.Now().UnixNano()),
		channels: channelMap,
		send:     make(chan []byte, 64),
	}

	h.register <- client
	defer func() {
		h.unregister <- client
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case data, ok := <-client.send:
			if !ok {
				return
			}
			_, err := w.Write(data)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
