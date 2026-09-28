package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"thinhthan/internal/conformance/gates"
)

func gate(id, title string) gates.Gate {
	return gates.Gate{ID: id, Title: title, Checks: []gates.Check{gates.Pass(id+".x", "")}}
}

// ADR-0077: the two-phase report lists the gates in exactly the order of a
// single full run — Q0..Q3 Go, the Unity gates, then Q4..Q6.
func TestSpliceGatesCanonicalOrder(t *testing.T) {
	before := []gates.Gate{gate("Q0", "a"), gate("Q3", "Test suites (Go)")}
	unity := []gates.Gate{gate("Q3", "Test suites (Unity EditMode)"), gate("Q3", "Test suites (Unity PlayMode)")}
	after := []gates.Gate{gate("Q4", "b"), gate("Q6", "c")}
	got := spliceGates(before, unity, after)
	want := []string{"a", "Test suites (Go)", "Test suites (Unity EditMode)", "Test suites (Unity PlayMode)", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %d gates, want %d", len(got), len(want))
	}
	for i, g := range got {
		if g.Title != want[i] {
			t.Fatalf("gate %d = %q, want %q", i, g.Title, want[i])
		}
	}
	if len(before) != 2 || len(unity) != 2 || len(after) != 2 {
		t.Fatal("spliceGates must not modify its inputs")
	}
}

func writePre(t *testing.T, p preUnityReport) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pre.json")
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// ADR-0077: -phase unity only reuses a pre-unity file produced for the same
// checkout; anything else makes it fall back to the full gate set.
func TestReadPreReportValidation(t *testing.T) {
	valid := preUnityReport{
		Schema:      preUnitySchema,
		Head:        "abc123",
		Commands:    []string{"go -C server test ./..."},
		BeforeUnity: []gates.Gate{gate("Q0", "a")},
		AfterUnity:  []gates.Gate{gate("Q6", "c")},
	}
	got, err := readPreReport(writePre(t, valid), "abc123")
	if err != nil {
		t.Fatalf("valid pre-unity report rejected: %v", err)
	}
	if !reflect.DeepEqual(*got, valid) {
		t.Fatalf("round trip changed the report: %+v", *got)
	}

	cases := map[string]func(p *preUnityReport){
		"foreign head":   func(p *preUnityReport) { p.Head = "def456" },
		"wrong schema":   func(p *preUnityReport) { p.Schema = "verify-report-v1" },
		"missing before": func(p *preUnityReport) { p.BeforeUnity = nil },
		"missing after":  func(p *preUnityReport) { p.AfterUnity = nil },
	}
	for name, mutate := range cases {
		p := valid
		mutate(&p)
		if _, err := readPreReport(writePre(t, p), "abc123"); err == nil {
			t.Errorf("%s: pre-unity report accepted", name)
		}
	}
	if _, err := readPreReport(filepath.Join(t.TempDir(), "absent.json"), "abc123"); err == nil {
		t.Error("absent pre-unity report accepted")
	}
	if _, err := readPreReport("", "abc123"); err == nil {
		t.Error("empty -pre-report path accepted")
	}
	noHead := valid
	noHead.Head = ""
	if _, err := readPreReport(writePre(t, noHead), ""); err == nil {
		t.Error("pre-unity report accepted without a known HEAD")
	}
}
