package id

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

// OperationKey is the durable uniqueness scope of a retriable value mutation:
// the operations PRIMARY KEY (operation_family, owner_id, operation_id)
// (data_model.md § operations). Per-owner scoping means a client-chosen
// operation_id can never collide with another owner's operation.
type OperationKey struct {
	Family  string // operation_family — stable dotted family
	OwnerID UUID   // account/character/guild ID; WORLD_OWNER_ID for world jobs
	ID      UUID   // operation_id
}

// OperationFamilyMaxBytes is the operations.operation_family bound
// (VARCHAR(48), data_model.md § operations).
const OperationFamilyMaxBytes = 48

var (
	// ErrInvalidOperationFamily — empty, over-48-byte or non-dotted family
	// name. Maps to PROTOCOL_MALFORMED at the edge.
	ErrInvalidOperationFamily = errors.New("id: invalid operation family")
	// ErrOperationConflict — the same operation key was reused with a
	// conflicting payload (ids.md § Operation IDs). Maps to OPERATION_CONFLICT.
	ErrOperationConflict = errors.New("id: operation ID reused with conflicting payload")
)

// ValidateOperationFamily enforces the declared shape of operation_family:
// a stable dotted family token (same grammar as static content IDs — e.g.
// "inventory.mutate", "auction.buy", "sim.kill_settlement") of at most
// OperationFamilyMaxBytes.
func ValidateOperationFamily(family string) error {
	if len(family) == 0 || len(family) > OperationFamilyMaxBytes {
		return fmt.Errorf("%w: %d bytes", ErrInvalidOperationFamily, len(family))
	}
	if !staticIDPattern.MatchString(family) {
		return fmt.Errorf("%w: %q is not a stable dotted family", ErrInvalidOperationFamily, family)
	}
	return nil
}

// ValidateClientOperationID enforces ids.md § Operation IDs for
// client-initiated requests (C2S message / HTTPS mutation): the client
// generates a UUID v4 once per user intent and reuses it on retry; the server
// rejects nil, non-v4 or malformed IDs (PROTOCOL_MALFORMED at the edge).
// Server-initiated jobs use ServerJobUUID (UUID v5) and must not be validated
// here.
func ValidateClientOperationID(u UUID) error {
	if err := ValidateUUID(u); err != nil {
		return err
	}
	if u.Version() != 4 {
		return ErrUUIDNotV4
	}
	return nil
}

// ClientOperationIDFromBytes decodes the 16-byte wire form
// (protobuf_conventions.md § 6) and applies ValidateClientOperationID.
func ClientOperationIDFromBytes(b []byte) (UUID, error) {
	u, err := UUIDFromBytes(b)
	if err != nil {
		return Nil, err
	}
	if err := ValidateClientOperationID(u); err != nil {
		return Nil, err
	}
	return u, nil
}

// OperationFingerprint is the SHA-256 of the canonical invariant-relevant
// request fields, stored in operations.request_fingerprint (data_model.md §
// operations).
func OperationFingerprint(canonicalRequest []byte) [32]byte {
	return sha256.Sum256(canonicalRequest)
}

// CheckOperationReplay implements the replay rule of ids.md § Operation IDs for
// one operation key: same committed request fingerprint → return/reconstruct
// the prior outcome (nil); conflicting fingerprint → ErrOperationConflict
// (OPERATION_CONFLICT at the edge). It never re-executes the mutation.
func CheckOperationReplay(committedFingerprint, retryFingerprint [32]byte) error {
	if committedFingerprint != retryFingerprint {
		return ErrOperationConflict
	}
	return nil
}
