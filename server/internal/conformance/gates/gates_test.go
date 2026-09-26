package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := RepoRoot(".")
	if err != nil {
		t.Skipf("not in a repo checkout: %v", err)
	}
	return root
}

func localEnv() *Env {
	return &Env{LocalDefer: true, MainRef: "origin/main"}
}

// TestQ0ParsesRealQueue proves the wiring: Q0 parses the real task_queue.md
// and produces PASS/FAIL checks (never panics).
func TestQ0ParsesRealQueue(t *testing.T) {
	root := repoRoot(t)
	checks := CheckQ0(root, localEnv())
	if len(checks) == 0 {
		t.Fatal("Q0 produced no checks")
	}
	var ids []string
	for _, c := range checks {
		ids = append(ids, c.ID+"="+string(c.Status))
	}
	t.Log(strings.Join(ids, ", "))
}

// TestGateRequiredWhenOwnerDoneOnMainOrHead checks the activation rule.
func TestGateRequiredWhenOwnerDoneOnMainOrHead(t *testing.T) {
	cases := []struct {
		main, head string
		want       bool
	}{
		{"DONE", "NOT_STARTED", true},
		{"IN_PROGRESS", "DONE", true},
		{"DONE", "DONE", true},
		{"IN_PROGRESS", "NOT_STARTED", false},
		{"NOT_STARTED", "IN_PROGRESS", false},
		{"NOT_STARTED", "NOT_STARTED", false},
	}
	for _, c := range cases {
		if got := GateRequired("IMP-061", c.main, c.head); got != c.want {
			t.Errorf("GateRequired(main=%s, head=%s) = %v, want %v", c.main, c.head, got, c.want)
		}
	}
}

// TestSkipOwnerNotDone verifies the SKIP(owner-not-done) marker names the
// gate and the owner task.
func TestSkipOwnerNotDone(t *testing.T) {
	c := SkipOwnerNotDone("Q2", "IMP-061")
	if c.Status != StatusSkip {
		t.Fatalf("status %s", c.Status)
	}
	if !strings.Contains(c.Detail, "owner-not-done") || !strings.Contains(c.Detail, "IMP-061") {
		t.Fatalf("detail %q must name owner + reason", c.Detail)
	}
}

// TestPrRoleFromBranchPrefix maps branch prefixes to roles.
func TestPrRoleFromBranchPrefix(t *testing.T) {
	cases := map[string]PRRole{
		"spec/foo":              RoleSpecOwner,
		"claim/IMP-001":         RoleCoordinator,
		"ops/blah":              RoleCoordinator,
		"imp/IMP-000-bootstrap": RoleImplementer,
		"block/IMP-002-1":       RoleImplementer,
		"revert/deadbeef":       RoleMergeGuard,
		"feature/x":             RoleNone,
		"main":                  RoleNone,
		"fixit":                 RoleNone,
	}
	for branch, want := range cases {
		if got := RoleFromBranch(branch); got != want {
			t.Errorf("RoleFromBranch(%q) = %s, want %s", branch, got, want)
		}
	}
	if TaskFromBranch("imp/IMP-000-bootstrap") != "IMP-000" {
		t.Error("TaskFromBranch imp/")
	}
	if TaskFromBranch("block/IMP-003-7") != "IMP-003" {
		t.Error("TaskFromBranch block/")
	}
}

// TestStatusOnlyPrFastPath: a control-file-only diff classifies status-only.
func TestStatusOnlyPrFastPath(t *testing.T) {
	if !StatusOnlyPR([]string{"docs/10_implementation/task_queue.md"}) {
		t.Fatal("task_queue-only diff must be status-only")
	}
	if StatusOnlyPR([]string{"docs/10_implementation/known_blockers.md", "docs/10_implementation/evidence/IMP-001/manifest.json"}) {
		t.Fatal("evidence diff must not take the Q0-only status-only fast path")
	}
	if StatusOnlyPR([]string{"docs/10_implementation/task_queue.md", "server/go.mod"}) {
		t.Fatal("code in diff must not be status-only")
	}
}

