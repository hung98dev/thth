package id

import (
	"strings"
	"testing"
)

func TestServerJobIdDeterministic(t *testing.T) {
	// Same (family, job_key) always recomputes the same operation_id — a retry
	// after a crash derives the identical value before anything is persisted
	// (ids.md § Operation IDs, ADR-0070).
	a := ServerJobUUID("privacy.erasure", "12345678-1234-4234-8234-123456789abc")
	b := ServerJobUUID("privacy.erasure", "12345678-1234-4234-8234-123456789abc")
	if a != b {
		t.Fatalf("same job inputs produced %v then %v", a, b)
	}
	if a.Version() != 5 || !a.IsRFC4122() {
		t.Fatalf("server job ID %v: version %d, RFC4122=%v", a, a.Version(), a.IsRFC4122())
	}

	// Distinct job_key or family yields a distinct operation_id.
	if c := ServerJobUUID("privacy.erasure", "99999999-9999-4999-8999-999999999999"); c == a {
		t.Fatal("different job_key produced the same operation_id")
	}
	if d := ServerJobUUID("economy.rollup", "12345678-1234-4234-8234-123456789abc"); d == a {
		t.Fatal("different operation_family produced the same operation_id")
	}

	// The derived name is exactly "<operation_family>:<job_key>" — derivation
	// over the raw concatenation without the separator must differ.
	withSep := ServerJobUUID("economy.rollup", "2026-09-26:daily")
	if withSep == UUIDv5(ServerJobNamespaceUUID(), "economy.rollup"+"2026-09-26:daily") {
		t.Fatal("job derivation ignores the ':' family/job_key separator")
	}
	if withSep != UUIDv5(ServerJobNamespaceUUID(), "economy.rollup:2026-09-26:daily") {
		t.Fatal("job derivation is not UUIDv5(namespace, family+\":\"+key)")
	}

	// Known-answer vectors computed independently with RFC 4122 § 4.3
	// (Python uuid.uuid5) over the pinned namespace.
	cases := []struct {
		family, key, want string
	}{
		{"privacy.erasure", "12345678-1234-4234-8234-123456789abc", "30e21151-2175-55ff-b867-9d13b2f550af"},
		{"economy.rollup", "2026-09-26:daily", "31a6e7c8-3ce0-5fb9-ab94-c7f28e0d4cf2"},
	}
	for _, c := range cases {
		if got := ServerJobUUID(c.family, c.key).String(); got != c.want {
			t.Fatalf("ServerJobUUID(%q, %q) = %v, want %v", c.family, c.key, got, c.want)
		}
	}
}

func TestServerJobNamespacePinned(t *testing.T) {
	// SERVER_JOB_NAMESPACE_UUID is the immutable pin of
	// technology_versions.md § Pinned Content System Constants (ADR-0070).
	got := ServerJobNamespaceUUID().String()
	if got != "64d34c40-8657-462b-887f-5970db9eaa5f" {
		t.Fatalf("SERVER_JOB_NAMESPACE_UUID = %q, want 64d34c40-8657-462b-887f-5970db9eaa5f", got)
	}
	// A server-job operation_id is a real RFC 4122 UUID, not nil.
	if err := ValidateUUID(ServerJobUUID("privacy.erasure", "x")); err != nil {
		t.Fatalf("server job ID failed generic UUID validation: %v", err)
	}
	// The two pinned namespaces are distinct constants.
	if ServerJobNamespaceUUID() == ContentGrantNamespaceUUID() {
		t.Fatal("SERVER_JOB_NAMESPACE_UUID equals CONTENT_GRANT_NAMESPACE_UUID")
	}
	// Neither pin is the all-zero UUID or an RFC 4122 standard namespace.
	if ServerJobNamespaceUUID().IsZero() || strings.HasPrefix(got, "6ba7b810") {
		t.Fatalf("namespace pin degenerate: %q", got)
	}
}
