package id

import (
	"errors"
	"fmt"
	"log/slog"
)

// Revision identifies one validated content bundle: schema_version +
// content_revision (config.md § Identity, ADR-0001). It travels inside
// compile/runtime diagnostic context: the required "revision" log attribute
// (engineering_conventions.md § 1.2, observability.md) and content compile
// diagnostics (content_authoring_contract.md § 6).
type Revision struct {
	// SchemaVersion is the monotonic bundle schema counter.
	SchemaVersion uint64
	// ContentRevision is the 64-lowercase-hex SHA-256 of the canonicalized
	// catalogs (content_authoring_contract.md § 5).
	ContentRevision string
}

// ContentRevisionLen is the exact length of a SHA-256 hex content_revision.
const ContentRevisionLen = 64

// ErrInvalidContentRevision — a content_revision that is not exactly 64
// lowercase hex characters.
var ErrInvalidContentRevision = errors.New("id: invalid content revision")

// ValidateContentRevision enforces the canonical content_revision form:
// exactly 64 lowercase hex characters.
func ValidateContentRevision(s string) error {
	if len(s) != ContentRevisionLen {
		return fmt.Errorf("%w: %d chars, want %d lowercase hex", ErrInvalidContentRevision, len(s), ContentRevisionLen)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return fmt.Errorf("%w: %q is not lowercase hex", ErrInvalidContentRevision, s)
		}
	}
	return nil
}

// Validate checks that r carries a canonical content_revision.
func (r Revision) Validate() error {
	return ValidateContentRevision(r.ContentRevision)
}

// String renders the diagnostic-context form carried in compile and runtime
// diagnostics, e.g. "schema_version=2 content_revision=0123abcd…".
func (r Revision) String() string {
	return fmt.Sprintf("schema_version=%d content_revision=%s", r.SchemaVersion, r.ContentRevision)
}

// LogValue is the structured form used as the "revision" log attribute
// required on business/error logs (engineering_conventions.md § 1.2).
func (r Revision) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("schema_version", r.SchemaVersion),
		slog.String("content_revision", r.ContentRevision),
	)
}
