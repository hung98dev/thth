package gates

import (
	"fmt"
	"path"
	"strings"
)

// PRClassification is Q0's read of the pull request: role, task, diff scope,
// fast-path and transition flags used to select gates.
type PRClassification struct {
	Role           PRRole
	TaskID         string   // task the implementer branch belongs to (imp//block/)
	Files          []string // diff file list base..head
	StatusOnly     bool     // control-file-only diff -> Q0-only fast path
	DoneTransition bool     // some packet's status becomes DONE on head
}

// ClassifyPR derives the classification from env + the diff.
func ClassifyPR(root string, e *Env) (*PRClassification, error) {
	files, err := DiffFiles(root, e.BaseSHA, e.HeadSHA)
	if err != nil {
		return nil, err
	}
	c := &PRClassification{
		Role:   RoleFromBranch(e.HeadRef),
		TaskID: TaskFromBranch(e.HeadRef),
		Files:  files,
	}
	// DONE transition detection needs base vs head packet statuses; defer on
	// parse failures — the packet checks report them separately.
	if data, err := RefFile(root, e.BaseSHA, "docs/10_implementation/task_queue.md"); err == nil {
		if bps, _, err := ParseTaskQueueText(data); err == nil {
			baseStatus := map[string]string{}
			for _, p := range bps {
				baseStatus[p.ID] = p.Status
			}
			if hps, _, err := ParseTaskQueue(root); err == nil {
				for _, p := range hps {
					if p.Status == "DONE" && baseStatus[p.ID] != "DONE" {
						c.DoneTransition = true
					}
				}
			}
		}
	}
	c.StatusOnly = StatusOnlyPR(files) && !c.DoneTransition
	return c, nil
}

// checkControlFileDiff enforces audit_gates.md § Protected Paths: implementer
// PRs change control content only in the allowed fields; other roles are
// unrestricted on control files but never touch code outside their scope.
func checkControlFileDiff(root string, e *Env, headPackets map[string]TaskPacket) []Check {
	const id = "Q0.control.diff"
	if ok, c := e.PRContextCheck(id); !ok {
		return []Check{c}
	}
	cl, err := ClassifyPR(root, e)
	if err != nil {
		return []Check{Fail(id, "cannot compute PR diff: "+err.Error())}
	}
	var checks []Check
	var problems []string

	// Role derivation: any other prefix fails.
	if cl.Role == RoleNone {
		problems = append(problems, fmt.Sprintf("branch %q matches no PR role prefix (spec//claim//ops//imp//block//revert/)", e.HeadRef))
	}

	// Per-file scope rules by role.
	for _, f := range cl.Files {
		switch cl.Role {
		case RoleImplementer:
			// Evidence is not a control file (keeps status-only PRs honest);
			// an implementer may only write evidence/<own task>/.
			if strings.HasPrefix(f, "docs/10_implementation/evidence/") {
				if !implementerControlFileAllowed(f, cl.TaskID) {
					problems = append(problems, "implementer may not change control file "+f)
				}
				continue
			}
			if IsControlFile(f) {
				if !implementerControlFileAllowed(f, cl.TaskID) {
					problems = append(problems, "implementer may not change control file "+f)
				}
				continue
			}
			if cl.TaskID == "" {
				problems = append(problems, "non-control file outside owned_paths: "+f)
				continue
			}
			if p, ok := headPackets[cl.TaskID]; ok {
				if !ownedFile(p, f) {
					problems = append(problems, fmt.Sprintf("file %s outside %s owned_paths", f, cl.TaskID))
				}
			} else {
				problems = append(problems, fmt.Sprintf("no packet for %s; cannot scope %s", cl.TaskID, f))
			}
		case RoleCoordinator:
			if !IsControlFile(f) {
				problems = append(problems, "coordinator PR may not change non-control file "+f)
			}
		case RoleSpecOwner, RoleMergeGuard:
			// unrestricted scope
		}
	}

	// Field-level rules for implementer edits of task_queue.md.
	if cl.Role == RoleImplementer && contains(cl.Files, "docs/10_implementation/task_queue.md") {
		problems = append(problems, implementerPacketDiff(root, e, cl.TaskID, headPackets)...)
	}

	// known_blockers.md is append-only for implementers.
	if cl.Role == RoleImplementer && contains(cl.Files, "docs/10_implementation/known_blockers.md") {
		if removed := removedLines(root, e, "docs/10_implementation/known_blockers.md"); len(removed) > 0 {
			problems = append(problems, "known_blockers.md is append-only for implementer PRs (removed lines: "+strings.Join(removed, ", ")+")")
		}
	}

	// block/ must transition IN_PROGRESS -> BLOCKED + blocked_by.
	if strings.HasPrefix(e.HeadRef, "block/") && cl.TaskID != "" {
		problems = append(problems, blockTransitionProblems(root, e, cl.TaskID, headPackets)...)
	}

	checks = append(checks, statusCheck(id, problems))
	checks = append(checks, Pass("Q0.control.role", "role="+string(cl.Role)+" task="+cl.TaskID+
		fmt.Sprintf(" status_only=%v done_transition=%v files=%d", cl.StatusOnly, cl.DoneTransition, len(cl.Files))))
	return checks
}

