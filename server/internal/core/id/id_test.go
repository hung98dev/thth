package id

import (
	"errors"
	"log/slog"
	"math"
	"strings"
	"testing"
)

func TestUUIDv4(t *testing.T) {
	u, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	if u.IsZero() {
		t.Fatal("NewUUIDv4 returned the zero UUID")
	}
	if got := u.Version(); got != 4 {
		t.Fatalf("version = %d, want 4", got)
	}
	if !u.IsRFC4122() {
		t.Fatal("variant bits are not RFC 4122")
	}
	// Canonical text form round-trips: lowercase 8-4-4-4-12.
	s := u.String()
	if len(s) != 36 {
		t.Fatalf("String() length = %d, want 36", len(s))
	}
	if s != strings.ToLower(s) {
		t.Fatalf("String() = %q is not canonical lowercase", s)
	}
	back, err := ParseUUID(s)
	if err != nil {
		t.Fatalf("ParseUUID(%q): %v", s, err)
	}
	if back != u {
		t.Fatalf("round-trip mismatch: %q parsed as %v, want %v", s, back, u)
	}
	// Binary (16-byte wire) form round-trips.
	bin, err := UUIDFromBytes(u[:])
	if err != nil {
		t.Fatalf("UUIDFromBytes: %v", err)
	}
	if bin != u {
		t.Fatalf("binary round-trip mismatch: %v, want %v", bin, u)
	}
	// A generated v4 passes the client operation ID contract.
	if err := ValidateClientOperationID(u); err != nil {
		t.Fatalf("ValidateClientOperationID(generated v4): %v", err)
	}
}

func TestUUIDv4Uniqueness(t *testing.T) {
	const n = 10000
	seen := make(map[UUID]struct{}, n)
	for i := 0; i < n; i++ {
		u, err := NewUUIDv4()
		if err != nil {
			t.Fatalf("NewUUIDv4 iteration %d: %v", i, err)
		}
		if _, dup := seen[u]; dup {
			t.Fatalf("duplicate UUID at iteration %d: %v", i, u)
		}
		seen[u] = struct{}{}
	}
}