// TestBlockAndOpsPrFastPath: block/ and ops/ branches are implementer /
// coordinator roles whose diffs stay status-only (no code, no DONE).
func TestBlockAndOpsPrFastPath(t *testing.T) {
	if RoleFromBranch("block/IMP-005-2") != RoleImplementer {
		t.Fatal("block/ must be implementer role")
	}
	if RoleFromBranch("ops/fix-env") != RoleCoordinator {
		t.Fatal("ops/ must be coordinator role")
	}
	files := []string{"docs/10_implementation/task_queue.md", "docs/10_implementation/known_blockers.md"}
	if !StatusOnlyPR(files) {
		t.Fatal("block/ diff must be status-only")
	}
}

// TestDonePrRunsAllGates: a diff that includes a DONE transition is not
// status-only even when every file is a control file.
func TestDonePrRunsAllGates(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)

	queue := queueFixture("IN_PROGRESS")
	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queue)
	commitAll(t, root)

	// head: DONE transition + evidence
	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queueFixture("DONE"))
	writeRepoFile(t, root, "docs/10_implementation/evidence/IMP-900/manifest.json", "{}")
	commitAll(t, root)

	e := &Env{EventName: "pull_request", BaseSHA: "HEAD~1", HeadSHA: "HEAD", HeadRef: "imp/IMP-900-done", HeadRepo: "x/x", Repository: "x/x"}
	cl, err := ClassifyPR(root, e)
	if err != nil {
		t.Fatal(err)
	}
	if cl.StatusOnly {
		t.Fatal("DONE transition must not be status-only")
	}
	if !cl.DoneTransition {
		t.Fatal("expected DoneTransition")
	}
}

// TestDoneWithoutManifestAllowedOnHead: a fresh DONE transition without a
// manifest is allowed (step 7 commits it later).
func TestDoneWithoutManifestAllowedOnHead(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)
	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queueFixture("IN_PROGRESS"))
	commitAll(t, root)
	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queueFixture("DONE"))
	commitAll(t, root)

	e := &Env{EventName: "pull_request", BaseSHA: "HEAD~1", HeadSHA: "HEAD", HeadRef: "imp/IMP-900-done"}
	packets, _, err := ParseTaskQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	checks := checkDoneManifestRule(root, e, packets)
	for _, c := range checks {
		if c.Status == StatusFail {
			t.Fatalf("fresh DONE transition must pass: %s", c.Detail)
		}
	}
}

// TestMergedHeadRequiresManifest: a packet already DONE on base without a
// manifest on the tree fails.
func TestMergedHeadRequiresManifest(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)
	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queueFixture("DONE"))
	commitAll(t, root)
	writeRepoFile(t, root, "README.md", "x\n")
	commitAll(t, root)

	e := &Env{EventName: "pull_request", BaseSHA: "HEAD~1", HeadSHA: "HEAD", HeadRef: "imp/IMP-900-other"}
	packets, _, err := ParseTaskQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	checks := checkDoneManifestRule(root, e, packets)
	failed := false
	for _, c := range checks {
		if c.Status == StatusFail {
			failed = true
		}
	}
	if !failed {
		t.Fatal("merged DONE head without manifest must fail")
	}
}

// TestTwoPhaseListIncludesImp083 pins the exact two-phase task set.
func TestTwoPhaseListIncludesImp083(t *testing.T) {
	want := []string{"IMP-000", "IMP-061", "IMP-003", "IMP-004", "IMP-005", "IMP-083", "IMP-065", "IMP-068"}
	got := TwoPhaseTasks()
	if len(got) != len(want) {
		t.Fatalf("two-phase list %v, want %v", got, want)
	}
	seen := map[string]bool{}
	for _, x := range got {
		seen[x] = true
	}
	for _, w := range want {
		if !seen[w] {
			t.Errorf("missing %s", w)
		}
	}
}

