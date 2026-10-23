package events

import (
	"log/slog"
	"sync"
)

// Bus implements Publisher in-memory with thread-safe subscription support.
type Bus struct {
	mu             sync.RWMutex
	handlers       map[string][]func(Event)
	allSubscribers []func(Event)
	logger         *slog.Logger
}

// NewBus creates a new in-memory event bus.
func NewBus(logger *slog.Logger) *Bus {
	return &Bus{
		handlers: make(map[string][]func(Event)),
		logger:   logger,
	}
}

// Subscribe registers a listener for a specific event type.
func (b *Bus) Subscribe(eventType string, fn func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], fn)
}

// SubscribeAll registers a listener for all published domain events.
func (b *Bus) SubscribeAll(fn func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.allSubscribers = append(b.allSubscribers, fn)
}

// Publish distributes the event to all registered subscribers.
func (b *Bus) Publish(event Event) error {
	b.mu.RLock()
	subscribers := append([]func(Event){}, b.handlers[event.Type]...)
	allSubs := append([]func(Event){}, b.allSubscribers...)
	b.mu.RUnlock()

	if b.logger != nil {
		b.logger.Info("event_published",
			"type", event.Type,
			"id", event.ID,
			"entity_type", event.EntityType,
			"entity_id", event.EntityID,
		)
	}

	for _, sub := range subscribers {
		go sub(event)
	}
	for _, sub := range allSubs {
		go sub(event)
	}
	return nil
}
