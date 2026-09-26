package id

import (
	"errors"
	"math"
)

// RuntimeEntityID is a transient unsigned 64-bit identifier of a simulation
// entity (monster, projectile, …) inside one owner lifetime (ids.md § Runtime
// Entity IDs; wire type uint64, protobuf_conventions.md § 6). Value 0 is
// reserved for "none" on the wire (messages.md entity fields), so issued IDs
// start at 1.
type RuntimeEntityID uint64

// EntityScope is the owner/epoch scope of runtime entity IDs:
// simulation_owner_id + ownership_epoch (ids.md; concurrency.md § Entity
// Ownership). An ID is unique only within its scope and is never a durable
// identity.
type EntityScope struct {
	OwnerID uint64 // simulation_owner_id
	Epoch   uint64 // ownership_epoch
}

// ScopedID binds a runtime entity ID to its scope. Every client reference is
// revalidated against the current owner/session/AOI/lifecycle — an out-of-scope
// reference is rejected (ids.md § Validation).
type ScopedID struct {
	Scope EntityScope
	ID    RuntimeEntityID
}

var (
	// ErrEntityIDExhausted — the scope issued every uint64 value.
	ErrEntityIDExhausted = errors.New("id: runtime entity ID space exhausted")
	// ErrEntityIDOutOfScope — a reference whose owner/epoch is not the current
	// authority, or the reserved none value 0. Maps to a stale-input/scope
	// rejection at the edge.
	ErrEntityIDOutOfScope = errors.New("id: runtime entity ID outside owner/epoch scope")
)

// EntityAllocator issues monotonic runtime entity IDs inside one EntityScope.
// It is owned by a single-writer partition loop (realtime_loop.md) and is not
// safe for concurrent use.
type EntityAllocator struct {
	scope EntityScope
	next  uint64 // last issued value; the next issue is next+1
}

// NewEntityAllocator returns an allocator issuing 1, 2, 3, … inside scope.
func NewEntityAllocator(scope EntityScope) *EntityAllocator {
	return &EntityAllocator{scope: scope}
}

// Next issues the next scoped ID. Exhaustion (after 2^64-1 issues) is reported
// rather than wrapping onto 0 or re-issuing a value.
func (a *EntityAllocator) Next() (ScopedID, error) {
	if a.next == math.MaxUint64 {
		return ScopedID{}, ErrEntityIDExhausted
	}
	a.next++
	return ScopedID{Scope: a.scope, ID: RuntimeEntityID(a.next)}, nil
}

// Scope returns the scope this allocator issues in.
func (a *EntityAllocator) Scope() EntityScope { return a.scope }

// Contains reports whether ref is a real entity ID issued inside s.
func (s EntityScope) Contains(ref ScopedID) bool {
	return ref.Scope == s && ref.ID != 0
}

// Validate rejects an out-of-scope or none (0) runtime reference.
func (s EntityScope) Validate(ref ScopedID) error {
	if !s.Contains(ref) {
		return ErrEntityIDOutOfScope
	}
	return nil
}
