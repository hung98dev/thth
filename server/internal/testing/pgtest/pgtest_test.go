package pgtest_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"thinhthan/internal/testing/pgtest"
)

func TestUsesPresetDsn(t *testing.T) {
	const dsn = "postgres://preset.example.invalid/db"
	t.Setenv(pgtest.DSNEnv, dsn)
	backend, value := pgtest.Select()
	if backend != pgtest.BackendPresetDSN {
		t.Fatalf("Select() backend = %v, want BackendPresetDSN", backend)
	}
	if value != dsn {
		t.Fatalf("Select() value = %q, want %q", value, dsn)
	}
	// DSN must return the preset verbatim without contacting any server.
	if got := pgtest.DSN(t); got != dsn {
		t.Fatalf("DSN() = %q, want %q", got, dsn)
	}
}

func TestStartsEdbBinariesWhenDsnUnset(t *testing.T) {
	// Force the unset path regardless of the caller's environment.
	t.Setenv(pgtest.DSNEnv, "")
	binDir := os.Getenv(pgtest.EDBBinEnv)
	if binDir == "" {
		backend, value := pgtest.Select()
		if backend != pgtest.BackendEDB {
			t.Skipf("no EDB binaries found (set %s); Select() = %v/%q",
				pgtest.EDBBinEnv, backend, value)
		}
		binDir = value
	}
	dsn := pgtest.StartEDB(t, binDir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("started postgres not reachable: %v", err)
	}
	defer conn.Close(ctx)
	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Fatalf("SELECT 1 = %d, %v", one, err)
	}
}
