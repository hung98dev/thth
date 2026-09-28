package taskgraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"thinhthan/internal/conformance/gates"
)

// ---------- fixtures ------------------------------------------------------

func bticks(xs []string) string {
	var out []string
	for _, x := range xs {
		out = append(out, "`"+x+"`")
	}
	return strings.Join(out, ", ")
}

// mkPacket renders one packet block for fixture queues.
func mkPacket(id, status, claimedBy, branch, blockedBy string, deps, owned, forbidden []string, tests string) string {
	claimedAt := ""
	if status != "NOT_STARTED" {
		claimedAt = "2026-01-01T00:00:00Z"
	}
	return fmt.Sprintf(`## `+"`"+`%s`+"`"+` — fixture packet
id: %s
status: %s
claimed_by: "%s"
branch: "%s"
claimed_at: "%s"
blocked_by: "%s"

specs: [`+"`"+`fixture.md`+"`"+`]
adrs: [`+"`"+`0001-fixture.md`+"`"+`]
depends_on: [%s]
owned_paths: [%s]
forbidden_paths: [%s]
contract_inputs: []
contract_outputs: []
consumers_checked: []

## Change
c

## Acceptance
- a

## Tests
%s
`, id, id, status, claimedBy, branch, claimedAt, blockedBy,
		strings.Join(deps, ", "), bticks(owned), bticks(forbidden), tests)
}

func mkBlockers(open, resolved string) string {
	return "# Known Blockers\n\n## Open Blockers\n\n" + open + "\n\n## Resolved Blockers\n\n" + resolved
}

func blkEntry(id, blocks string) string {
	return fmt.Sprintf("### `%s` — fixture blocker\nopened_by: x   opened_at: 2026-01-01\nblocks: %s\n", id, blocks)
}

func parseQueue(t *testing.T, text string) []gates.TaskPacket {
	t.Helper()
	ps, _, err := gates.ParseTaskQueueText(text)
	if err != nil {
		t.Fatalf("fixture queue does not parse: %v", err)
	}
	return ps
}

func queueOf(blocks ...string) string {
	return "# Task Queue\n\n" + strings.Join(blocks, "\n")
}

func byID(ps []gates.TaskPacket) map[string]gates.TaskPacket {
	m := map[string]gates.TaskPacket{}
	for _, p := range ps {
		m[p.ID] = p
	}
	return m
}

// ---------- real-tree assertions ------------------------------------------

func realPackets(t *testing.T) []gates.TaskPacket {
	t.Helper()
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not inside a git worktree: %v", err)
	}
	ps, _, err := gates.ParseTaskQueue(root)
	if err != nil {
		t.Fatalf("real queue: %v", err)
	}
	return ps
}

func realBlockers(t *testing.T) BlockerDoc {
	t.Helper()
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not inside a git worktree: %v", err)
	}
	return LoadBlockers(root)
}

func realExists(t *testing.T) func(string) bool {
	t.Helper()
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not inside a git worktree: %v", err)
	}
	return fileExistsFunc(root)
}

// ---------- tests ----------------------------------------------------------

func TestDagAcyclic(t *testing.T) {
	if got := DagAcyclicProblems(realPackets(t)); len(got) != 0 {
		t.Fatalf("real task DAG has cycles: %v", got)
	}
	bad := parseQueue(t, queueOf(
		mkPacket("IMP-901", "NOT_STARTED", "", "", "", []string{"IMP-901"}, nil, nil, "- t"),
	))
	if got := DagAcyclicProblems(bad); len(got) == 0 {
		t.Fatal("self-dependency was not reported")
	}
	cycle := parseQueue(t, queueOf(
		mkPacket("IMP-901", "NOT_STARTED", "", "", "", []string{"IMP-902"}, nil, nil, "- t"),
		mkPacket("IMP-902", "NOT_STARTED", "", "", "", []string{"IMP-901"}, nil, nil, "- t"),
	))
	got := DagAcyclicProblems(cycle)
	if len(got) == 0 || !strings.Contains(strings.Join(got, " "), "cycle") {
		t.Fatalf("2-cycle not reported: %v", got)
	}
}

