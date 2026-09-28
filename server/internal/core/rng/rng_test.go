package rng

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"os"
	"testing"

	"thinhthan/internal/core/id"
)

// testRevision is a fixed canonical content revision (64 lowercase hex).
const testRevision = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// testSeed is the fixed server seed used by tests that do not go through the
// regression fixture.
var testSeed = func() (s Seed) {
	for i := range s {
		s[i] = byte(i*16 + 7)
	}
	return s
}()

func mustStream(t *testing.T, rev string, ctx Context, seed Seed) *Stream {
	t.Helper()
	st, err := New(rev, ctx, seed)
	if err != nil {
		t.Fatalf("New(%q, %v, seed): %v", rev, ctx, err)
	}
	return st
}

// regressionFixture is the on-disk schema of testdata/pcg64_regression.json.
// The fixture is immutable: it pins the derivation contract so a change in
// seed derivation, PCG-64 wiring or any draw mapping fails loudly.
type regressionFixture struct {
	Schema          string   `json:"schema"`
	ContentRevision string   `json:"content_revision"`
	ServerSeedHex   string   `json:"server_seed_hex"`
	Weights         []uint64 `json:"weights"`
	BasisPoints     uint64   `json:"basis_points"`
	Vectors         []struct {
		Name           string   `json:"name"`
		Kind           string   `json:"kind"`
		Subject        string   `json:"subject"`
		Uint64Hex      []string `json:"uint64_hex"`
		Float64BitsHex []string `json:"float64_0_1_bits_hex"`
		WeightedIndex  int      `json:"weighted_index"`
		BasisPointsHit bool     `json:"basis_points_hit"`
	} `json:"vectors"`
}

func loadFixture(t *testing.T) regressionFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/pcg64_regression.json")
	if err != nil {
		t.Fatalf("read regression fixture: %v", err)
	}
	var f regressionFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse regression fixture: %v", err)
	}
	return f
}

func parseSeed(t *testing.T, hexSeed string) Seed {
	t.Helper()
	b, err := hex.DecodeString(hexSeed)
	if err != nil {
		t.Fatalf("seed hex: %v", err)
	}
	if len(b) != SeedLen {
		t.Fatalf("seed length = %d, want %d", len(b), SeedLen)
	}
	var s Seed
	copy(s[:], b)
	return s
}

func TestPCG64RegressionVector(t *testing.T) {
	f := loadFixture(t)
	if f.Schema != "thinhthan/rng-regression/1" {
		t.Fatalf("fixture schema = %q", f.Schema)
	}
	seed := parseSeed(t, f.ServerSeedHex)
	for _, v := range f.Vectors {
		ctx := Context{Kind: Kind(v.Kind), Subject: v.Subject}
		st := mustStream(t, f.ContentRevision, ctx, seed)

		// The stream must be exactly PCG-64 over the derived (state, seq):
		// cross-check the first draw against a raw math/rand/v2 PCG.
		state, seq := derive(f.ContentRevision, ctx, seed)
		raw := rand.New(rand.NewPCG(state, seq))
		for i, wantHex := range v.Uint64Hex {
			want, err := hex.DecodeString(wantHex)
			if err != nil || len(want) != 8 {
				t.Fatalf("%s: bad uint64_hex[%d] %q", v.Name, i, wantHex)
			}
			wantU := binary.BigEndian.Uint64(want)
			if got := st.Uint64(); got != wantU {
				t.Fatalf("%s: Uint64 draw %d = %#x, want %#x", v.Name, i, got, wantU)
			}
			if got := raw.Uint64(); got != wantU {
				t.Fatalf("%s: raw PCG-64 draw %d = %#x, want %#x — stream is not PCG-64 over the derived seed", v.Name, i, got, wantU)
			}
		}
		for i, wantHex := range v.Float64BitsHex {
			want, err := hex.DecodeString(wantHex)
			if err != nil || len(want) != 8 {
				t.Fatalf("%s: bad float64_0_1_bits_hex[%d] %q", v.Name, i, wantHex)
			}
			got, err := st.Float64(0.0, 1.0)
			if err != nil {
				t.Fatalf("%s: Float64(0,1): %v", v.Name, err)
			}
			if gotBits := math.Float64bits(got); gotBits != binary.BigEndian.Uint64(want) {
				t.Fatalf("%s: Float64(0,1) draw %d bits = %#x, want %#x", v.Name, i, gotBits, binary.BigEndian.Uint64(want))
			}
		}
		if got, err := st.WeightedIndex(f.Weights); err != nil {
			t.Fatalf("%s: WeightedIndex(%v): %v", v.Name, f.Weights, err)
		} else if got != v.WeightedIndex {
			t.Fatalf("%s: WeightedIndex(%v) = %d, want %d", v.Name, f.Weights, got, v.WeightedIndex)
		}
		if got, err := st.RollBasisPoints(f.BasisPoints); err != nil {
			t.Fatalf("%s: RollBasisPoints(%d): %v", v.Name, f.BasisPoints, err)
		} else if got != v.BasisPointsHit {
			t.Fatalf("%s: RollBasisPoints(%d) = %v, want %v", v.Name, f.BasisPoints, got, v.BasisPointsHit)
		}
	}
}

