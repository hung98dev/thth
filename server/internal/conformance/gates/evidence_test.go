package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceTreeHashExclusions(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeRepoFile(t, dir, "x.txt", "one")
	writeRepoFile(t, dir, "docs/10_implementation/evidence/IMP-900/manifest.json", "{}")
	writeRepoFile(t, dir, "docs/10_implementation/task_queue.md", "q1")
	writeRepoFile(t, dir, "docs/10_implementation/known_blockers.md", "b1")
	commitAll(t, dir)
	h1, err := SourceTreeHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, dir, "docs/10_implementation/evidence/IMP-900/manifest.json", `{"changed":true}`)
	writeRepoFile(t, dir, "docs/10_implementation/task_queue.md", "q2")
	writeRepoFile(t, dir, "docs/10_implementation/known_blockers.md", "b2")
	commitAll(t, dir)
	h2, err := SourceTreeHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatal("hash changed by excluded control/evidence paths")
	}
	writeRepoFile(t, dir, "x.txt", "two")
	commitAll(t, dir)
	h3, _ := SourceTreeHash(dir)
	if h3 == h2 {
		t.Fatal("hash did not change on a tracked source change")
	}
}

func TestSourceTreeHashIdenticalAcrossOs(t *testing.T) {
	// Two clones with the same content must produce the identical hash —
	// the manifest hashes identically on linux and windows CI jobs.
	a, b := t.TempDir(), t.TempDir()
	for _, dir := range []string{a, b} {
		initRepo(t, dir)
		writeRepoFile(t, dir, "same/f.txt", "bytes")
		writeRepoFile(t, dir, "docs/10_implementation/evidence/IMP-900/m.json", "{}")
		commitAll(t, dir)
	}
	ha, _ := SourceTreeHash(a)
	hb, _ := SourceTreeHash(b)
	if ha != hb {
		t.Fatalf("identical content hashed differently: %s vs %s", ha, hb)
	}
}

func validManifest(t *testing.T, dir string) string {
	t.Helper()
	reports := map[string]VerifyReport{
		"linux":   {Schema: "verify-report-v1", Result: "PASSED", OS: "linux", Commands: []string{"go test ./..."}, Gates: []Gate{}},
		"windows": {Schema: "verify-report-v1", Result: "PASSED", OS: "windows", Commands: []string{"go test ./..."}, Gates: []Gate{}},
	}
	m, err := MergeReports("IMP-900", strings.Repeat("ab", 32), "12345", 1, reports)
	if err != nil {
		t.Fatal(err)
	}
	m.TestSummary.Total = 0
	p := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestManifestSchemaV2(t *testing.T) {
	dir := t.TempDir()
	p := validManifest(t, dir)
	if probs := ValidateManifest(p, "IMP-900"); len(probs) != 0 {
		t.Fatalf("valid manifest rejected: %v", probs)
	}
	if probs := ValidateManifest(p, "IMP-999"); len(probs) == 0 {
		t.Fatal("wrong task_id accepted")
	}
	if probs := ValidateManifest(filepath.Join(dir, "absent.json"), "IMP-900"); len(probs) == 0 {
		t.Fatal("missing manifest accepted")
	}
}

func TestEvidenceManifestJobMergesBothReports(t *testing.T) {
	dir := t.TempDir()
	p := validManifest(t, dir)
	var m EvidenceManifest
	data, _ := os.ReadFile(p)
	if err := decodeManifest(data, &m); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, j := range m.Jobs {
		seen[j.OS] = true
	}
	if !seen["linux"] || !seen["windows"] {
		t.Fatalf("manifest must merge both OS reports, jobs=%v", m.Jobs)
	}
	if len(m.Commands) == 0 || m.CIRunID != "12345" || m.RunAttempt != 1 {
		t.Fatalf("merged fields missing: %+v", m)
	}
}

func TestRunIdAttemptApiCheck(t *testing.T) {
	// The API check cannot run offline — it must fail closed on missing
	// repo/run_id/token rather than pass silently.
	probs := CheckRunIdentity(t.TempDir(), "", "1", 1, "")
	if len(probs) == 0 {
		t.Fatal("empty inputs must produce problems")
	}
	probs = CheckRunIdentity(t.TempDir(), "o/r", "", 1, "tok")
	if len(probs) == 0 {
		t.Fatal("missing run_id must produce problems")
	}
}

func TestLandedViaMergeSquashAncestor(t *testing.T) {
	// Squash merges land the PR's content as a NEW commit on main: the PR
	// head sha is never an ancestor of HEAD, the merge/squash commit is.
	// The gate must accept the merge commit and still fail closed when no
	// candidate is an ancestor.
	dir := t.TempDir()
	initRepo(t, dir)
	writeRepoFile(t, dir, "x.txt", "one")
	commitAll(t, dir)
	// PR head content squashed into a separate main commit (like GitHub's
	// squash merge: new sha, same tree delta).
	writeRepoFile(t, dir, "x.txt", "two")
	commitAll(t, dir)
	squashed := strings.TrimSpace(gitT(t, dir, "rev-parse", "HEAD"))
	// A commit that exists but is NOT an ancestor (orphan line).
	gitT(t, dir, "checkout", "-q", "--orphan", "side")
	gitT(t, dir, "rm", "-rf", ".")
	writeRepoFile(t, dir, "side.txt", "s")
	commitAll(t, dir)
	orphan := strings.TrimSpace(gitT(t, dir, "rev-parse", "HEAD"))
	gitT(t, dir, "checkout", "-q", "main")
	if !landedViaMerge(dir, []string{squashed}) {
		t.Fatal("merge-commit ancestor must satisfy the squash fallback")
	}
	if landedViaMerge(dir, []string{orphan}) {
		t.Fatal("non-ancestor candidate must not pass")
	}
	if landedViaMerge(dir, nil) {
		t.Fatal("no candidates must fail closed")
	}
}

func TestTwoPhaseStatusPrOwnRunEvidence(t *testing.T) {
	// A -done PR's own verify run produces the manifest the merged head
	// must carry: merge → write under evidence/<task>/ → validate.
	dir := t.TempDir()
	p := validManifest(t, dir)
	target := filepath.Join(dir, "docs", "10_implementation", "evidence", "IMP-900", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if probs := ValidateManifest(target, "IMP-900"); len(probs) != 0 {
		t.Fatalf("own-run manifest must validate for the -done head: %v", probs)
	}
}