func TestDanglingRefs(t *testing.T) {
	if got := DanglingRefProblems(realPackets(t), realExists(t), realBlockers(t)); len(got) != 0 {
		t.Fatalf("real queue has dangling refs: %v", got)
	}
	exists := func(rel string) bool {
		return rel == "docs/10_implementation/fixture.md" || rel == "docs/11_decisions/0001-fixture.md"
	}
	clean := queueOf(mkPacket("IMP-901", "NOT_STARTED", "", "", "", nil, nil, nil, "- t"))
	if got := DanglingRefProblems(parseQueue(t, clean), exists, BlockerDoc{Open: map[string]BlockerEntry{}}); len(got) != 0 {
		t.Fatalf("valid fixture flagged: %v", got)
	}
	bad := parseQueue(t, queueOf(
		mkPacket("IMP-901", "NOT_STARTED", "", "", "", []string{"IMP-999"}, nil, nil, "- t"),
	))
	if got := DanglingRefProblems(bad, exists, BlockerDoc{}); len(got) == 0 {
		t.Fatal("dangling depends_on not reported")
	}
	missingSpec := strings.Replace(clean, "fixture.md", "missing.md", 1)
	if got := DanglingRefProblems(parseQueue(t, missingSpec), exists, BlockerDoc{}); len(got) == 0 {
		t.Fatal("missing spec file not reported")
	}
	missingADR := strings.Replace(clean, "0001-fixture.md", "9999-none.md", 1)
	if got := DanglingRefProblems(parseQueue(t, missingADR), exists, BlockerDoc{}); len(got) == 0 {
		t.Fatal("missing ADR not reported")
	}
	// BLOCKED with blocked_by naming no open entry.
	blocked := queueOf(mkPacket("IMP-901", "BLOCKED", "a", "imp/IMP-901-x", "BLK-404", nil, nil, nil, "- t"))
	if got := DanglingRefProblems(parseQueue(t, blocked), exists, BlockerDoc{Open: map[string]BlockerEntry{}}); len(got) == 0 {
		t.Fatal("blocked_by to nonexistent entry not reported")
	}
	okBlk := BlockerDoc{Open: map[string]BlockerEntry{"BLK-404": {ID: "BLK-404"}}}
	if got := DanglingRefProblems(parseQueue(t, blocked), exists, okBlk); len(got) != 0 {
		t.Fatalf("open blocked_by flagged: %v", got)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOwnedForbiddenOverlap(t *testing.T) {
	if got := OwnedForbiddenProblems(realPackets(t)); len(got) != 0 {
		t.Fatalf("real queue: %v", got)
	}
	// own owned ∩ forbidden
	bad := parseQueue(t, queueOf(
		mkPacket("IMP-901", "NOT_STARTED", "", "", "", nil,
			[]string{"server/internal/x/"}, []string{"server/internal/x/"}, "- t"),
	))
	if got := OwnedForbiddenProblems(bad); len(got) == 0 {
		t.Fatal("own owned∩forbidden overlap not reported")
	}
	// two packets sharing an owned path without ordering
	bad2 := parseQueue(t, queueOf(
		mkPacket("IMP-901", "NOT_STARTED", "", "", "", nil, []string{"server/internal/x/"}, nil, "- t"),
		mkPacket("IMP-902", "NOT_STARTED", "", "", "", nil, []string{"server/internal/x/"}, nil, "- t"),
	))
	if got := OwnedForbiddenProblems(bad2); len(got) == 0 {
		t.Fatal("unordered shared ownership not reported")
	}
	// shared path is legal when depends_on orders them
	ok := parseQueue(t, queueOf(
		mkPacket("IMP-901", "DONE", "a", "imp/IMP-901-x", "", nil, []string{"server/internal/x/"}, nil, "- t"),
		mkPacket("IMP-902", "NOT_STARTED", "", "", "", []string{"IMP-901"}, []string{"server/internal/x/sub/"}, nil, "- t"),
	))
	if got := OwnedForbiddenProblems(ok); len(got) != 0 {
		t.Fatalf("ordered overlap flagged: %v", got)
	}
}

func TestTestPathsOwned(t *testing.T) {
	known := realBlockers(t).AllIDs()
	if got := TestPathProblems(realPackets(t), known); len(got) != 0 {
		t.Fatalf("real queue: %v", got)
	}
	tests := "- `server/internal/mine/x_test.go`: TestA.\n- `server/internal/foreign/y_test.go` (BLK-100): TestB."
	knownIDs := map[string]bool{"BLK-100": true}
	ok := parseQueue(t, queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil,
			[]string{"server/internal/mine/"}, nil, tests),
	))
	if got := TestPathProblems(ok, knownIDs); len(got) != 0 {
		t.Fatalf("owned+BLK-marked paths flagged: %v", got)
	}
	bad := parseQueue(t, queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil,
			[]string{"server/internal/mine/"}, nil,
			"- `server/internal/foreign/y_test.go`: TestB."),
	))
	if got := TestPathProblems(bad, knownIDs); len(got) == 0 {
		t.Fatal("unowned unmarked test path not reported")
	}
	// unknown marker id fails too
	badMarker := parseQueue(t, queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil,
			[]string{"server/internal/mine/"}, nil,
			"- `server/internal/foreign/y_test.go` (BLK-999): TestB."),
	))
	if got := TestPathProblems(badMarker, knownIDs); len(got) == 0 {
		t.Fatal("marker naming a nonexistent blocker not reported")
	}
}