func TestDeterministicSameInputs(t *testing.T) {
	ctx := Context{Kind: "drop_table", Subject: "drop.rung_u_minh.elite"}
	a := mustStream(t, testRevision, ctx, testSeed)
	b := mustStream(t, testRevision, ctx, testSeed)
	// Same content revision + seed + context => identical stream.
	for i := 0; i < 64; i++ {
		if ga, gb := a.Uint64(), b.Uint64(); ga != gb {
			t.Fatalf("draw %d diverged: %#x != %#x", i, ga, gb)
		}
	}
	// Same ordered candidates => identical bounded selections.
	ca, cb := mustStream(t, testRevision, ctx, testSeed), mustStream(t, testRevision, ctx, testSeed)
	weights := []uint64{400, 150, 150, 300}
	for i := 0; i < 32; i++ {
		ia, err := ca.WeightedIndex(weights)
		if err != nil {
			t.Fatalf("WeightedIndex: %v", err)
		}
		ib, err := cb.WeightedIndex(weights)
		if err != nil {
			t.Fatalf("WeightedIndex: %v", err)
		}
		if ia != ib {
			t.Fatalf("selection %d diverged: %d != %d", i, ia, ib)
		}
	}
}

func TestIndependentContextStreams(t *testing.T) {
	base := Context{Kind: "spawn_pool", Subject: "zone.rung_u_minh.group.3"}
	other := []struct {
		name string
		rev  string
		ctx  Context
		seed Seed
	}{
		{"different kind", testRevision, Context{Kind: "drop_table", Subject: base.Subject}, testSeed},
		{"different subject", testRevision, Context{Kind: base.Kind, Subject: "zone.rung_u_minh.group.4"}, testSeed},
		{"different revision", "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210", base, testSeed},
		{"different seed", testRevision, base, Seed{9}},
	}
	for _, o := range other {
		a := mustStream(t, testRevision, base, testSeed)
		b := mustStream(t, o.rev, o.ctx, o.seed)
		same := true
		for i := 0; i < 8; i++ {
			if a.Uint64() != b.Uint64() {
				same = false
				break
			}
		}
		if same {
			t.Fatalf("%s: streams produced identical draws", o.name)
		}
	}
	// Interleaved draws from two live streams do not perturb either stream.
	a := mustStream(t, testRevision, base, testSeed)
	b := mustStream(t, testRevision, Context{Kind: "combat", Subject: "combat_event.tick.1.actor.1"}, testSeed)
	aSolo := mustStream(t, testRevision, base, testSeed)
	bSolo := mustStream(t, testRevision, Context{Kind: "combat", Subject: "combat_event.tick.1.actor.1"}, testSeed)
	for i := 0; i < 16; i++ {
		if ga, gs := a.Uint64(), aSolo.Uint64(); ga != gs {
			t.Fatalf("stream A draw %d perturbed by interleave: %#x != %#x", i, ga, gs)
		}
		if gb, gs := b.Uint64(), bSolo.Uint64(); gb != gs {
			t.Fatalf("stream B draw %d perturbed by interleave: %#x != %#x", i, gb, gs)
		}
	}
}

