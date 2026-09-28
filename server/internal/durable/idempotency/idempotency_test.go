package idempotency_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thinhthan/internal/core/id"
	"thinhthan/internal/durable/db"
	"thinhthan/internal/durable/idempotency"
	"thinhthan/internal/durable/schema"
	"thinhthan/internal/testing/pgtest"
)

// migratedPool provisions a scratch database, applies the baseline schema and
// returns an open pool. Tests needing no DSN skip inside pgtest.
func migratedPool(t *testing.T, prefix string) *pgxpool.Pool {
	t.Helper()
	base := pgtest.DSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	dsn, drop, err := pgtest.ScratchDSN(ctx, base, prefix)
	if err != nil {
		t.Fatalf("scratch: %v", err)
	}
	t.Cleanup(drop)
	m, err := schema.NewMigrator(dsn, schema.MigrationsDir())
	if err != nil {
		t.Fatalf("migrator: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("migrator close: %v", err)
	}
	pool, err := db.Open(context.Background(), dsn, 4)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func fp(seed byte) [32]byte {
	return sha256.Sum256([]byte{seed})
}

// outcomeEqual compares JSONB outcomes semantically — PostgreSQL JSONB does
// not preserve input whitespace.
func outcomeEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("unmarshal outcome a: %v", err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("unmarshal outcome b: %v", err)
	}
	return reflect.DeepEqual(va, vb)
}

func newUUID(t *testing.T) id.UUID {
	t.Helper()
	u, err := id.NewUUIDv4()
	if err != nil {
		t.Fatalf("uuid: %v", err)
	}
	return u
}

// bumpCounter is a representative value mutation: it inserts one
// rate_limit_counters row and reports the inserted count as the outcome —
// at-most-once is observable by counting committed rows.
func bumpCounter(keyHash []byte) func(context.Context, pgx.Tx) (idempotency.Outcome, error) {
	return func(ctx context.Context, tx pgx.Tx) (idempotency.Outcome, error) {
		tag, err := tx.Exec(ctx, `
			INSERT INTO rate_limit_counters
				(key_hash, window_seconds, window_start, current_count)
			VALUES ($1, 60, NOW(), 1)`, keyHash)
		if err != nil {
			return idempotency.Outcome{}, err
		}
		return idempotency.Outcome{
			JSON: []byte(fmt.Sprintf(`{"inserted":%d}`, tag.RowsAffected())),
		}, nil
	}
}

func countCounters(t *testing.T, pool *pgxpool.Pool, keyHash []byte) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM rate_limit_counters WHERE key_hash = $1`, keyHash).
		Scan(&n); err != nil {
		t.Fatalf("count counters: %v", err)
	}
	return n
}

func TestOperationDeduplication(t *testing.T) {
	pool := migratedPool(t, "idem_dedupe")
	store := idempotency.NewStore(pool)
	key := sha256.Sum256([]byte("dedupe-key"))
	op := idempotency.Operation{
		Family:      "test.dedupe",
		OwnerKind:   idempotency.OwnerAccount,
		OwnerID:     newUUID(t),
		OperationID: newUUID(t),
		Fingerprint: fp(1),
	}
	runs := 0
	fn := func(ctx context.Context, tx pgx.Tx) (idempotency.Outcome, error) {
		runs++
		return bumpCounter(key[:])(ctx, tx)
	}
	out, replayed, err := store.Execute(context.Background(), op, fn)
	if err != nil || replayed {
		t.Fatalf("first execute: out=%s replayed=%v err=%v", out.JSON, replayed, err)
	}
	out2, replayed2, err := store.Execute(context.Background(), op, fn)
	if err != nil {
		t.Fatalf("retry execute: %v", err)
	}
	if !replayed2 {
		t.Fatal("retry did not replay")
	}
	if !outcomeEqual(t, out.JSON, out2.JSON) {
		t.Fatalf("replayed outcome %s != original %s", out2.JSON, out.JSON)
	}
	if runs != 1 {
		t.Fatalf("fn ran %d times, want 1", runs)
	}
	if n := countCounters(t, pool, key[:]); n != 1 {
		t.Fatalf("committed %d mutation rows, want at-most-once", n)
	}
}

func TestCommitBeforeResponseRetry(t *testing.T) {
	pool := migratedPool(t, "idem_retry")
	key := sha256.Sum256([]byte("retry-key"))
	op := idempotency.Operation{
		Family:      "test.retry",
		OwnerKind:   idempotency.OwnerCharacter,
		OwnerID:     newUUID(t),
		OperationID: newUUID(t),
		Fingerprint: fp(7),
	}
	out, _, err := idempotency.NewStore(pool).Execute(context.Background(), op, bumpCounter(key[:]))
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	// Commit succeeded; simulate the response being lost and the operation
	// retried — a fresh Store (as after a process restart) must reconstruct
	// the same outcome without re-running the mutation.
	out2, replayed, err := idempotency.NewStore(pool).Execute(context.Background(), op, bumpCounter(key[:]))
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !replayed || !outcomeEqual(t, out.JSON, out2.JSON) {
		t.Fatalf("retry reconstructed %s (replayed=%v), want %s", out2.JSON, replayed, out.JSON)
	}
	if n := countCounters(t, pool, key[:]); n != 1 {
		t.Fatalf("committed %d mutation rows, want 1", n)
	}
}

func TestConflictingPayloadRejection(t *testing.T) {
	pool := migratedPool(t, "idem_conflict")
	key := sha256.Sum256([]byte("conflict-key"))
	op := idempotency.Operation{
		Family:      "test.conflict",
		OwnerKind:   idempotency.OwnerAccount,
		OwnerID:     newUUID(t),
		OperationID: newUUID(t),
		Fingerprint: fp(1),
	}
	if _, _, err := idempotency.NewStore(pool).Execute(context.Background(), op, bumpCounter(key[:])); err != nil {
		t.Fatalf("first execute: %v", err)
	}
	conflict := op
	conflict.Fingerprint = fp(2) // same operation ID, different payload
	_, _, err := idempotency.NewStore(pool).Execute(context.Background(), conflict, bumpCounter(key[:]))
	if !errors.Is(err, idempotency.ErrOperationConflict) {
		t.Fatalf("conflicting payload err = %v, want ErrOperationConflict", err)
	}
	if n := countCounters(t, pool, key[:]); n != 1 {
		t.Fatalf("conflicting retry committed; %d mutation rows", n)
	}
}

func TestPostgresUniqueConstraint(t *testing.T) {
	pool := migratedPool(t, "idem_unique")
	var (
		owner  = pgtype.UUID{Bytes: newUUID(t), Valid: true}
		opID   = pgtype.UUID{Bytes: newUUID(t), Valid: true}
		digest = fp(9)
		now    = time.Now().UTC()
	)
	insert := `
		INSERT INTO operations
			(operation_family, owner_kind, owner_id, operation_id,
			 request_fingerprint, outcome, created_at, completed_at)
		VALUES ('test.uniq', 'ACCOUNT', $1, $2, $3, '{}'::jsonb, $4, $4)`
	if _, err := pool.Exec(context.Background(), insert, owner, opID, digest[:], now); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err := pool.Exec(context.Background(), insert, owner, opID, digest[:], now)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("duplicate key err = %v, want SQLSTATE 23505", err)
	}
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM operations`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("operations rows = %d, %v; want 1", n, err)
	}
}

