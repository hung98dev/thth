package taskgraph

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"thinhthan/internal/conformance/gates"
)

// PRDiff is the control-file view of a pull request: the branch-derived role
// and task, the base..head file list and the base/head text of the two
// control files.
type PRDiff struct {
	Role         gates.PRRole
	Branch       string
	TaskID       string // imp//block/ task extracted from the branch name
	Files        []string
	BaseQueue    string
	HeadQueue    string
	BaseBlockers string
	HeadBlockers string
}

// BuildPRDiff collects the PRDiff for the env's PR context (base..head).
func BuildPRDiff(root string, e *gates.Env) (PRDiff, error) {
	d := PRDiff{
		Role:   gates.RoleFromBranch(e.HeadRef),
		Branch: e.HeadRef,
		TaskID: gates.TaskFromBranch(e.HeadRef),
	}
	files, err := gates.DiffFiles(root, e.BaseSHA, e.HeadSHA)
	if err != nil {
		return d, err
	}
	d.Files = files
	if d.BaseQueue, err = refFileOr(root, e.BaseSHA, taskQueuePath); err != nil {
		return d, err
	}
	if d.BaseBlockers, err = refFileOr(root, e.BaseSHA, knownBlockersPath); err != nil {
		return d, err
	}
	d.HeadQueue = readOr(root, taskQueuePath)
	d.HeadBlockers = readOr(root, knownBlockersPath)
	return d, nil
}

func refFileOr(root, ref, path string) (string, error) {
	s, err := gates.RefFile(root, ref, path)
	if err != nil {
		return "", fmt.Errorf("read %s at %s: %w", path, ref, err)
	}
	return s, nil
}

func readOr(root, rel string) string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	return string(data)
}

// mutablePacketFields are the only packet fields a non-spec-owner PR may
// change (agent_execution_protocol.md §3 + audit_gates.md § Protected Paths).
var mutablePacketFields = map[string]bool{
	"status": true, "claimed_by": true, "branch": true, "claimed_at": true, "blocked_by": true,
}

