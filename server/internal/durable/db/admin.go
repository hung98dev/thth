package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CreateDatabase creates an empty database on the cluster reachable via
// adminDSN (which must connect to a maintenance database such as postgres).
// The name is carried as a pgx identifier, never interpolated raw.
func CreateDatabase(ctx context.Context, adminDSN, name string) error {
	return adminExec(ctx, adminDSN,
		`CREATE DATABASE `+pgx.Identifier{name}.Sanitize(),
		"db: create "+name)
}

// DropDatabase drops the named database when it exists, terminating any
// sessions still attached to it (DROP DATABASE ... WITH (FORCE)).
func DropDatabase(ctx context.Context, adminDSN, name string) error {
	return adminExec(ctx, adminDSN,
		`DROP DATABASE IF EXISTS `+pgx.Identifier{name}.Sanitize()+` WITH (FORCE)`,
		"db: drop "+name)
}

func adminExec(ctx context.Context, adminDSN, stmt, what string) error {
	pool, err := Open(ctx, adminDSN, 1)
	if err != nil {
		return fmt.Errorf("db: admin pool: %w", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}
