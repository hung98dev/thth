package gates

import (
	"regexp"
	"strings"
)

// PRRole is the role a PR plays, derived from its head-branch prefix
// (audit_gates.md § Protected Paths).
type PRRole string

const (
	RoleSpecOwner   PRRole = "spec-owner"
	RoleCoordinator PRRole = "coordinator"
	RoleImplementer PRRole = "implementer"
	RoleMergeGuard  PRRole = "merge-guard"
	RoleNone        PRRole = "none"
)

// RoleFromBranch maps a head branch to its PR role. A branch outside the
// reserved prefixes fails Q0.
func RoleFromBranch(branch string) PRRole {
	switch {
	case strings.HasPrefix(branch, "spec/"):
		return RoleSpecOwner
	case strings.HasPrefix(branch, "claim/"), strings.HasPrefix(branch, "ops/"):
		return RoleCoordinator
	case strings.HasPrefix(branch, "imp/"), strings.HasPrefix(branch, "block/"):
		return RoleImplementer
	case strings.HasPrefix(branch, "revert/"):
		return RoleMergeGuard
	default:
		return RoleNone
	}
}

// TaskFromBranch extracts the IMP id an implementer branch belongs to:
// imp/IMP-000-bootstrap -> IMP-000, imp/IMP-000-done -> IMP-000,
// block/IMP-000-1 -> IMP-000. Other roles return "".
var branchTaskRe = regexp.MustCompile(`^(IMP-[0-9]+)`)

func TaskFromBranch(branch string) string {
	for _, p := range []string{"imp/", "block/"} {
		if strings.HasPrefix(branch, p) {
			if m := branchTaskRe.FindStringSubmatch(strings.TrimPrefix(branch, p)); m != nil {
				return m[1]
			}
		}
	}
	return ""
}

// twoPhaseTasks are the tasks whose implementation PR merges at IN_PROGRESS
// and whose `imp/IMP-XXX-done` status PR performs the DONE transition
// (audit_gates.md § Two-Phase, ADR-0068/0072).
var twoPhaseTasks = []string{
	"IMP-000", "IMP-061", "IMP-003", "IMP-004", "IMP-005", "IMP-083", "IMP-065", "IMP-068",
}

// TwoPhaseTasks returns the canonical two-phase task list.
func TwoPhaseTasks() []string { return append([]string{}, twoPhaseTasks...) }

// IsTwoPhase reports whether taskID is a two-phase gate task.
func IsTwoPhase(taskID string) bool {
	for _, t := range twoPhaseTasks {
		if t == taskID {
			return true
		}
	}
	return false
}

// ControlFiles is the set of repo paths a status-only PR may touch: the
// packet fields + summary row of task_queue.md and known_blockers.md
// appends. Evidence files are deliberately NOT control files — a PR that
// touches only evidence would otherwise take the Q0-only fast path and skip
// every gate. Implementer writes under evidence/<own task>/ are allowed by
// the per-file scope check instead.
var controlFilePrefixes = []string{
	"docs/10_implementation/task_queue.md",
	"docs/10_implementation/known_blockers.md",
}

// IsControlFile reports whether path is a status/control file.
func IsControlFile(path string) bool {
	for _, p := range controlFilePrefixes {
		if path == strings.TrimSuffix(p, "/") || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// StatusOnlyPR reports whether a PR's changed file list is entirely control
// files — the Q0-only fast path (audit_gates.md § Protected Paths). A PR that
// sets DONE is never status-only.
func StatusOnlyPR(files []string) bool {
	if len(files) == 0 {
		return false
	}
	for _, f := range files {
		if !IsControlFile(f) {
			return false
		}
	}
	return true
}

// GateRequired reports whether a gate owned by ownerTask must run given the
// packet statuses on main (base) and on the PR head (ADR-0068).
func GateRequired(ownerTask string, statusOnMain, statusOnHead string) bool {
	return statusOnMain == "DONE" || statusOnHead == "DONE"
}

// FirstUnmetOwner returns the first owner whose DONE is not satisfied by the
// required predicate, or "" when every owner is DONE.
func FirstUnmetOwner(owners []string, required func(string) bool) string {
	for _, o := range owners {
		if !required(o) {
			return o
		}
	}
	return ""
}

// AnyOwnerRequired reports whether at least one owner is DONE (multi-owner
// gates like Q5).
func AnyOwnerRequired(owners []string, required func(string) bool) bool {
	for _, o := range owners {
		if required(o) {
			return true
		}
	}
	return false
}
