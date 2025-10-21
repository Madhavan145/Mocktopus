package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

// Event types for inter-service communication
const (
	SimulationStarted   = "simulation.started"
	SimulationProgress  = "simulation.progress"
	SimulationCompleted = "simulation.completed"
	SimulationFailed    = "simulation.failed"

	MetricsDerived      = "metrics.derived"
	CostEstimated       = "cost.estimated"
	TopologyUpdated     = "topology.updated"
	ProjectCreated      = "project.created"
	ProjectDeleted      = "project.deleted"
)

// Event is the standard event envelope
type Event struct {
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	RunID     string          `json:"runId,omitempty"`
	ProjectID string          `json:"projectId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// Handler is a function that processes an event
type Handler func(ctx context.Context, event Event) error

// Bus provides pub/sub event communication over Redis
type Bus struct {
	client   *redis.Client
	handlers map[string][]Handler
	mu       sync.RWMutex
}

// NewBus creates an event bus backed by Redis pub/sub
func NewBus(client *redis.Client) *Bus {
	return &Bus{
		client:   client,
		handlers: make(map[string][]Handler),
	}
}

// Publish sends an event to all subscribers
func (b *Bus) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return b.client.Publish(ctx, "moktopus:events:"+event.Type, data).Err()
}

// Subscribe registers a handler for an event type
func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Listen starts listening for events on all subscribed channels
func (b *Bus) Listen(ctx context.Context) error {
	b.mu.RLock()
	channels := make([]string, 0, len(b.handlers))
	for eventType := range b.handlers {
		channels = append(channels, "moktopus:events:"+eventType)
	}
	b.mu.RUnlock()

	if len(channels) == 0 {
		return nil
	}

	pubsub := b.client.Subscribe(ctx, channels...)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var event Event
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("[event-bus] unmarshal error: %v", err)
				continue
			}
			b.dispatch(ctx, event)
		}
	}
}

func (b *Bus) dispatch(ctx context.Context, event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			log.Printf("[event-bus] handler error for %s: %v", event.Type, err)
		}
	}
}
