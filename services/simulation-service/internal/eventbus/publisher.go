package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Event types for inter-service communication
const (
	SimulationStarted   = "simulation.started"
	SimulationCompleted = "simulation.completed"
	SimulationFailed    = "simulation.failed"
	TopologyUpdated     = "topology.updated"
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

// Publisher sends events to Redis pub/sub
type Publisher struct {
	client *redis.Client
}

// NewPublisher creates a Redis event publisher; returns nil if REDIS_URL is not set
func NewPublisher() *Publisher {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		log.Println("No REDIS_URL set, event publishing disabled")
		return nil
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Printf("WARN: invalid REDIS_URL: %v", err)
		return nil
	}

	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("WARN: Redis ping failed: %v, event publishing disabled", err)
		return nil
	}

	log.Println("Connected to Redis for event publishing")
	return &Publisher{client: client}
}

// Publish sends an event to the appropriate channel
func (p *Publisher) Publish(ctx context.Context, event Event) error {
	if p == nil {
		return nil
	}
	event.Timestamp = time.Now().UnixMilli()
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	channel := "moktopus:events:" + event.Type
	return p.client.Publish(ctx, channel, data).Err()
}

// Close closes the Redis connection
func (p *Publisher) Close() {
	if p != nil && p.client != nil {
		p.client.Close()
	}
}
