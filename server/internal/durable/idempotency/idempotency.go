// Package idempotency implements the durable operation-dedupe / result-replay
// primitive of database.md § Idempotency and data_model.md § operations
// (ADR-0065).
//
// Every retriable value mutation persists one `operations` row keyed
// (operation_family, owner_id, operation_id) carrying the SHA-256 request
// fingerprint and the committed outcome. The row is inserted LAST in the
// committing transaction (database.md § Lock Order: "`operations` rows are
// inserted last in the same transaction"), so only committed outcomes ever
// exist — a retry after commit-before-response returns/reconstructs the
// stored outcome instead of re-executing the mutation.
package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thinhthan/internal/core/id"
)

// OwnerKind is the invariant scope an operation's owner belongs to
// (operations.owner_kind CHECK).
type OwnerKind string

const (
	OwnerAccount   OwnerKind = "ACCOUNT"
	OwnerCharacter OwnerKind = "CHARACTER"
	OwnerGuild     OwnerKind = "GUILD"
	OwnerWorld     OwnerKind = "WORLD"
)

// WorldOwnerID is the reserved operations.owner_id for world-scoped jobs
// (data_model.md § accounts schema). It is not an account row.
var WorldOwnerID = id.UUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}

// ErrOperationConflict reports a retry whose operation key was already
// committed under a different request fingerprint — the same operation ID
// carrying a different payload (data_model.md § operations). Mapped to the
// OPERATION_CONFLICT wire code at the edge.
var ErrOperationConflict = errors.New("idempotency: operation conflict")

// maxSerializationRetries bounds the deadlock/serialization retry loop.
// database.md § Transactions allows such retries only for idempotent
// operations reusing the same operation ID — exactly what Execute does.
const maxSerializationRetries = 5

// Operation is the stable identity of one retriable value mutation.
// OperationID is generated once by the caller (client intent UUID or a
// deterministic id.UUIDv5 grant key) and reused across retries.
// Fingerprint is the SHA-256 of the canonical invariant-relevant request
// fields: a retry reusing the key with a different payload is a conflict.
type Operation struct {
	Family      string
	OwnerKind   OwnerKind
	OwnerID     id.UUID
	OperationID id.UUID
	Fingerprint [32]byte
}

// Outcome is the committed result of an operation: bounded, schema-versioned
// JSONB (IDs, amounts) replayed verbatim on deduped retries.
type Outcome struct {
	JSON json.RawMessage
}

// Store executes operations against the operations table.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore builds a Store on an existing pool; the caller owns the pool's
// lifecycle.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Execute runs fn inside one READ COMMITTED transaction and commits the
// operations row with it, so "committed outcome" and "operation record" are
// always atomic (save_rules.md: durable success -> DB commit first; a lost
// response != a lost or duplicated commit).
//
// Semantics:
//   - First call: fn runs; the row commits with fn's Outcome.
//   - Retry with same key + same fingerprint: fn is NOT re-run; the stored
//     outcome returns with replayed=true.
//   - Retry with same key + different fingerprint: ErrOperationConflict.
//   - fn error: the transaction rolls back — no partial mutation, no
//     operations row; a later retry executes fresh.
func (s *Store) Execute(ctx context.Context, op Operation, fn func(ctx context.Context, tx pgx.Tx) (Outcome, error)) (Outcome, bool, error) {
	var zero Outcome
	if err := op.validate(); err != nil {
		return zero, false, err
	}
	for attempt := 0; ; attempt++ {
		// Only committed outcomes exist (the row lands last in its
		// transaction), so any visible row is authoritative.
		outcome, found, err := s.lookup(ctx, op)
		if err != nil {
			return zero, false, err
		}
		if found {
			return outcome, true, nil
		}

		outcome, replayed, err := s.executeOnce(ctx, op, fn)
		if err == nil {
			return outcome, replayed, nil
		}
		if errors.Is(err, ErrOperationConflict) {
			return zero, false, err
		}
		if isSerializationError(err) && attempt < maxSerializationRetries-1 {
			continue
		}
		return zero, false, err
	}
}