func TestRequirementIdCoverage(t *testing.T) {
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not inside a git worktree: %v", err)
	}
	if got := RequirementCoverageProblems(root, realPackets(t)); len(got) != 0 {
		t.Fatalf("real tree: %v", got)
	}
	// mutation: a spec dir with a Requirement IDs table; id uncovered.
	tmp := t.TempDir()
	writeFile(t, tmp, "docs/03_systems/items.md",
		"## Requirement IDs\n\n| `FOO-001` | thing |\n")
	if got := RequirementCoverageProblems(tmp, nil); len(got) == 0 {
		t.Fatal("uncovered requirement id not reported")
	}
	// acceptance names FOO-001 but tests must name it too (same packet).
	withCov := strings.Replace(queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil, nil, nil,
			"- `server/internal/x/x_test.go`: TestFoo (FOO-001)."),
	), "- a", "- FOO-001 covered", 1)
	if got := RequirementCoverageProblems(tmp, parseQueue(t, withCov)); len(got) != 0 {
		t.Fatalf("covered id flagged: %v", got)
	}
	// split across two packets must NOT count.
	split := queueOf(
		strings.Replace(mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil, nil, nil, "- t"), "- a", "- FOO-001 in acceptance", 1) +
			mkPacket("IMP-902", "IN_PROGRESS", "a", "imp/IMP-902-x", "", nil, nil, nil, "- `server/internal/y/y_test.go`: TestFoo (FOO-001)."),
	)
	if got := RequirementCoverageProblems(tmp, parseQueue(t, split)); len(got) == 0 {
		t.Fatal("id split across packets wrongly counted as covered")
	}
}