// ControlDiffProblems enforces the branch-prefix table of audit_gates.md §
// Protected Paths on the control files: file scope per role, per-packet
// field diffs, status-transition legality and known_blockers.md entry rules
// (block/ appends an entry; ops/ records or resolves an OPS entry and flips
// the tasks it blocked).
func ControlDiffProblems(d PRDiff) []string {
	var problems []string
	if d.Role == gates.RoleNone {
		return []string{fmt.Sprintf("branch %q matches no PR role prefix (spec//claim//ops//imp//block//revert/)", d.Branch)}
	}
	isClaim := strings.HasPrefix(d.Branch, "claim/")
	isOps := strings.HasPrefix(d.Branch, "ops/")
	isBlock := strings.HasPrefix(d.Branch, "block/")

	basePackets, baseSummary, errB := gates.ParseTaskQueueText(d.BaseQueue)
	headPackets, headSummary, errH := gates.ParseTaskQueueText(d.HeadQueue)
	baseBlk := ParseBlockers(d.BaseBlockers)
	headBlk := ParseBlockers(d.HeadBlockers)

	baseByID := map[string]gates.TaskPacket{}
	for _, p := range basePackets {
		baseByID[p.ID] = p
	}
	headByID := map[string]gates.TaskPacket{}
	for _, p := range headPackets {
		headByID[p.ID] = p
	}

	// ---- file scope by role ----------------------------------------------
	for _, f := range d.Files {
		switch d.Role {
		case gates.RoleImplementer:
			if f == taskQueuePath || f == knownBlockersPath {
				continue
			}
			if strings.HasPrefix(f, "docs/10_implementation/evidence/"+d.TaskID+"/") {
				continue
			}
			if d.TaskID == "" {
				problems = append(problems, "non-control file outside owned_paths: "+f)
				continue
			}
			p, ok := headByID[d.TaskID]
			if !ok {
				problems = append(problems, "no packet for "+d.TaskID+"; cannot scope "+f)
				continue
			}
			if !OwnedFile(p, f) {
				problems = append(problems, fmt.Sprintf("file %s outside %s owned_paths", f, d.TaskID))
			}
		case gates.RoleCoordinator:
			if !gates.IsControlFile(f) {
				problems = append(problems, "coordinator PR may not change non-control file "+f)
				continue
			}
			if isClaim && f != taskQueuePath {
				problems = append(problems, "claim/ may change only "+taskQueuePath)
			}
			if isOps && f != taskQueuePath && f != knownBlockersPath {
				problems = append(problems, "ops/ may change only "+taskQueuePath+" and "+knownBlockersPath)
			}
		case gates.RoleSpecOwner, gates.RoleMergeGuard:
			// unrestricted file scope
		}
	}

	// ---- packet field diffs + status transitions -------------------------
	if queueTouched(d) {
		if errB != nil {
			problems = append(problems, "base task_queue.md does not parse: "+errB.Error())
		}
		if errH != nil {
			problems = append(problems, "head task_queue.md does not parse: "+errH.Error())
		}
		for _, hp := range headPackets {
			bp, exists := baseByID[hp.ID]
			if !exists {
				if d.Role != gates.RoleSpecOwner {
					problems = append(problems, hp.ID+": new packet added by non-spec-owner PR")
				}
				continue
			}
			if hp.ChangeBody != bp.ChangeBody || hp.AcceptanceBody != bp.AcceptanceBody || hp.TestsBody != bp.TestsBody {
				if d.Role != gates.RoleSpecOwner {
					problems = append(problems, hp.ID+": body sections are immutable for "+string(d.Role)+" PRs")
				}
				continue
			}
			problems = append(problems, fieldDiffProblems(d, bp, hp)...)
			problems = append(problems, transitionProblems(d, bp, hp, baseBlk, headBlk)...)
		}
		for _, bp := range basePackets {
			if _, ok := headByID[bp.ID]; !ok && d.Role != gates.RoleSpecOwner {
				problems = append(problems, bp.ID+": packet removed by non-spec-owner PR")
			}
		}
		// summary rows: only rows of legally transitioned packets may change,
		// and the new cell must equal the packet's head status.
		for id, s := range headSummary {
			if baseSummary[id] == s {
				continue
			}
			if d.Role == gates.RoleImplementer && id != d.TaskID {
				problems = append(problems, fmt.Sprintf("summary row of %s changed by %s's PR", id, d.TaskID))
				continue
			}
			if hp, ok := headByID[id]; ok && hp.Status != s {
				problems = append(problems, fmt.Sprintf("summary row of %s (%s) does not match packet status %s", id, s, hp.Status))
			}
		}
		for id := range baseSummary {
			if _, ok := headSummary[id]; !ok && d.Role != gates.RoleSpecOwner {
				problems = append(problems, id+": summary row removed by non-spec-owner PR")
			}
		}
	}

	// ---- known_blockers.md content rules ---------------------------------
	if blockersTouched(d) {
		switch {
		case d.Role == gates.RoleImplementer:
			if !isSubsequence(d.BaseBlockers, d.HeadBlockers) {
				problems = append(problems, "known_blockers.md is append-only for implementer PRs")
			}
		case isClaim:
			problems = append(problems, "claim/ may not change "+knownBlockersPath)
		case isOps:
			problems = append(problems, opsBlockerProblems(baseBlk, headBlk)...)
		}
	}

	// ---- role-specific packet-state assertions ---------------------------
	if isBlock && d.TaskID != "" {
		hp, ok := headByID[d.TaskID]
		if !ok {
			problems = append(problems, "block/ PR: packet "+d.TaskID+" not found")
		} else {
			if hp.Status != "BLOCKED" || hp.BlockedBy == "" {
				problems = append(problems, "block/ PR must set "+d.TaskID+" to BLOCKED with blocked_by")
			} else if _, open := headBlk.Open[hp.BlockedBy]; !open && !revertRefRe.MatchString(hp.BlockedBy) {
				problems = append(problems, "block/ PR: blocked_by "+hp.BlockedBy+" names no open entry")
			}
			// The PR must append the entry it cites when it is new.
			if hp.BlockedBy != "" {
				if _, inBase := baseBlk.AllIDs()[hp.BlockedBy]; !inBase {
					if !isSubsequence(d.BaseBlockers, d.HeadBlockers) {
						problems = append(problems, "block/ PR must append the "+hp.BlockedBy+" entry to known_blockers.md")
					}
				}
			}
		}
	}
	if isOps {
		problems = append(problems, opsFlipProblems(baseByID, headByID, baseBlk, headBlk)...)
	}

	return dedup(problems)
}