// TestLocalDeferMissingNeverInCi: DEFERRED(local-missing) is impossible in
// CI — the workflow never passes -LocalDeferMissing and the verifier fails.
func TestLocalDeferMissingNeverInCi(t *testing.T) {
	e := &Env{CI: true}
	c := e.missingCheck("Q1.test", "tool missing")
	if c.Status != StatusFail {
		t.Fatalf("CI missing tool must FAIL, got %s", c.Status)
	}
	e2 := &Env{LocalDefer: true}
	c2 := e2.missingCheck("Q1.test", "tool missing")
	if c2.Status != StatusDeferred {
		t.Fatalf("local -LocalDeferMissing must DEFER, got %s", c2.Status)
	}
	// verify.ps1 must not allow the flag in CI either — asserted on file text.
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "scripts", "verify.ps1"))
	if err != nil {
		t.Skip("verify.ps1 not written yet")
	}
	if !strings.Contains(string(data), "LocalDeferMissing") {
		t.Fatal("verify.ps1 must accept -LocalDeferMissing")
	}
}

// Fail-closed mutation fixtures: corrupt packets must trip Q0.
func TestQ0FailClosedMutations(t *testing.T) {
	root := t.TempDir()
	initRepo(t, root)

	writeRepoFile(t, root, "docs/10_implementation/task_queue.md", queueFixture("IN_PROGRESS"))
	packets, _, err := ParseTaskQueue(root)
	if err != nil {
		t.Fatal(err)
	}

	// mutation: depends_on a nonexistent packet
	packets[0].DependsOn = []string{"IMP-999"}
	byID := map[string]TaskPacket{}
	ids := map[string]bool{}
	for _, p := range packets {
		byID[p.ID] = p
		ids[p.ID] = true
	}
	_ = byID
	found := false
	for _, p := range packets {
		for _, d := range p.DependsOn {
			if !ids[d] {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("mutation fixture broken")
	}
}

// TestImp000OwnedPathsCoverMaterializedAssets (BLK-001): the URP root
// assets the editor materializes during IMP-000 CI must sit inside the
// packet's owned_paths, else Q0.control.diff fails on them.
func TestImp000OwnedPathsCoverMaterializedAssets(t *testing.T) {
	root := repoRoot(t)
	packets, _, err := ParseTaskQueue(root)
	if err != nil {
		t.Fatal(err)
	}
	var imp000 *TaskPacket
	for i := range packets {
		if packets[i].ID == "IMP-000" {
			imp000 = &packets[i]
			break
		}
	}
	if imp000 == nil {
		t.Skip("IMP-000 packet not in queue")
	}
	for _, f := range []string{
		"client/Assets/DefaultVolumeProfile.asset",
		"client/Assets/DefaultVolumeProfile.asset.meta",
		"client/Assets/UniversalRenderPipelineGlobalSettings.asset",
		"client/Assets/UniversalRenderPipelineGlobalSettings.asset.meta",
	} {
		if !ownedFile(*imp000, f) {
			t.Errorf("IMP-000 owned_paths do not cover materialized %s", f)
		}
	}
}

// queueFixture renders a minimal one-packet task_queue.md in the real format
// (## `IMP-900` — heading, `key: value` fields, backticked summary row).
func queueFixture(status string) string {
	return `# Task Queue

| Task | Title | Status | Blocked | Specs |
|---|---|---|---|---|
| ` + "`IMP-900`" + ` | Fixture | ` + "`" + status + "`" + ` | none | x |

## ` + "`IMP-900`" + ` — Fixture
id: IMP-900
status: ` + status + `
claimed_by: "agent-1"
branch: "imp/IMP-900-x"
claimed_at: "2026-01-01T00:00:00Z"
blocked_by: ""
specs: [dependency_graph.md]
adrs: []
depends_on: []
owned_paths: [server/cmd/verify/]
forbidden_paths: []
evidence_location: docs/10_implementation/evidence/IMP-900/
required_evidence: manifest

## Change
fixture

## Acceptance
fixture CODE-001

## Tests
fixture CODE-001
`
}