func TestInvalidContextInputs(t *testing.T) {
	good := Context{Kind: "enhancement", Subject: "item_instance.550e8400-e29b-41d4-a716-446655440000.plus.15"}
	for _, tc := range []struct {
		name string
		rev  string
		ctx  Context
		seed Seed
		want error
	}{
		{"empty revision", "", good, testSeed, id.ErrInvalidContentRevision},
		{"short revision", "abcd", good, testSeed, id.ErrInvalidContentRevision},
		{"uppercase revision", "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", good, testSeed, id.ErrInvalidContentRevision},
		{"empty kind", testRevision, Context{Kind: "", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"uppercase kind", testRevision, Context{Kind: "COMBAT", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"kind with space", testRevision, Context{Kind: "drop table", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"kind leading underscore", testRevision, Context{Kind: "_combat", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"kind trailing underscore", testRevision, Context{Kind: "combat_", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"kind empty word", testRevision, Context{Kind: "combat__roll", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"kind with dot", testRevision, Context{Kind: "drop.table", Subject: good.Subject}, testSeed, ErrInvalidKind},
		{"empty subject", testRevision, Context{Kind: good.Kind, Subject: ""}, testSeed, ErrInvalidSubject},
		{"subject leading separator", testRevision, Context{Kind: good.Kind, Subject: ".abc"}, testSeed, ErrInvalidSubject},
		{"subject trailing separator", testRevision, Context{Kind: good.Kind, Subject: "abc."}, testSeed, ErrInvalidSubject},
		{"subject empty segment", testRevision, Context{Kind: good.Kind, Subject: "a..b"}, testSeed, ErrInvalidSubject},
		{"subject uppercase", testRevision, Context{Kind: good.Kind, Subject: "UPPER"}, testSeed, ErrInvalidSubject},
		{"subject with space", testRevision, Context{Kind: good.Kind, Subject: "has space"}, testSeed, ErrInvalidSubject},
		{"zero seed", testRevision, good, Seed{}, ErrZeroSeed},
	} {
		_, err := New(tc.rev, tc.ctx, tc.seed)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: New error = %v, want %v", tc.name, err, tc.want)
		}
	}
	// Overlong kind / subject.
	overKind := Context{Kind: Kind(stringOf('a', KindMaxLen+1)), Subject: good.Subject}
	if _, err := New(testRevision, overKind, testSeed); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("overlong kind: New error = %v, want %v", err, ErrInvalidKind)
	}
	overSubject := Context{Kind: good.Kind, Subject: stringOf('a', SubjectMaxLen+1)}
	if _, err := New(testRevision, overSubject, testSeed); !errors.Is(err, ErrInvalidSubject) {
		t.Fatalf("overlong subject: New error = %v, want %v", err, ErrInvalidSubject)
	}
}

func stringOf(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func TestWeightedSelection(t *testing.T) {
	ctx := Context{Kind: "drop_table", Subject: "drop.rung_u_minh.normal"}
	// Result is bounded in [0, len(weights)) and obeys the cumulative-weight
	// mapping: re-derive the draw on a clone stream and predict the index.
	weights := []uint64{400, 150, 150, 300}
	st := mustStream(t, testRevision, ctx, testSeed)
	clone := mustStream(t, testRevision, ctx, testSeed)
	var total uint64
	for _, w := range weights {
		total += w
	}
	for i := 0; i < 64; i++ {
		got, err := st.WeightedIndex(weights)
		if err != nil {
			t.Fatalf("WeightedIndex: %v", err)
		}
		if got < 0 || got >= len(weights) {
			t.Fatalf("WeightedIndex = %d, out of [0,%d)", got, len(weights))
		}
		v, err := clone.Uint64N(total)
		if err != nil {
			t.Fatalf("clone Uint64N: %v", err)
		}
		want := -1
		var acc uint64
		for j, w := range weights {
			if acc <= v && v < acc+w {
				want = j
				break
			}
			acc += w
		}
		if got != want {
			t.Fatalf("selection %d: WeightedIndex = %d, cumulative mapping wants %d", i, got, want)
		}
	}
	// Degenerate weight sets still return a bounded index.
	for i, w := range [][]uint64{{1, 0, 0}, {0, 0, 5}, {0, 7, 0}} {
		s := mustStream(t, testRevision, ctx, testSeed)
		got, err := s.WeightedIndex(w)
		if err != nil {
			t.Fatalf("weights %v: %v", w, err)
		}
		if wantIdx := map[int]int{0: 0, 1: 2, 2: 1}[i]; got != wantIdx {
			t.Fatalf("weights %v: index = %d, want %d", w, got, wantIdx)
		}
	}
	// Invalid weight sets are rejected and consume no draw.
	for _, w := range [][]uint64{nil, {}, {0, 0, 0}, {math.MaxUint64, 1}, {math.MaxUint64, math.MaxUint64}} {
		s := mustStream(t, testRevision, ctx, testSeed)
		if _, err := s.WeightedIndex(w); !errors.Is(err, ErrInvalidWeights) {
			t.Fatalf("weights %v: error = %v, want %v", w, err, ErrInvalidWeights)
		}
	}
	// Ordering matters: candidates are positional — moving the single live
	// weight moves the selected index, on the same stream state.
	for _, tc := range []struct {
		weights []uint64
		want    int
	}{
		{[]uint64{1, 0, 0, 0}, 0},
		{[]uint64{0, 0, 0, 1}, 3},
	} {
		s := mustStream(t, testRevision, ctx, testSeed)
		got, err := s.WeightedIndex(tc.weights)
		if err != nil {
			t.Fatalf("weights %v: %v", tc.weights, err)
		}
		if got != tc.want {
			t.Fatalf("weights %v: index = %d, want %d", tc.weights, got, tc.want)
		}
	}
}

func TestHalfOpenRangeBoundaries(t *testing.T) {
	ctx := Context{Kind: "combat", Subject: "combat_event.tick.7.actor.3"}
	st := mustStream(t, testRevision, ctx, testSeed)
	// Every sample lands in [lo, hi): lower bound reachable, upper bound
	// never returned.
	for i := 0; i < 4096; i++ {
		v, err := st.Float64(0.0, 1.0)
		if err != nil {
			t.Fatalf("Float64(0,1): %v", err)
		}
		if v < 0.0 || v >= 1.0 {
			t.Fatalf("Float64(0,1) = %v, want [0,1)", v)
		}
	}
	for i := 0; i < 1024; i++ {
		v, err := st.Float64(-2.5, 7.5)
		if err != nil {
			t.Fatalf("Float64(-2.5,7.5): %v", err)
		}
		if v < -2.5 || v >= 7.5 {
			t.Fatalf("Float64(-2.5,7.5) = %v, want [-2.5,7.5)", v)
		}
	}
	// Invalid ranges are rejected: empty, reversed, non-finite, infinite span.
	for _, r := range [][2]float64{
		{1.0, 1.0}, {2.0, 1.0},
		{math.NaN(), 1.0}, {0.0, math.NaN()},
		{0.0, math.Inf(1)}, {math.Inf(-1), 0.0},
		{-math.MaxFloat64, math.MaxFloat64}, // span overflows to +Inf
	} {
		s := mustStream(t, testRevision, ctx, testSeed)
		if _, err := s.Float64(r[0], r[1]); !errors.Is(err, ErrInvalidRange) {
			t.Fatalf("Float64(%v,%v): error = %v, want %v", r[0], r[1], err, ErrInvalidRange)
		}
	}
	// Uint64N: n == 0 rejected, every draw < n, Uint64N(1) always 0.
	for i := 0; i < 256; i++ {
		v, err := st.Uint64N(37)
		if err != nil {
			t.Fatalf("Uint64N(37): %v", err)
		}
		if v >= 37 {
			t.Fatalf("Uint64N(37) = %d, want < 37", v)
		}
		if v, err := st.Uint64N(1); err != nil || v != 0 {
			t.Fatalf("Uint64N(1) = %d, %v, want 0, nil", v, err)
		}
	}
	if _, err := st.Uint64N(0); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("Uint64N(0): error = %v, want %v", err, ErrInvalidRange)
	}
}

func TestRollBasisPoints(t *testing.T) {
	ctx := Context{Kind: "enhancement", Subject: "item_instance.0192a4bc.plus.15"}
	// bp == 0 never hits; bp == BasisPointsTotal always hits; both consume a
	// draw so the stream position is identical either way.
	zero := mustStream(t, testRevision, ctx, testSeed)
	full := mustStream(t, testRevision, ctx, testSeed)
	for i := 0; i < 64; i++ {
		if hit, err := zero.RollBasisPoints(0); err != nil || hit {
			t.Fatalf("RollBasisPoints(0) = %v, %v; want false, nil", hit, err)
		}
		if hit, err := full.RollBasisPoints(BasisPointsTotal); err != nil || !hit {
			t.Fatalf("RollBasisPoints(10000) = %v, %v; want true, nil", hit, err)
		}
	}
	if zu, fu := zero.Uint64(), full.Uint64(); zu != fu {
		t.Fatalf("bp extremes diverged the stream: %#x != %#x", zu, fu)
	}
	// Out-of-range bp rejects without consuming.
	if _, err := zero.RollBasisPoints(BasisPointsTotal + 1); !errors.Is(err, ErrInvalidBasisPoints) {
		t.Fatalf("RollBasisPoints(10001): error = %v, want %v", err, ErrInvalidBasisPoints)
	}
	// Same inputs => same hit sequence.
	a := mustStream(t, testRevision, ctx, testSeed)
	b := mustStream(t, testRevision, ctx, testSeed)
	for i := 0; i < 64; i++ {
		ha, err := a.RollBasisPoints(2500)
		if err != nil {
			t.Fatalf("RollBasisPoints: %v", err)
		}
		hb, err := b.RollBasisPoints(2500)
		if err != nil {
			t.Fatalf("RollBasisPoints: %v", err)
		}
		if ha != hb {
			t.Fatalf("bp roll %d diverged: %v != %v", i, ha, hb)
		}
	}
}

func TestNewSeed(t *testing.T) {
	s1, err := NewSeed()
	if err != nil {
		t.Fatalf("NewSeed: %v", err)
	}
	s2, err := NewSeed()
	if err != nil {
		t.Fatalf("NewSeed: %v", err)
	}
	if s1.IsZero() || s1 == s2 {
		t.Fatalf("NewSeed produced zero/identical seeds")
	}
}