// implementerControlFileAllowed: an implementer may touch its own packet's
// mutable fields + summary row (task_queue.md), append known_blockers.md and
// write evidence/<own ID>/.
func implementerControlFileAllowed(f, taskID string) bool {
	switch {
	case f == "docs/10_implementation/task_queue.md":
		return true // field-level check separate
	case f == "docs/10_implementation/known_blockers.md":
		return true // appends validated by content check below
	case strings.HasPrefix(f, "docs/10_implementation/evidence/"+taskID+"/"):
		return true
	}
	return false
}

// implementerPacketDiff diffs packet fields base->head and requires: identical
// packets for all other tasks; only mutable fields changed on the PR's task;
// no body-section changes anywhere.
func implementerPacketDiff(root string, e *Env, taskID string, headPackets map[string]TaskPacket) []string {
	var problems []string
	data, err := RefFile(root, e.BaseSHA, "docs/10_implementation/task_queue.md")
	if err != nil {
		return []string{"cannot read task_queue.md at base: " + err.Error()}
	}
	basePackets, baseSummary, err := ParseTaskQueueText(data)
	if err != nil {
		return []string{"cannot parse base task_queue.md: " + err.Error()}
	}
	baseByID := map[string]TaskPacket{}
	for _, p := range basePackets {
		baseByID[p.ID] = p
	}
	for _, hp := range headPackets {
		bp, ok := baseByID[hp.ID]
		if !ok {
			continue // new packet — only coordinators may add; flag if implementer
			// covered by "implementer may not change control file"? Keep: flag.
		}
		if hp.ChangeBody != bp.ChangeBody || hp.AcceptanceBody != bp.AcceptanceBody || hp.TestsBody != bp.TestsBody {
			problems = append(problems, hp.ID+": body sections are immutable for implementer PRs")
			continue
		}
		bf, hf := packetFieldMap(bp), packetFieldMap(hp)
		for k, v := range hf {
			if bf[k] == v {
				continue
			}
			if hp.ID != taskID {
				problems = append(problems, fmt.Sprintf("%s: field %q changed by %s's PR", hp.ID, k, taskID))
				continue
			}
			if !mutablePacketFields[k] {
				problems = append(problems, fmt.Sprintf("%s: immutable field %q changed", hp.ID, k))
			}
		}
		for k := range bf {
			if _, ok := hf[k]; !ok {
				problems = append(problems, fmt.Sprintf("%s: field %q removed", hp.ID, k))
			}
		}
	}
	// Summary rows: other tasks' rows may not change.
	_, headSummary, err := ParseTaskQueue(root)
	if err == nil {
		for id, s := range headSummary {
			if id != taskID && baseSummary[id] != s {
				problems = append(problems, fmt.Sprintf("summary row of %s changed by %s's PR", id, taskID))
			}
		}
	}
	return problems
}

// blockTransitionProblems asserts a block/ PR transitions its packet
// IN_PROGRESS -> BLOCKED with a blocked_by entry.
func blockTransitionProblems(root string, e *Env, taskID string, headPackets map[string]TaskPacket) []string {
	var problems []string
	hp, ok := headPackets[taskID]
	if !ok {
		return []string{"packet " + taskID + " not found"}
	}
	if hp.Status != "BLOCKED" || hp.BlockedBy == "" {
		problems = append(problems, fmt.Sprintf("block/ PR must set %s to BLOCKED with blocked_by", taskID))
	}
	return problems
}

// ownedFile reports whether a changed file is inside the packet's
// owned_paths, including implied .meta ownership (a .meta for a path under
// owned_paths or for a folder that contains one).
func ownedFile(p TaskPacket, f string) bool {
	candidates := []string{f}
	if strings.HasSuffix(f, ".meta") {
		base := strings.TrimSuffix(f, ".meta")
		candidates = append(candidates, base)
	}
	for _, o := range p.OwnedPaths {
		oc := path.Clean(o)
		for _, c := range candidates {
			cc := path.Clean(c)
			if cc == oc || strings.HasPrefix(cc, oc+"/") {
				return true
			}
			// .meta of a parent folder of an owned path
			if strings.HasSuffix(f, ".meta") && strings.HasPrefix(oc, cc+"/") {
				return true
			}
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