func (op Operation) validate() error {
	switch op.OwnerKind {
	case OwnerAccount, OwnerCharacter, OwnerGuild, OwnerWorld:
	default:
		return fmt.Errorf("idempotency: invalid owner_kind %q", op.OwnerKind)
	}
	if op.Family == "" {
		return fmt.Errorf("idempotency: empty operation_family")
	}
	return nil
}

// lookup reads the committed operations row for op's key; found=false when
// the operation never committed. A fingerprint match is required — a
// mismatch is ErrOperationConflict (same ID, different payload).
func (s *Store) lookup(ctx context.Context, op Operation) (Outcome, bool, error) {
	var (
		fingerprint []byte
		outcome     []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT request_fingerprint, outcome
		FROM operations
		WHERE operation_family = $1 AND owner_id = $2 AND operation_id = $3`,
		op.Family, toPgUUID(op.OwnerID), toPgUUID(op.OperationID)).
		Scan(&fingerprint, &outcome)
	if errors.Is(err, pgx.ErrNoRows) {
		return Outcome{}, false, nil
	}
	if err != nil {
		return Outcome{}, false, fmt.Errorf("idempotency: read operation: %w", err)
	}
	if len(fingerprint) != len(op.Fingerprint) {
		return Outcome{}, false, fmt.Errorf("idempotency: stored fingerprint is %d bytes", len(fingerprint))
	}
	for i := range fingerprint {
		if fingerprint[i] != op.Fingerprint[i] {
			return Outcome{}, false, ErrOperationConflict
		}
	}
	return Outcome{JSON: json.RawMessage(outcome)}, true, nil
}

// executeOnce runs fn and inserts the operations row as the last write of
// the committing transaction. A concurrent retry that committed between our
// pre-check and our INSERT is reported by ON CONFLICT DO NOTHING; the
// winner's row is then replayed and our in-flight work rolls back.
func (s *Store) executeOnce(ctx context.Context, op Operation, fn func(ctx context.Context, tx pgx.Tx) (Outcome, error)) (Outcome, bool, error) {
	var zero Outcome
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return zero, false, fmt.Errorf("idempotency: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	outcome, err := fn(ctx, tx)
	if err != nil {
		return zero, false, err
	}
	payload := []byte(outcome.JSON)
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	now := time.Now().UTC()
	tag, err := tx.Exec(ctx, `
		INSERT INTO operations
			(operation_family, owner_kind, owner_id, operation_id,
			 request_fingerprint, outcome, created_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (operation_family, owner_id, operation_id) DO NOTHING`,
		op.Family, string(op.OwnerKind), toPgUUID(op.OwnerID), toPgUUID(op.OperationID),
		op.Fingerprint[:], payload, now)
	if err != nil {
		return zero, false, fmt.Errorf("idempotency: insert operation: %w", err)
	}
	if tag.RowsAffected() != 1 {
		// A concurrent execution committed this operation between the
		// pre-check and our insert. The row is committed (inserted last in
		// its transaction) and visible to this READ COMMITTED statement
		// snapshot, so read it and discard our own work by rolling back.
		var (
			fingerprint []byte
			stored      []byte
		)
		err := tx.QueryRow(ctx, `
			SELECT request_fingerprint, outcome
			FROM operations
			WHERE operation_family = $1 AND owner_id = $2 AND operation_id = $3`,
			op.Family, toPgUUID(op.OwnerID), toPgUUID(op.OperationID)).
			Scan(&fingerprint, &stored)
		if err != nil {
			return zero, false, fmt.Errorf("idempotency: read raced operation: %w", err)
		}
		if len(fingerprint) != len(op.Fingerprint) {
			return zero, false, ErrOperationConflict
		}
		for i := range fingerprint {
			if fingerprint[i] != op.Fingerprint[i] {
				return zero, false, ErrOperationConflict
			}
		}
		return Outcome{JSON: json.RawMessage(stored)}, true, nil // deferred rollback
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, false, fmt.Errorf("idempotency: commit: %w", err)
	}
	return outcome, false, nil
}

func toPgUUID(u id.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// isSerializationError reports deadlock (40P01) or serialization (40001)
// failures, which may be retried under the same operation ID.
func isSerializationError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "40P01" || pgErr.Code == "40001"
}
