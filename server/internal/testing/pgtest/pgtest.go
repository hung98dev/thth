// Package pgtest is the PostgreSQL 18.6 test harness (ADR-0058, packet
// IMP-005): tests connect through THINHTHAN_TEST_PG_DSN when it is set (Linux
// CI postgres:18.6 service container, local docker start by verify.ps1) and
// otherwise start the pinned EDB binaries (Windows job / local Windows) and
// export the DSN.
package pgtest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DSNEnv is the environment variable carrying the test DSN.
const DSNEnv = "THINHTHAN_TEST_PG_DSN"

// EDBBinEnv optionally points at the bin/ directory of the pinned EDB
// PostgreSQL 18.6 binaries (initdb/pg_ctl). When unset, the standard cache
// locations used by .github/workflows/verify.yml are searched.
const EDBBinEnv = "THINHTHAN_TEST_EDB_BIN"

// Backend identifies how the harness provides the test server.
type Backend int

const (
	// BackendNone means no backing is available (no DSN, no binaries).
	BackendNone Backend = iota
	// BackendPresetDSN means THINHTHAN_TEST_PG_DSN is set.
	BackendPresetDSN
	// BackendEDB means the pinned EDB binaries will be started locally.
	BackendEDB
)

// Select reports the harness backing: a preset DSN always wins over
// locally-started binaries (ADR-0058). The returned value is the DSN for
// BackendPresetDSN or the binaries directory for BackendEDB.
func Select() (Backend, string) {
	if dsn := os.Getenv(DSNEnv); dsn != "" {
		return BackendPresetDSN, dsn
	}
	if dir := edbBinDir(); dir != "" {
		return BackendEDB, dir
	}
	return BackendNone, ""
}

// DSN returns a DSN usable by the calling test, starting the EDB binaries
// when needed, and skips the test when no backing exists.
func DSN(t testing.TB) string {
	t.Helper()
	switch backend, value := Select(); backend {
	case BackendPresetDSN:
		return value
	case BackendEDB:
		return StartEDB(t, value)
	default:
		t.Skipf("pgtest: neither %s nor pinned EDB binaries (%s) are available",
			DSNEnv, EDBBinEnv)
		return ""
	}
}

// edbBinDir locates the pinned EDB bin directory: EDBBinEnv wins, then the
// verify.yml cache layouts ($RUNNER_TEMP\edb-cache\pgsql\bin and its
// \bin fallback, plus $TEMP equivalents for local Windows runs).
func edbBinDir() string {
	if dir := os.Getenv(EDBBinEnv); dir != "" {
		if _, err := os.Stat(exePath(dir, "initdb")); err == nil {
			return dir
		}
		return ""
	}
	var roots []string
	if v := os.Getenv("RUNNER_TEMP"); v != "" {
		roots = append(roots, v)
	}
	roots = append(roots, os.TempDir())
	for _, root := range roots {
		for _, sub := range [][]string{
			{"edb-cache", "pgsql", "bin"},
			{"edb-cache", "bin"},
		} {
			dir := filepath.Join(append([]string{root}, sub...)...)
			if _, err := os.Stat(exePath(dir, "initdb")); err == nil {
				return dir
			}
		}
	}
	return ""
}

func exePath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".exe")
	}
	return filepath.Join(dir, name)
}

// StartEDB provisions a fresh datadir under the test's temp dir and starts
// postgres on a free loopback port — mirroring the pinned verify.yml step
// (initdb -U postgres -A trust -E UTF8 --locale=C; pg_ctl -w -t 120).
func StartEDB(t testing.TB, binDir string) string {
	t.Helper()
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "pgdata")
	logFile := filepath.Join(dir, "postgres.log")

	out, err := exec.Command(exePath(binDir, "initdb"),
		"-D", dataDir, "-U", "postgres", "-A", "trust", "-E", "UTF8",
		"--locale=C").CombinedOutput()
	if err != nil {
		t.Fatalf("pgtest: initdb: %v\n%s", err, out)
	}

	port := freePort(t)
	out, err = exec.Command(exePath(binDir, "pg_ctl"),
		"-D", dataDir, "-o", "-p "+strconv.Itoa(port),
		"-l", logFile, "-w", "-t", "120", "start").CombinedOutput()
	if err != nil {
		t.Fatalf("pgtest: pg_ctl start: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		_ = exec.CommandContext(ctx, exePath(binDir, "pg_ctl"),
			"-D", dataDir, "-w", "-t", "60", "-m", "fast", "stop").Run()
	})
	return fmt.Sprintf("postgres://postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
}

func freePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pgtest: reserve port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

var scratchSeq atomic.Uint64

// ScratchDSN creates an isolated empty database <prefix>_<pid>_<seq> over the
// admin connection of baseDSN and returns its DSN plus a drop function.
// Callers apply migrations via schema.NewMigrator.
func ScratchDSN(ctx context.Context, baseDSN, prefix string) (string, func(), error) {
	u, err := url.Parse(baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("pgtest: parse dsn: %w", err)
	}
	name := fmt.Sprintf("%s_%d_%d", prefix, os.Getpid(), scratchSeq.Add(1))
	adminURL := *u
	adminURL.Path = "/postgres"

	admin, err := pgxpool.New(ctx, adminURL.String())
	if err != nil {
		return "", nil, fmt.Errorf("pgtest: admin pool: %w", err)
	}
	defer admin.Close()
	qname := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+qname); err != nil {
		return "", nil, fmt.Errorf("pgtest: drop stale %s: %w", name, err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+qname); err != nil {
		return "", nil, fmt.Errorf("pgtest: create %s: %w", name, err)
	}

	u.Path = "/" + name
	dsn := u.String()
	drop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a, err := pgxpool.New(ctx, adminURL.String())
		if err != nil {
			return
		}
		defer a.Close()
		_, _ = a.Exec(ctx, `DROP DATABASE IF EXISTS `+qname+` WITH (FORCE)`)
	}
	return dsn, drop, nil
}

// Scratch is DSN + ScratchDSN in one call; the scratch database is dropped at
// test cleanup.
func Scratch(t testing.TB, prefix string) string {
	t.Helper()
	base := DSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dsn, drop, err := ScratchDSN(ctx, base, prefix)
	if err != nil {
		t.Fatalf("pgtest: scratch: %v", err)
	}
	t.Cleanup(drop)
	return dsn
}
