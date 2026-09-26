package id

import (
	"errors"
	"fmt"
	"regexp"
)

// StaticIDMaxBytes is the wire bound of a content ID string
// (docs/05_network/protobuf_conventions.md § 6: max 128 bytes).
const StaticIDMaxBytes = 128

// staticIDPattern is the canonical grammar of docs/06_data/ids.md § Static
// Content IDs: [a-z0-9]+(\.[a-z0-9_]+)+ — ASCII lowercase, segments separated
// by ".", "_" allowed inside non-first segments only, at least two segments.
var staticIDPattern = regexp.MustCompile(`\A[a-z0-9]+(\.[a-z0-9_]+)+\z`)

// ErrInvalidStaticID — a string that is not a canonical static content ID.
// Maps to PROTOCOL_MALFORMED at the edge; at compile time it is a validation
// error of the owning catalog (config.md § Validation never silently rewrites).
var ErrInvalidStaticID = errors.New("id: invalid static content ID")

// ValidateStaticID enforces the canonical static content ID grammar and the
// 128-byte wire bound. The u_minh / rung_u_minh region-token split of ids.md
// is intentional: both forms satisfy the grammar and must not be flagged.
func ValidateStaticID(s string) error {
	if s == "" {
		return fmt.Errorf("%w: empty", ErrInvalidStaticID)
	}
	if len(s) > StaticIDMaxBytes {
		return fmt.Errorf("%w: %d bytes exceeds %d", ErrInvalidStaticID, len(s), StaticIDMaxBytes)
	}
	if !staticIDPattern.MatchString(s) {
		return fmt.Errorf("%w: %q does not match [a-z0-9]+(\\.[a-z0-9_]+)+", ErrInvalidStaticID, s)
	}
	return nil
}
