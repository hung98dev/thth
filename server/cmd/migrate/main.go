// Command migrate is the single schema-migration owner tool (migrations.md
// § Ownership). It applies/rolls back the immutable numbered pairs in
// server/migrations via golang-migrate 4.20.1 over pgx/v5.
//
// Usage:
//
//	migrate -dsn postgres://... [-dir server/migrations] <command>
//
// Commands: up | down <n> | version. down is data-destructive — local/staging
// rehearsal only (migrations.md § Down / Rollback); never run it against a
// data-bearing environment.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"thinhthan/internal/conformance/gates"
	"thinhthan/internal/durable/schema"
)

func main() {
	os.Exit(run())
}

func run() int {
	dsn := flag.String("dsn", os.Getenv("THINHTHAN_DATABASE_URL"), "PostgreSQL DSN (or THINHTHAN_DATABASE_URL)")
	dir := flag.String("dir", "", "migrations directory (default <repo>/server/migrations)")
	flag.Parse()
	args := flag.Args()
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "migrate: -dsn or THINHTHAN_DATABASE_URL is required")
		return 2
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "migrate: command required: up | down <n> | version")
		return 2
	}

	migDir := *dir
	if migDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			return 2
		}
		root, err := gates.RepoRoot(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			return 2
		}
		migDir = filepath.Join(root, "server", "migrations")
	}

	m, err := schema.NewMigrator(*dsn, migDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		return 1
	}
	defer func() { _ = m.Close() }()

	switch args[0] {
	case "up":
		if err := m.Up(); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			return 1
		}
	case "down":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "migrate: down requires explicit step count")
			return 2
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			fmt.Fprintln(os.Stderr, "migrate: down step count must be a positive integer")
			return 2
		}
		if err := m.Steps(-n); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			return 1
		}
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
			return 1
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
	default:
		fmt.Fprintf(os.Stderr, "migrate: unknown command %q\n", args[0])
		return 2
	}
	return 0
}