func TestOperationKeyScopedByOwner(t *testing.T) {
	pool := migratedPool(t, "idem_owner")
	key := sha256.Sum256([]byte("owner-key"))
	op := idempotency.Operation{
		Family:      "test.owner",
		OwnerKind:   idempotency.OwnerAccount,
		OwnerID:     newUUID(t),
		OperationID: newUUID(t),
		Fingerprint: fp(3),
	}
	if _, _, err := idempotency.NewStore(pool).Execute(context.Background(), op, bumpCounter(key[:])); err != nil {
		t.Fatalf("owner A execute: %v", err)
	}
	// ADR-0065: the same operation_id under a different owner_id is a
	// different operation and commits separately.
	other := op
	other.OwnerID = newUUID(t)
	key2 := sha256.Sum256([]byte("owner-key-2"))
	if _, replayed, err := idempotency.NewStore(pool).Execute(context.Background(), other, bumpCounter(key2[:])); err != nil || replayed {
		t.Fatalf("owner B execute: replayed=%v err=%v", replayed, err)
	}
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM operations WHERE operation_family = 'test.owner' AND operation_id = $1`,
		pgtype.UUID{Bytes: op.OperationID, Valid: true}).Scan(&n); err != nil || n != 2 {
		t.Fatalf("operations rows for operation_id = %d, %v; want 2", n, err)
	}
}