func TestParseUUIDRejections(t *testing.T) {
	canonical := "f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b"
	if _, err := ParseUUID(canonical); err != nil {
		t.Fatalf("canonical form rejected: %v", err)
	}
	// Uppercase hex parses (input leniency); String emits canonical lowercase.
	if u, err := ParseUUID(strings.ToUpper(canonical)); err != nil || u.String() != canonical {
		t.Fatalf("uppercase parse = %v, %v", u, err)
	}

	malformed := []string{
		"",                                       // empty
		"f7a3d2b14e8c4a2f9b3e6d1c5f8e7a2b",       // no hyphens
		"f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2",    // 35 chars
		"f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2bc",  // 37 chars
		"f7a3d2b14-e8c-4a2f-9b3e-6d1c5f8e7a2b",   // hyphen at wrong position
		"f7a3d2b1-4e8c4-a2f9-b3e6-d1c5f8e7a2b",   // all hyphens misplaced
		"g7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b",   // non-hex 'g'
		"f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2 ",   // trailing space
		"{f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b}", // braces
		"f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2\n",  // trailing newline
	}
	for _, s := range malformed {
		if _, err := ParseUUID(s); !errors.Is(err, ErrUUIDMalformed) {
			t.Fatalf("ParseUUID(%q) = %v, want ErrUUIDMalformed", s, err)
		}
	}

	for _, n := range []int{0, 8, 15, 17} {
		if _, err := UUIDFromBytes(make([]byte, n)); !errors.Is(err, ErrUUIDMalformed) {
			t.Fatalf("UUIDFromBytes(len=%d) = %v, want ErrUUIDMalformed", n, err)
		}
	}

	// Zero UUID parses as text but is rejected where a real ID is required.
	zero, err := ParseUUID("00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("ParseUUID(zero): %v", err)
	}
	if !zero.IsZero() {
		t.Fatal("parsed zero UUID is not IsZero")
	}
	if err := ValidateUUID(zero); !errors.Is(err, ErrUUIDNil) {
		t.Fatalf("ValidateUUID(zero) = %v, want ErrUUIDNil", err)
	}
	if err := ValidateClientOperationID(zero); !errors.Is(err, ErrUUIDNil) {
		t.Fatalf("ValidateClientOperationID(zero) = %v, want ErrUUIDNil", err)
	}
	// Non-RFC4122 variant is rejected as a real ID.
	notRFC := UUID{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x4f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	if notRFC.IsRFC4122() {
		t.Fatal("corrupted variant still reports RFC 4122")
	}
	if err := ValidateUUID(notRFC); !errors.Is(err, ErrUUIDNotRFC4122) {
		t.Fatalf("ValidateUUID(non-RFC4122) = %v, want ErrUUIDNotRFC4122", err)
	}
	// Non-v4 (e.g. the v5 derived forms) is rejected as a client operation ID.
	v5 := UUIDv5(ContentGrantNamespaceUUID(), "x")
	if v5.Version() != 5 || !v5.IsRFC4122() {
		t.Fatalf("UUIDv5 produced version %d / RFC4122=%v", v5.Version(), v5.IsRFC4122())
	}
	if err := ValidateClientOperationID(v5); !errors.Is(err, ErrUUIDNotV4) {
		t.Fatalf("ValidateClientOperationID(v5) = %v, want ErrUUIDNotV4", err)
	}
	if _, err := ClientOperationIDFromBytes(v5[:]); !errors.Is(err, ErrUUIDNotV4) {
		t.Fatalf("ClientOperationIDFromBytes(v5) = %v, want ErrUUIDNotV4", err)
	}
	// A valid v4 in binary form decodes and validates.
	u4, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	if got, err := ClientOperationIDFromBytes(u4[:]); err != nil || got != u4 {
		t.Fatalf("ClientOperationIDFromBytes(v4) = %v, %v", got, err)
	}
}

func TestUUIDv5DeterministicContentGrant(t *testing.T) {
	// The pinned namespace constant is the one recorded in
	// technology_versions.md § Pinned Content System Constants.
	if got := ContentGrantNamespaceUUID().String(); got != "f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b" {
		t.Fatalf("CONTENT_GRANT_NAMESPACE_UUID = %q, want f7a3d2b1-4e8c-4a2f-9b3e-6d1c5f8e7a2b", got)
	}

	scope := "season.1.cosmetic.kim_non_thien_moc.aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	a := ContentGrantUUID(scope)
	b := ContentGrantUUID(scope)
	if a != b {
		t.Fatalf("same scope produced %v then %v", a, b)
	}
	if c := ContentGrantUUID("season.2.cosmetic.kim_non_thien_moc.aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"); c == a {
		t.Fatal("different grant scope produced the same key")
	}
	if a.Version() != 5 || !a.IsRFC4122() {
		t.Fatalf("grant key %v: version %d, RFC4122=%v", a, a.Version(), a.IsRFC4122())
	}
	// Known-answer vector computed independently with RFC 4122 § 4.3
	// (Python uuid.uuid5) over the pinned namespace.
	if want := "ad0639d7-6f73-5e15-b0fb-6faeeb5c6cef"; a.String() != want {
		t.Fatalf("ContentGrantUUID(%q) = %v, want %v", scope, a, want)
	}
	// The RFC 4122 standard namespaces must not be substituted for the pin.
	dnsNS := UUID{0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8}
	if UUIDv5(dnsNS, scope) == a {
		t.Fatal("grant key matches derivation over the DNS namespace — wrong namespace")
	}
}

func TestValidateStaticContentID(t *testing.T) {
	valid := []string{
		"skill.kim.kiem_quang",
		"monster.rung_u_minh.quoi_rung",
		"zone.rung_u_minh.north",
		"item.u_minh.go_u_minh", // intentional u_minh token in item namespace (ids.md)
		"set.u_minh.hai_manh",
		"class.kim",
		"npc.shopkeeper",
		"a.b",
		"a0.b_1.c2",
	}
	for _, s := range valid {
		if err := ValidateStaticID(s); err != nil {
			t.Fatalf("ValidateStaticID(%q) = %v, want nil", s, err)
		}
	}

	invalid := []string{
		"",                              // empty
		"nodots",                        // single segment
		"Upper.case",                    // uppercase
		"SKILL.KIM",                     // uppercase
		"a..b",                          // empty segment
		".a.b",                          // leading dot
		"a.b.",                          // trailing dot
		"a-b.c",                         // '-' inside segment
		"a_b.c",                         // '_' in first segment (grammar allows it only later)
		"a.b-c",                         // '-' is not a word separator
		"a.b ",                          // trailing space
		"a.b\n",                         // trailing newline
		"kỹ_năng.kim",                   // non-ASCII (Vietnamese diacritics are display data)
		"a.b\tc",                        // tab
		strings.Repeat("a", 129) + ".b", // over 128 bytes
	}
	for _, s := range invalid {
		if err := ValidateStaticID(s); !errors.Is(err, ErrInvalidStaticID) {
			t.Fatalf("ValidateStaticID(%q) = %v, want ErrInvalidStaticID", s, err)
		}
	}
	// Exactly at the wire bound is legal: 128 bytes total.
	if err := ValidateStaticID(strings.Repeat("a", 126) + ".b"); err != nil {
		t.Fatalf("ValidateStaticID(128 bytes) = %v, want nil", err)
	}
}

func TestRuntimeEntityID(t *testing.T) {
	owner := EntityScope{OwnerID: 7, Epoch: 3}
	a := NewEntityAllocator(owner)
	if a.Scope() != owner {
		t.Fatalf("allocator scope = %v, want %v", a.Scope(), owner)
	}

	first, err := a.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if first.ID != 1 {
		t.Fatalf("first issued ID = %d, want 1 (0 is reserved none)", first.ID)
	}
	if first.Scope != owner {
		t.Fatalf("issued scope = %v, want %v", first.Scope, owner)
	}
	second, err := a.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if second.ID != 2 || second.ID <= first.ID {
		t.Fatalf("IDs not monotonic: %d then %d", first.ID, second.ID)
	}
	if err := owner.Validate(first); err != nil {
		t.Fatalf("own scope rejected its own ID: %v", err)
	}

	// Same numeric value under a different owner/epoch is a different identity.
	otherOwner := EntityScope{OwnerID: 8, Epoch: 3}
	alien := ScopedID{Scope: otherOwner, ID: first.ID}
	if err := owner.Validate(alien); !errors.Is(err, ErrEntityIDOutOfScope) {
		t.Fatalf("cross-owner reference = %v, want ErrEntityIDOutOfScope", err)
	}
	otherEpoch := EntityScope{OwnerID: 7, Epoch: 4}
	stale := ScopedID{Scope: otherEpoch, ID: first.ID}
	if err := owner.Validate(stale); !errors.Is(err, ErrEntityIDOutOfScope) {
		t.Fatalf("stale-epoch reference = %v, want ErrEntityIDOutOfScope", err)
	}
	// 0 is "none", never a valid reference.
	if err := owner.Validate(ScopedID{Scope: owner, ID: 0}); !errors.Is(err, ErrEntityIDOutOfScope) {
		t.Fatalf("zero ID = %v, want ErrEntityIDOutOfScope", err)
	}
	// A different scope owns an independent sequence starting at 1.
	b := NewEntityAllocator(otherOwner)
	got, err := b.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if got.ID != 1 || got.Scope != otherOwner {
		t.Fatalf("second allocator issued %v, want ID 1 in its own scope", got)
	}
	if err := otherOwner.Validate(got); err != nil {
		t.Fatalf("other scope rejected its own ID: %v", err)
	}

	// Exhaustion is reported, never wraps onto 0 or re-issues.
	a.next = math.MaxUint64
	if _, err := a.Next(); !errors.Is(err, ErrEntityIDExhausted) {
		t.Fatalf("exhausted Next() = %v, want ErrEntityIDExhausted", err)
	}
}

func TestContentRevisionDiagnosticContext(t *testing.T) {
	hex64 := strings.Repeat("0123456789abcdef", 4)
	rev := Revision{SchemaVersion: 2, ContentRevision: hex64}
	if err := rev.Validate(); err != nil {
		t.Fatalf("canonical revision rejected: %v", err)
	}

	// Compile/runtime diagnostic context carries both identity fields.
	diag := rev.String()
	if !strings.Contains(diag, "schema_version=2") || !strings.Contains(diag, hex64) {
		t.Fatalf("diagnostic context %q lacks schema_version/content_revision", diag)
	}
	// Structured form serves the required "revision" log attribute.
	lv := rev.LogValue()
	if lv.Kind() != slog.KindGroup {
		t.Fatalf("LogValue kind = %v, want group", lv.Kind())
	}
	attrs := lv.Group()
	if len(attrs) != 2 || attrs[0].Key != "schema_version" || attrs[1].Key != "content_revision" {
		t.Fatalf("LogValue attrs = %v", attrs)
	}
	if attrs[0].Value.Uint64() != 2 || attrs[1].Value.String() != hex64 {
		t.Fatalf("LogValue values = %v, %v", attrs[0].Value, attrs[1].Value)
	}

	rejects := []string{
		"",
		strings.Repeat("0", 63), // short
		strings.Repeat("0", 65), // long
		strings.ToUpper(hex64),  // uppercase hex
		strings.Repeat("g", 64), // non-hex
	}
	for _, s := range rejects {
		if err := ValidateContentRevision(s); !errors.Is(err, ErrInvalidContentRevision) {
			t.Fatalf("ValidateContentRevision(%q) = %v, want ErrInvalidContentRevision", s, err)
		}
	}
}

func TestOperationPayloadConsistency(t *testing.T) {
	owner, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	opID, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	key := OperationKey{Family: "inventory.mutate", OwnerID: owner, ID: opID}

	// The request fingerprint is deterministic: same canonical fields ->
	// same fingerprint (retry safe), any change -> different fingerprint.
	fp := OperationFingerprint([]byte("inventory.mutate|slot=3|qty=-1|item.x"))
	if fp != OperationFingerprint([]byte("inventory.mutate|slot=3|qty=-1|item.x")) {
		t.Fatal("fingerprint is not deterministic for identical input")
	}
	if fp == OperationFingerprint([]byte("inventory.mutate|slot=3|qty=-2|item.x")) {
		t.Fatal("different canonical payload produced the same fingerprint")
	}

	// Same key + same committed request -> replay returns the prior outcome.
	if err := CheckOperationReplay(fp, fp); err != nil {
		t.Fatalf("identical retry rejected: %v", err)
	}
	// Same key + conflicting payload -> OPERATION_CONFLICT.
	conflict := OperationFingerprint([]byte("inventory.mutate|slot=3|qty=-5|item.x"))
	if err := CheckOperationReplay(fp, conflict); !errors.Is(err, ErrOperationConflict) {
		t.Fatalf("conflicting retry = %v, want ErrOperationConflict", err)
	}

	// Per-owner scoping: a client-chosen operation_id under another owner is a
	// different key and can never collide with this owner's operation.
	otherOwner, err := NewUUIDv4()
	if err != nil {
		t.Fatalf("NewUUIDv4: %v", err)
	}
	crossOwner := OperationKey{Family: key.Family, OwnerID: otherOwner, ID: key.ID}
	if crossOwner == key {
		t.Fatal("operation keys with different owners compared equal")
	}
	// Same owner but a different family is likewise a different operation.
	crossFamily := OperationKey{Family: "auction.buy", OwnerID: owner, ID: key.ID}
	if crossFamily == key {
		t.Fatal("operation keys with different families compared equal")
	}

	// Families follow the stable dotted form within the VARCHAR(48) bound.
	if err := ValidateOperationFamily("inventory.mutate"); err != nil {
		t.Fatalf("ValidateOperationFamily: %v", err)
	}
	if err := ValidateOperationFamily("sim.kill_settlement"); err != nil {
		t.Fatalf("ValidateOperationFamily(underscored segment): %v", err)
	}
	for _, f := range []string{"", "Inventory.Mutate", "nodots", strings.Repeat("a.", 24) + "a"} {
		if err := ValidateOperationFamily(f); !errors.Is(err, ErrInvalidOperationFamily) {
			t.Fatalf("ValidateOperationFamily(%q) = %v, want ErrInvalidOperationFamily", f, err)
		}
	}
}
