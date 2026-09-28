package taskgraph

import (
	"strings"

	"thinhthan/internal/conformance/gates"
)

// CheckQ0 assembles the IMP-083-owned Q0 sub-gates as gates.Check values so
// the verifier can wire them once the task's own gate activates.
func CheckQ0(root string, e *gates.Env) []gates.Check {
	var checks []gates.Check
	id := func(s string) string { return "Q0.tg." + s }

	packets, _, err := gates.ParseTaskQueue(root)
	if err != nil {
		return append(checks, gates.Fail(id("parse"), "task_queue.md does not parse: "+err.Error()))
	}
	checks = append(checks, gates.Pass(id("parse"), "task_queue.md parses"))

	blockers := LoadBlockers(root)
	checks = append(checks, statusCheck(id("dag_acyclic"), DagAcyclicProblems(packets)))
	checks = append(checks, statusCheck(id("dangling_refs"), DanglingRefProblems(packets, fileExistsFunc(root), blockers)))
	checks = append(checks, statusCheck(id("states_claim_fields"), StatusProblems(packets)))
	checks = append(checks, statusCheck(id("owned_forbidden"), OwnedForbiddenProblems(packets)))
	checks = append(checks, statusCheck(id("test_paths_owned"), TestPathProblems(packets, blockers.AllIDs())))
	checks = append(checks, statusCheck(id("requirement_coverage"), RequirementCoverageProblems(root, packets)))

	// Control-file diff + transitions need a PR context; they defer locally.
	if ok, c := e.PRContextCheck(id("control_diff")); !ok {
		checks = append(checks, c)
	} else {
		d, derr := BuildPRDiff(root, e)
		if derr != nil {
			checks = append(checks, gates.Fail(id("control_diff"), "cannot compute PR diff: "+derr.Error()))
		} else {
			checks = append(checks, statusCheck(id("control_diff"), ControlDiffProblems(d)))
		}
	}
	return checks
}

func statusCheck(id string, problems []string) gates.Check {
	if len(problems) == 0 {
		return gates.Pass(id, "")
	}
	if len(problems) > 8 {
		problems = append(problems[:8], "...")
	}
	return gates.Fail(id, strings.Join(problems, "; "))
}
