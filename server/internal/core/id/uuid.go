package id

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
)

// UUID is a 16-byte RFC 4122 UUID in network (big-endian) order — the canonical
// durable and wire form (docs/06_data/ids.md § Durable IDs,
// docs/05_network/protobuf_conventions.md § 6). The canonical text form is
// lowercase 8-4-4-4-12.
type UUID [16]byte

// Nil is the all-zero UUID. It is never a valid real ID and is rejected wherever
// an ID is required (ids.md § Validation).
var Nil UUID

// Domain errors of this package. They are mapped 1:1 to the wire codes of
// docs/05_network/errors.md at the edge; the core package never emits wire
// codes itself.
var (
	// ErrUUIDMalformed — not parseable as a UUID (bad length, bad hyphen
	// positions, non-hex text, non-16-byte binary). Maps to PROTOCOL_MALFORMED.
	ErrUUIDMalformed = errors.New("id: malformed UUID")
	// ErrUUIDNil — the zero UUID where a real ID is required. Maps to
	// PROTOCOL_MALFORMED.
	ErrUUIDNil = errors.New("id: nil/zero UUID")
	// ErrUUIDNotV4 — a UUID whose version field is not 4 where the contract
	// requires v4 (client-generated operation IDs). Maps to PROTOCOL_MALFORMED.
	ErrUUIDNotV4 = errors.New("id: UUID is not version 4")
	// ErrUUIDNotRFC4122 — variant bits do not encode the RFC 4122 variant.
	// Maps to PROTOCOL_MALFORMED.
	ErrUUIDNotRFC4122 = errors.New("id: UUID is not the RFC 4122 variant")
)

// NewUUIDv4 returns a fresh RFC 4122 UUID v4 produced by authoritative Go code
// from crypto/rand (ids.md § Durable IDs). A generation failure fails the
// operation: the error is returned, never swallowed.
func NewUUIDv4() (UUID, error) {
	var u UUID
	if _, err := rand.Read(u[:]); err != nil {
		return Nil, fmt.Errorf("id: generate UUID v4: %w", err)
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant RFC 4122
	return u, nil
}

// ParseUUID parses the 8-4-4-4-12 text form. Hex digits may be upper- or
// lowercase; String always emits the canonical lowercase form. Anything that
// is not exactly 36 characters with hyphens at 8/13/18/23 is
// ErrUUIDMalformed.
func ParseUUID(s string) (UUID, error) {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return Nil, ErrUUIDMalformed
	}
	var u UUID
	src := 0
	for j := 0; j < 16; j++ {
		if src == 8 || src == 13 || src == 18 || src == 23 {
			src++
		}
		hi, ok1 := hexNibble(s[src])
		lo, ok2 := hexNibble(s[src+1])
		if !ok1 || !ok2 {
			return Nil, ErrUUIDMalformed
		}
		u[j] = hi<<4 | lo
		src += 2
	}
	return u, nil
}

// UUIDFromBytes converts the wire/binary form: exactly 16 bytes, network order
// (protobuf_conventions.md § 6). Any other length is ErrUUIDMalformed; callers
// handling optional fields apply "empty = absent" before calling.
func UUIDFromBytes(b []byte) (UUID, error) {
	if len(b) != 16 {
		return Nil, ErrUUIDMalformed
	}
	var u UUID
	copy(u[:], b)
	return u, nil
}

// String renders the canonical lowercase 8-4-4-4-12 text form (ids.md §
// Serialization). HTTPS JSON uses this form.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

// IsZero reports whether u is the all-zero (nil) UUID.
func (u UUID) IsZero() bool { return u == Nil }

// Version returns the RFC 4122 version field (4 = random, 5 = SHA-1 name).
func (u UUID) Version() int { return int(u[6] >> 4) }

// IsRFC4122 reports whether the variant field encodes the RFC 4122 variant
// (10xxxxxx in byte 8).
func (u UUID) IsRFC4122() bool { return u[8]&0xc0 == 0x80 }

// ValidateUUID enforces "a real ID is required" (ids.md § Validation): the UUID
// must be non-zero and carry the RFC 4122 variant. Callers needing the
// client-operation-ID contract use ValidateClientOperationID instead.
func ValidateUUID(u UUID) error {
	if u.IsZero() {
		return ErrUUIDNil
	}
	if !u.IsRFC4122() {
		return ErrUUIDNotRFC4122
	}
	return nil
}

// UUIDv5 derives a deterministic name-based UUID per RFC 4122 § 4.3: SHA-1 over
// namespace bytes then the name, truncated to 16 bytes with version 5 and
// RFC 4122 variant bits set (ids.md § Deterministic Content-Grant Idempotency
// Keys, § Operation IDs). Same (namespace, name) input always yields the same
// UUID.
func UUIDv5(namespace UUID, name string) UUID {
	h := sha1.New()
	h.Write(namespace[:])
	h.Write([]byte(name))
	sum := h.Sum(nil) // 20 bytes
	var u UUID
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | 0x50 // version 5
	u[8] = (u[8] & 0x3f) | 0x80 // variant RFC 4122
	return u
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