func queueTouched(d PRDiff) bool    { return containsStr(d.Files, taskQueuePath) }
func blockersTouched(d PRDiff) bool { return containsStr(d.Files, knownBlockersPath) }

func containsStr(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

var fieldLineRe = regexp.MustCompile(`^([a-z_]+):\s*(.*)$`)

// packetFields extracts `key: value` field lines from a packet's raw text.
func packetFields(p gates.TaskPacket) map[string]string {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(p.Raw))
	for sc.Scan() {
		trim := strings.TrimSpace(sc.Text())
		if fm := fieldLineRe.FindStringSubmatch(trim); fm != nil {
			out[fm[1]] = fm[2]
		}
	}
	return out
}

// fieldDiffProblems applies the per-role field-change rules to one packet.
func fieldDiffProblems(d PRDiff, bp, hp gates.TaskPacket) []string {
	bf, hf := packetFields(bp), packetFields(hp)
	var changed []string
	for k, v := range hf {
		if bv, ok := bf[k]; !ok || bv != v {
			changed = append(changed, k)
		}
	}
	for k := range bf {
		if _, ok := hf[k]; !ok {
			changed = append(changed, k+" (removed)")
		}
	}
	if len(changed) == 0 {
		return nil
	}
	var problems []string
	switch d.Role {
	case gates.RoleImplementer:
		if hp.ID != d.TaskID {
			problems = append(problems, fmt.Sprintf("%s: fields %v changed by %s's PR", hp.ID, changed, d.TaskID))
			break
		}
		for _, k := range changed {
			name := strings.TrimSuffix(k, " (removed)")
			if !mutablePacketFields[name] {
				problems = append(problems, fmt.Sprintf("%s: immutable field %q changed", hp.ID, name))
			}
		}
	case gates.RoleCoordinator:
		for _, k := range changed {
			name := strings.TrimSuffix(k, " (removed)")
			if !mutablePacketFields[name] {
				problems = append(problems, fmt.Sprintf("%s: coordinator may not change field %q", hp.ID, name))
			}
		}
	case gates.RoleMergeGuard:
		for _, k := range changed {
			name := strings.TrimSuffix(k, " (removed)")
			if name != "status" && name != "blocked_by" {
				problems = append(problems, fmt.Sprintf("%s: revert/ may not change field %q", hp.ID, name))
			}
		}
	}
	return problems
}

// transitionProblems enforces the status-transition table of audit_gates.md
// § Protected Paths:
//
//	NOT_STARTED -> IN_PROGRESS   claim/ only
//	IN_PROGRESS -> NOT_STARTED   claim/ only (unclaim)
//	IN_PROGRESS -> BLOCKED       block/ of the task, or revert/ (dependents)
//	BLOCKED    -> NOT_STARTED    spec/ (BLK resolved), ops/ (OPS resolved,
//	                            listed tasks), claim/ (REVERT-<sha> unclaim)
//	IN_PROGRESS -> DONE          imp/ of the task
//	DONE       -> IN_PROGRESS    revert/ only (reverted squash)
//	all others                   illegal
func transitionProblems(d PRDiff, bp, hp gates.TaskPacket, baseBlk, headBlk BlockerDoc) []string {
	bt, ht := bp.Status, hp.Status
	if bt == ht {
		return nil
	}
	transition := bt + " -> " + ht
	allowed := false
	switch {
	case ht == "DONE":
		allowed = d.Role == gates.RoleImplementer && strings.HasPrefix(d.Branch, "imp/") &&
			hp.ID == d.TaskID && bt == "IN_PROGRESS"
	case bt == "NOT_STARTED" && ht == "IN_PROGRESS":
		allowed = strings.HasPrefix(d.Branch, "claim/")
	case bt == "IN_PROGRESS" && ht == "NOT_STARTED":
		allowed = strings.HasPrefix(d.Branch, "claim/")
	case ht == "BLOCKED":
		allowed = (strings.HasPrefix(d.Branch, "block/") && hp.ID == d.TaskID && bt == "IN_PROGRESS") ||
			d.Role == gates.RoleMergeGuard
	case bt == "BLOCKED" && ht == "NOT_STARTED":
		allowed = unblockedLegally(d, bp, hp, baseBlk, headBlk)
	case bt == "DONE" && ht == "IN_PROGRESS":
		allowed = d.Role == gates.RoleMergeGuard
	}
	if !allowed {
		return []string{fmt.Sprintf("%s: illegal status transition %s on %s", hp.ID, transition, d.Branch)}
	}
	return nil
}