func TestControlFileDiffRules(t *testing.T) {
	baseQ := queueOf(mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil,
		[]string{"server/internal/mine/"}, nil, "- `server/internal/mine/x_test.go`: TestA."))
	headQ := baseQ

	// unknown prefix fails
	d := PRDiff{Role: gates.RoleNone, Branch: "feature/x", Files: []string{"server/internal/mine/x.go"}, BaseQueue: baseQ, HeadQueue: headQ}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("unprefixed branch not rejected")
	}
	// implementer touching a protected doc fails
	d = PRDiff{Role: gates.RoleImplementer, Branch: "imp/IMP-901-x", TaskID: "IMP-901",
		Files: []string{"docs/04_architecture/client_performance.md"}, BaseQueue: baseQ, HeadQueue: headQ}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("protected spec change by implementer not rejected")
	}
	// implementer changing another packet's fields fails
	other := queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil, []string{"server/internal/mine/"}, nil, "- t"),
		mkPacket("IMP-902", "IN_PROGRESS", "a", "imp/IMP-902-x", "", nil, []string{"server/internal/theirs/"}, nil, "- t"),
	)
	otherHead := strings.Replace(other, `imp/IMP-902-x`, `imp/IMP-902-y`, 1)
	d = PRDiff{Role: gates.RoleImplementer, Branch: "imp/IMP-901-x", TaskID: "IMP-901",
		Files: []string{taskQueuePath}, BaseQueue: other, HeadQueue: otherHead}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("foreign packet field change not rejected")
	}
	// implementer own packet IN_PROGRESS -> DONE is legal; files in scope pass
	doneHead := strings.Replace(baseQ, "status: IN_PROGRESS", "status: DONE", 1)
	d = PRDiff{Role: gates.RoleImplementer, Branch: "imp/IMP-901-x", TaskID: "IMP-901",
		Files:     []string{taskQueuePath, "server/internal/mine/x.go", "docs/10_implementation/evidence/IMP-901/manifest.json"},
		BaseQueue: baseQ, HeadQueue: doneHead}
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("valid implementer diff flagged: %v", got)
	}
	// implementer may not remove lines from known_blockers.md
	baseB := mkBlockers(blkEntry("BLK-100", "IMP-901"), "")
	headB := mkBlockers("", "")
	d = PRDiff{Role: gates.RoleImplementer, Branch: "imp/IMP-901-x", TaskID: "IMP-901",
		Files: []string{knownBlockersPath}, BaseQueue: baseQ, HeadQueue: headQ,
		BaseBlockers: baseB, HeadBlockers: headB}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("non-append edit to known_blockers.md not rejected")
	}
	// implementer may not write evidence of another task
	d = PRDiff{Role: gates.RoleImplementer, Branch: "imp/IMP-901-x", TaskID: "IMP-901",
		Files: []string{"docs/10_implementation/evidence/IMP-902/manifest.json"}, BaseQueue: baseQ, HeadQueue: headQ}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("foreign evidence write not rejected")
	}
	// coordinator may touch only control files
	d = PRDiff{Role: gates.RoleCoordinator, Branch: "claim/IMP-901", TaskID: "",
		Files: []string{"server/internal/mine/x.go"}, BaseQueue: baseQ, HeadQueue: headQ}
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("coordinator code change not rejected")
	}
}

func TestBlockPrAllowedFields(t *testing.T) {
	baseQ := queueOf(
		mkPacket("IMP-901", "IN_PROGRESS", "a", "imp/IMP-901-x", "", nil,
			[]string{"server/internal/mine/"}, nil, "- t"),
		mkPacket("IMP-902", "IN_PROGRESS", "a", "imp/IMP-902-x", "", nil,
			[]string{"server/internal/theirs/"}, nil, "- t"),
	)
	baseB := mkBlockers("", "")
	headB := mkBlockers(blkEntry("BLK-100", "IMP-901"), "")

	blockHead := func(q string, id string) string {
		out := strings.Replace(q, "id: "+id+"\nstatus: IN_PROGRESS", "id: "+id+"\nstatus: BLOCKED", 1)
		return strings.Replace(out, "id: "+id+"\nstatus: BLOCKED\nclaimed_by: \"a\"\nbranch: \"imp/"+id+"-x\"\nclaimed_at: \"2026-01-01T00:00:00Z\"\nblocked_by: \"\"",
			"id: "+id+"\nstatus: BLOCKED\nclaimed_by: \"a\"\nbranch: \"imp/"+id+"-x\"\nclaimed_at: \"2026-01-01T00:00:00Z\"\nblocked_by: \"BLK-100\"", 1)
	}

	// valid block/: own packet IN_PROGRESS -> BLOCKED + blocked_by + appended entry
	d := PRDiff{Role: gates.RoleImplementer, Branch: "block/IMP-901-1", TaskID: "IMP-901",
		Files:     []string{taskQueuePath, knownBlockersPath},
		BaseQueue: baseQ, HeadQueue: blockHead(baseQ, "IMP-901"),
		BaseBlockers: baseB, HeadBlockers: headB}
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("valid block/ PR flagged: %v", got)
	}
	// missing blocked_by fails
	noBB := strings.Replace(d.HeadQueue, `blocked_by: "BLK-100"`, ``, 1)
	d.HeadQueue = strings.Replace(noBB, "status: BLOCKED", "status: BLOCKED", 1)
	d.HeadQueue = noBB
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("BLOCKED without blocked_by not rejected")
	}
	// blocked_by naming an entry that was never appended fails
	d.HeadQueue = blockHead(baseQ, "IMP-901")
	d.HeadBlockers = baseB
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("blocked_by to nonexistent entry not rejected")
	}
	d.HeadBlockers = headB
	// a block/ PR that leaves its own packet IN_PROGRESS fails
	d.HeadQueue = baseQ
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("block/ without own BLOCKED transition not rejected")
	}
	// a block/ PR flipping a *different* packet to BLOCKED fails
	d.HeadQueue = blockHead(blockHead(baseQ, "IMP-901"), "IMP-902")
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("block/ flipping a foreign packet not rejected")
	}
}

