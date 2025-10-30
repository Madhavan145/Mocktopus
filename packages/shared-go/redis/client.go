package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client
type Client struct {
	*redis.Client
}

// Connect creates a new Redis client from REDIS_URL env var
func Connect(ctx context.Context) (*Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{Client: client}, nil
}

// Close shuts down the Redis connection
func (c *Client) Close() error {
	return c.Client.Close()
}