// unblockedLegally reports whether a BLOCKED -> NOT_STARTED flip is legal:
// spec/ resolves the BLK it names, ops/ resolves the OPS entry it names (or
// that lists the task), claim/ unclaims only REVERT-<sha> blocks.
func unblockedLegally(d PRDiff, bp, hp gates.TaskPacket, baseBlk, headBlk BlockerDoc) bool {
	switch {
	case strings.HasPrefix(d.Branch, "claim/"):
		return strings.HasPrefix(bp.BlockedBy, "REVERT-")
	case strings.HasPrefix(d.Branch, "ops/"):
		for id, entry := range baseBlk.Open {
			if _, resolved := headBlk.Resolved[id]; !resolved {
				continue
			}
			if bp.BlockedBy == id {
				return true
			}
			for _, t := range entry.Blocks {
				if t == hp.ID {
					return true
				}
			}
		}
		return false
	case d.Role == gates.RoleSpecOwner:
		if !strings.HasPrefix(bp.BlockedBy, "BLK-") {
			return false
		}
		_, resolved := headBlk.Resolved[bp.BlockedBy]
		return resolved
	}
	return false
}

// opsBlockerProblems enforces the ops/ entry rules on known_blockers.md: an
// OPS entry is appended under Open, or moved from Open to Resolved (never
// deleted; a resolve may add resolved_* fields but keeps its id and body).
func opsBlockerProblems(baseBlk, headBlk BlockerDoc) []string {
	var problems []string
	for id := range baseBlk.Open {
		if _, still := headBlk.Open[id]; still {
			continue
		}
		if _, resolved := headBlk.Resolved[id]; !resolved {
			problems = append(problems, "ops/ removed open entry "+id+" without resolving it")
		}
	}
	for id := range headBlk.Resolved {
		if _, wasOpen := baseBlk.Open[id]; wasOpen {
			continue
		}
		if _, already := baseBlk.Resolved[id]; !already {
			problems = append(problems, "ops/ resolved entry "+id+" that was not open at base")
		}
	}
	return problems
}

// opsFlipProblems requires every task blocked by an OPS entry resolved by
// this PR to transition BLOCKED -> NOT_STARTED.
func opsFlipProblems(baseByID, headByID map[string]gates.TaskPacket, baseBlk, headBlk BlockerDoc) []string {
	var problems []string
	for id, entry := range baseBlk.Open {
		if _, resolved := headBlk.Resolved[id]; !resolved {
			continue
		}
		must := map[string]bool{}
		for tid, bp := range baseByID {
			if bp.BlockedBy == id && bp.Status == "BLOCKED" {
				must[tid] = true
			}
		}
		for _, t := range entry.Blocks {
			if bp, ok := baseByID[t]; ok && bp.Status == "BLOCKED" {
				must[t] = true
			}
		}
		for tid := range must {
			if hp, ok := headByID[tid]; !ok || hp.Status != "NOT_STARTED" {
				problems = append(problems, fmt.Sprintf("ops/ resolved %s but %s is not NOT_STARTED", id, tid))
			}
		}
	}
	return problems
}

// isSubsequence reports whether every non-empty line of base appears in head
// in order — the append-only contract for known_blockers.md.
func isSubsequence(base, head string) bool {
	bl := nonEmptyLines(base)
	hl := nonEmptyLines(head)
	i, j := 0, 0
	for i < len(bl) && j < len(hl) {
		if bl[i] == hl[j] {
			i++
		}
		j++
	}
	return i == len(bl)
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
