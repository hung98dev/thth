package rng

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"log/slog"
	"math"
	"math/rand/v2"

	"thinhthan/internal/core/id"
)

// SeedLen is the byte length of a server-owned Seed.
const SeedLen = 32

// Seed is the server-owned seed material feeding stream derivation
// (concurrency.md § RNG Concurrency: "server-owned seed"). It is generated
// with crypto/rand and stored/transmitted only by server infrastructure —
// a client can never supply it, so a client can never supply an
// authoritative RNG result (config.md § Client Contract).
type Seed [SeedLen]byte

// NewSeed returns fresh server seed material from crypto/rand
// (ids.md: UUID/secrets remain crypto/rand). A generation failure fails the
// operation: the error is returned, never swallowed.
func NewSeed() (Seed, error) {
	var s Seed
	if _, err := cryptorand.Read(s[:]); err != nil {
		return Seed{}, fmt.Errorf("rng: generate server seed: %w", err)
	}
	return s, nil
}

// IsZero reports whether s is the all-zero seed — never a valid real seed.
func (s Seed) IsZero() bool {
	return s == Seed{}
}

// Kind is the typed RNG context kind: which authoritative stream family a
// roll belongs to (combat rolls, enhancement, item secondary rolls,
// spawn-pool choice, drop-table choice, boss scaling resample, fishing
// catch — gameplay.md § Regression Seeds). The kind participates in stream
// derivation, so two kinds can never share one stream's outputs.
//
// Canonical form: one or more lowercase ASCII words joined by "_"
// ([a-z0-9]+(_[a-z0-9]+)*), e.g. "combat", "drop_table", "fishing_catch".
type Kind string

// Kind grammar limits.
const (
	KindMaxLen = 64
)

// Context is the typed RNG context a stream belongs to: Kind selects the
// stream family and Subject carries the stable authority ID that scopes the
// roll — a partition/instance/source ID or an operation/event ID
// (concurrency.md § RNG Concurrency). The same (revision, context, seed)
// always derives the same stream; different contexts derive independent
// streams.
//
// Subject canonical form: lowercase ASCII segments joined by one of
// ".", "_", ":" or "-" ([a-z0-9]+([._:-][a-z0-9]+)*), covering static
// content IDs, UUIDs and composite keys such as
// "fishing.<character_id>.<utc_date>.<cast_sequence>" (IMP-058).
type Context struct {
	Kind    Kind
	Subject string
}

// Subject grammar limits.
const (
	SubjectMaxLen = 128
)

// Domain errors of this package. They are mapped to wire codes at the edge;
// the core package never emits wire codes itself.
var (
	// ErrInvalidKind — a context kind that is empty, overlong, or outside
	// the canonical grammar.
	ErrInvalidKind = errors.New("rng: invalid context kind")
	// ErrInvalidSubject — a context subject that is empty, overlong, or
	// outside the canonical grammar.
	ErrInvalidSubject = errors.New("rng: invalid context subject")
	// ErrZeroSeed — the all-zero seed where real seed material is required.
	ErrZeroSeed = errors.New("rng: zero server seed")
	// ErrInvalidRange — a half-open range that is empty or not finite
	// (lo >= hi, NaN/Inf bound, infinite span, or n == 0).
	ErrInvalidRange = errors.New("rng: invalid range")
	// ErrInvalidBasisPoints — a basis-point threshold above
	// BasisPointsTotal.
	ErrInvalidBasisPoints = errors.New("rng: basis points out of range")
	// ErrInvalidWeights — an empty weight set, an all-zero weight set, or a
	// weight sum that overflows uint64.
	ErrInvalidWeights = errors.New("rng: invalid weight set")
)

// Validate enforces the canonical Context form.
func (c Context) Validate() error {
	if err := validateKind(c.Kind); err != nil {
		return err
	}
	return validateSubject(c.Subject)
}

func validateKind(k Kind) error {
	s := string(k)
	if len(s) == 0 || len(s) > KindMaxLen {
		return fmt.Errorf("%w: %d bytes, want 1..%d", ErrInvalidKind, len(s), KindMaxLen)
	}
	prevUnderscore := true // leading "_" rejected; also catches ".."-free single-word rule
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', '0' <= c && c <= '9':
			prevUnderscore = false
		case c == '_':
			if prevUnderscore {
				return fmt.Errorf("%w: %q has empty word", ErrInvalidKind, s)
			}
			prevUnderscore = true
		default:
			return fmt.Errorf("%w: %q contains %q, want [a-z0-9_]", ErrInvalidKind, s, string(c))
		}
	}
	if prevUnderscore {
		return fmt.Errorf("%w: %q ends with '_'", ErrInvalidKind, s)
	}
	return nil
}

func validateSubject(s string) error {
	if len(s) == 0 || len(s) > SubjectMaxLen {
		return fmt.Errorf("%w: %d bytes, want 1..%d", ErrInvalidSubject, len(s), SubjectMaxLen)
	}
	prevSep := true // leading separator rejected
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', '0' <= c && c <= '9':
			prevSep = false
		case c == '.' || c == '_' || c == ':' || c == '-':
			if prevSep {
				return fmt.Errorf("%w: %q has empty segment", ErrInvalidSubject, s)
			}
			prevSep = true
		default:
			return fmt.Errorf("%w: %q contains %q, want [a-z0-9._:-]", ErrInvalidSubject, s, string(c))
		}
	}
	if prevSep {
		return fmt.Errorf("%w: %q ends with a separator", ErrInvalidSubject, s)
	}
	return nil
}

// String renders the diagnostic form, e.g. "combat:combat_event.tick.42".
func (c Context) String() string {
	return string(c.Kind) + ":" + c.Subject
}

