package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool with convenience methods
type Pool struct {
	*pgxpool.Pool
}

// Connect creates a new database connection pool from DATABASE_URL env var
func Connect(ctx context.Context) (*Pool, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}
	return ConnectWithURL(ctx, url)
}

// ConnectWithURL creates a pool with a specific connection string
func ConnectWithURL(ctx context.Context, url string) (*Pool, error) {
	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// Close cleanly shuts down the pool
func (p *Pool) Close() {
	p.Pool.Close()
}