func TestOpsPrAllowedFields(t *testing.T) {
	baseQ := queueOf(mkPacket("IMP-901", "BLOCKED", "a", "imp/IMP-901-x", "OPS-001", nil,
		[]string{"server/internal/mine/"}, nil, "- t"))
	baseB := mkBlockers(blkEntry("OPS-001", "IMP-901"), "")
	// valid resolve: OPS-001 -> Resolved, task BLOCKED -> NOT_STARTED
	headQ := strings.Replace(baseQ, "status: BLOCKED", "status: NOT_STARTED", 1)
	headQ = strings.Replace(headQ, `blocked_by: "OPS-001"`, `blocked_by: ""`, 1)
	headQ = strings.Replace(headQ, `claimed_by: "a"`, `claimed_by: ""`, 1)
	headQ = strings.Replace(headQ, `branch: "imp/IMP-901-x"`, `branch: ""`, 1)
	headQ = strings.Replace(headQ, `claimed_at: "2026-01-01T00:00:00Z"`, `claimed_at: ""`, 1)
	headB := mkBlockers("", blkEntry("OPS-001", "IMP-901")+"resolved_by: owner\n")
	d := PRDiff{Role: gates.RoleCoordinator, Branch: "ops/fix-1",
		Files:     []string{taskQueuePath, knownBlockersPath},
		BaseQueue: baseQ, HeadQueue: headQ,
		BaseBlockers: baseB, HeadBlockers: headB}
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("valid ops/ resolve flagged: %v", got)
	}
	// resolving OPS while the blocked task stays BLOCKED fails
	d2 := d
	d2.HeadQueue = baseQ // packet still BLOCKED
	if got := ControlDiffProblems(d2); len(got) == 0 {
		t.Fatal("ops/ resolve leaving listed task BLOCKED not rejected")
	}
	// ops/ opening a new OPS entry (no task flips) is legal
	d3 := PRDiff{Role: gates.RoleCoordinator, Branch: "ops/new-1",
		Files:     []string{knownBlockersPath},
		BaseQueue: baseQ, HeadQueue: baseQ,
		BaseBlockers: baseB, HeadBlockers: mkBlockers(blkEntry("OPS-001", "IMP-901")+blkEntry("OPS-002", "ALL"), "")}
	if got := ControlDiffProblems(d3); len(got) != 0 {
		t.Fatalf("ops/ open entry flagged: %v", got)
	}
	// ops/ touching code fails
	d4 := PRDiff{Role: gates.RoleCoordinator, Branch: "ops/x",
		Files: []string{"server/internal/mine/x.go"}, BaseQueue: baseQ, HeadQueue: baseQ}
	if got := ControlDiffProblems(d4); len(got) == 0 {
		t.Fatal("ops/ code change not rejected")
	}
	// ops/ flipping a task not linked to a resolved OPS fails
	d5 := d
	d5.HeadBlockers = baseB // OPS-001 still open
	if got := ControlDiffProblems(d5); len(got) == 0 {
		t.Fatal("ops/ unlinked BLOCKED->NOT_STARTED not rejected")
	}
	// ops/ setting DONE fails
	doneQ := strings.Replace(baseQ, "status: BLOCKED", "status: DONE", 1)
	d6 := PRDiff{Role: gates.RoleCoordinator, Branch: "ops/x",
		Files: []string{taskQueuePath}, BaseQueue: baseQ, HeadQueue: doneQ,
		BaseBlockers: baseB, HeadBlockers: baseB}
	if got := ControlDiffProblems(d6); len(got) == 0 {
		t.Fatal("ops/ DONE transition not rejected")
	}
}