// LogValue is the structured form used as a log attribute where a roll is
// diagnosed (engineering_conventions.md § 1.2).
func (c Context) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", string(c.Kind)),
		slog.String("subject", c.Subject),
	)
}

// BasisPointsTotal is the integer basis-point denominator used by gameplay
// probability (config.md § Probability): 10000 bp = 100%.
const BasisPointsTotal = 10000

// Stream is one deterministic PCG-64 (math/rand/v2) stream owned by exactly
// one RNG context. A Stream is not safe for concurrent use: each
// deterministic context owns its stream on the single-writer goroutine of
// its partition/scope, so goroutine scheduling can never change roll
// outcomes (concurrency.md § RNG Concurrency).
type Stream struct {
	r *rand.Rand
}

// domainSep separates this derivation from every other SHA-256 use in the
// codebase. Changing it changes every stream: it is part of the frozen
// derivation contract pinned by the testdata regression fixture.
const domainSep = "thinhthan/rng/pcg64/1\x00"

// New derives the deterministic PCG-64 stream owned by ctx under
// contentRevision and seed. The same (contentRevision, ctx, seed) triple
// always derives the same stream (config.md § Probability,
// gameplay.md § Regression Seeds); any difference yields an independent
// stream.
//
// contentRevision is the canonical 64-lowercase-hex content revision
// (id.ValidateContentRevision, ADR-0001): the stream pins the revision its
// rolls were produced under. seed is server-owned; ctx must satisfy
// Context.Validate.
func New(contentRevision string, ctx Context, seed Seed) (*Stream, error) {
	if err := id.ValidateContentRevision(contentRevision); err != nil {
		return nil, err
	}
	if err := ctx.Validate(); err != nil {
		return nil, err
	}
	if seed.IsZero() {
		return nil, ErrZeroSeed
	}
	state, seq := derive(contentRevision, ctx, seed)
	return &Stream{r: rand.New(rand.NewPCG(state, seq))}, nil
}

// derive maps the authority inputs onto the (state, sequence) pair of one
// PCG-64 stream via SHA-256 over a domain-separated, length-prefixed
// encoding — no field can shift into a neighbor's bytes.
func derive(contentRevision string, ctx Context, seed Seed) (state, seq uint64) {
	h := sha256.New()
	h.Write([]byte(domainSep))
	writeField(h, []byte(ctx.Kind))
	writeField(h, []byte(ctx.Subject))
	writeField(h, []byte(contentRevision))
	h.Write(seed[:])
	d := h.Sum(nil)
	return binary.BigEndian.Uint64(d[0:8]), binary.BigEndian.Uint64(d[8:16])
}

func writeField(h hash.Hash, b []byte) {
	var l [8]byte
	binary.BigEndian.PutUint64(l[:], uint64(len(b)))
	h.Write(l[:])
	h.Write(b)
}

// Uint64 returns the next raw 64-bit draw of the stream.
func (s *Stream) Uint64() uint64 {
	return s.r.Uint64()
}

// Uint64N returns the next draw uniformly mapped onto [0, n). n == 0 is an
// invalid range.
func (s *Stream) Uint64N(n uint64) (uint64, error) {
	if n == 0 {
		return 0, fmt.Errorf("%w: Uint64N(0)", ErrInvalidRange)
	}
	return s.r.Uint64N(n), nil
}

// Float64 is rand_float(lo, hi): a half-open sample on [lo, hi) from the
// owning stream (concurrency.md § RNG Concurrency, stats.md). lo, hi and the
// span hi-lo must be finite and lo < hi.
func (s *Stream) Float64(lo, hi float64) (float64, error) {
	if !(lo < hi) || math.IsInf(hi-lo, 0) {
		return 0, fmt.Errorf("%w: rand_float(%v, %v) is not a finite half-open range", ErrInvalidRange, lo, hi)
	}
	return lo + (hi-lo)*s.r.Float64(), nil
}

// RollBasisPoints consumes one draw and reports whether it lands under the
// bp threshold: Uint64N(BasisPointsTotal) < bp (config.md § Probability —
// chance_bp entries). bp <= BasisPointsTotal; bp == 0 always misses and
// bp == BasisPointsTotal always hits while still consuming one draw, so the
// stream state never depends on the threshold value.
func (s *Stream) RollBasisPoints(bp uint64) (bool, error) {
	if bp > BasisPointsTotal {
		return false, fmt.Errorf("%w: %d > %d", ErrInvalidBasisPoints, bp, BasisPointsTotal)
	}
	v, err := s.Uint64N(BasisPointsTotal)
	if err != nil {
		return false, err
	}
	return v < bp, nil
}

// WeightedIndex performs one bounded weighted selection over ordered
// candidates: it draws once and returns the index i in [0, len(weights))
// whose cumulative-weight interval contains the draw, preserving candidate
// order (config.md § Probability — integer weights). The weight set must be
// non-empty, must not be all zero, and its sum must not overflow uint64.
func (s *Stream) WeightedIndex(weights []uint64) (int, error) {
	if len(weights) == 0 {
		return -1, fmt.Errorf("%w: empty weight set", ErrInvalidWeights)
	}
	var total uint64
	for _, w := range weights {
		if w > math.MaxUint64-total {
			return -1, fmt.Errorf("%w: weight sum overflows uint64", ErrInvalidWeights)
		}
		total += w
	}
	if total == 0 {
		return -1, fmt.Errorf("%w: all-zero weight set", ErrInvalidWeights)
	}
	v, err := s.Uint64N(total)
	if err != nil {
		return -1, err
	}
	for i, w := range weights {
		if v < w {
			return i, nil
		}
		v -= w
	}
	return -1, fmt.Errorf("%w: draw outside weight sum", ErrInvalidWeights) // unreachable while v < total
}
