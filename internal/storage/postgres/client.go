package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/aegis-decision-engine/ade/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps a PostgreSQL connection pool
type Client struct {
	pool *pgxpool.Pool
}

// NewClient creates a new PostgreSQL client
func NewClient(cfg *config.Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{pool: pool}, nil
}

// Close closes the database connection pool
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Health returns true if the database is healthy
func (c *Client) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.pool.Ping(ctx)
}