func TestBlockedToNotStartedOnlyBySpecOrOps(t *testing.T) {
	mk := func(branch, blockedBy string, baseB, headB string) PRDiff {
		baseQ := queueOf(mkPacket("IMP-901", "BLOCKED", "a", "imp/IMP-901-x", blockedBy, nil,
			[]string{"server/internal/mine/"}, nil, "- t"))
		headQ := strings.Replace(baseQ, "status: BLOCKED", "status: NOT_STARTED", 1)
		headQ = strings.Replace(headQ, `blocked_by: "`+blockedBy+`"`, `blocked_by: ""`, 1)
		headQ = strings.Replace(headQ, `claimed_by: "a"`, `claimed_by: ""`, 1)
		headQ = strings.Replace(headQ, `branch: "imp/IMP-901-x"`, `branch: ""`, 1)
		headQ = strings.Replace(headQ, `claimed_at: "2026-01-01T00:00:00Z"`, `claimed_at: ""`, 1)
		role := gates.RoleFromBranch(branch)
		files := []string{taskQueuePath}
		if baseB != headB {
			files = append(files, knownBlockersPath)
		}
		return PRDiff{Role: role, Branch: branch, TaskID: gates.TaskFromBranch(branch),
			Files:     files,
			BaseQueue: baseQ, HeadQueue: headQ,
			BaseBlockers: baseB, HeadBlockers: headB}
	}
	openBlk := mkBlockers(blkEntry("BLK-050", "IMP-901"), "")
	resBlk := mkBlockers("", blkEntry("BLK-050", "IMP-901")+"resolved_by: spec\n")
	openOps := mkBlockers(blkEntry("OPS-002", "IMP-901"), "")
	resOps := mkBlockers("", blkEntry("OPS-002", "IMP-901")+"resolved_by: owner\n")

	// implementer branches may never un-block
	for _, br := range []string{"imp/IMP-901-x", "block/IMP-901-1"} {
		d := mk(br, "BLK-050", openBlk, resBlk)
		if got := ControlDiffProblems(d); len(got) == 0 {
			t.Fatalf("%s allowed BLOCKED->NOT_STARTED", br)
		}
	}
	// claim/ only for REVERT-<sha> blocks
	d := mk("claim/IMP-901", "BLK-050", openBlk, resBlk)
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("claim/ allowed BLK un-block")
	}
	d = mk("claim/IMP-901", "REVERT-deadbeef", mkBlockers("", ""), mkBlockers("", ""))
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("claim/ REVERT un-block flagged: %v", got)
	}
	// spec/ un-blocks only when the BLK is resolved at head
	d = mk("spec/x", "BLK-050", openBlk, resBlk)
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("spec/ BLK-resolve un-block flagged: %v", got)
	}
	d = mk("spec/x", "BLK-050", openBlk, openBlk)
	if got := ControlDiffProblems(d); len(got) == 0 {
		t.Fatal("spec/ un-blocked a still-open BLK")
	}
	// ops/ un-blocks tasks of the resolved OPS
	d = mk("ops/x", "OPS-002", openOps, resOps)
	if got := ControlDiffProblems(d); len(got) != 0 {
		t.Fatalf("ops/ resolve un-block flagged: %v", got)
	}
}

func TestMetaImpliedByOwnership(t *testing.T) {
	p := gates.TaskPacket{ID: "IMP-901", OwnedPaths: []string{"client/Assets/Scripts/Core/Runtime/"}}
	cases := []struct {
		f    string
		want bool
	}{
		{"client/Assets/Scripts/Core/Runtime/FrameLoop.cs", true},
		{"client/Assets/Scripts/Core/Runtime/FrameLoop.cs.meta", true}, // P.meta owned with P
		{"client/Assets/Scripts/Core/Runtime.meta", true},              // .meta of owned dir
		{"client/Assets/Scripts/Core.meta", true},                      // .meta of parent folder of owned path
		{"client/Assets/Scripts/Core/Assets/Loc.cs.meta", false},       // foreign .meta
		{"client/Assets/Scripts/UI/Hud.cs", false},                     // foreign file
	}
	for _, c := range cases {
		if got := OwnedFile(p, c.f); got != c.want {
			t.Errorf("OwnedFile(%q) = %v, want %v", c.f, got, c.want)
		}
	}
}
