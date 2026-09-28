// Package db owns the pinned pgx/v5 pool construction for the durable layer
// (database.md § Technology, § Connection Pool): bounded connections, UTC
// session, context/deadline-acquired connections. It is the single place the
// pgx driver and pool tuning live so feature code never opens connections
// directly.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open creates a bounded pgxpool for the single world process
// (database.md § Connection Pool). The pool maximum keeps operational
// headroom for migrations/ops tools (ADR-0052); the exact size is
// environment capacity configuration, so callers size it via config and
// this constructor only applies structural defaults.
func Open(ctx context.Context, dsn string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	// Bounded connection lifetime keeps pool churn predictable; acquisition
	// itself is deadline-bound by the caller's context (database.md
	// § Connection Pool).
	cfg.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}
