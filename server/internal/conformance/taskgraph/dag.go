package taskgraph

import (
	"regexp"
	"sort"
	"strings"

	"thinhthan/internal/conformance/gates"
)

// DagAcyclicProblems reports self-dependencies and dependency cycles in the
// packet DAG (architecture_conformance.md §4 rule 1; task_queue.md is the
// single source of truth, so a cycle is a graph defect, not a doc wart).
func DagAcyclicProblems(packets []gates.TaskPacket) []string {
	byID := map[string]gates.TaskPacket{}
	for _, p := range packets {
		byID[p.ID] = p
	}
	var problems []string
	for _, p := range packets {
		for _, d := range p.DependsOn {
			if d == p.ID {
				problems = append(problems, p.ID+": depends on itself")
			}
		}
	}
	problems = append(problems, cycleProblems(packets, byID)...)
	return dedup(problems)
}

// DanglingRefProblems reports references that do not resolve:
// depends_on ids must be packets, specs must be files under docs/ (resolved
// relative to docs/10_implementation/), adrs must be files under
// docs/11_decisions/, and blocked_by must name an open BLK/OPS entry or a
// REVERT-<sha> guard marker (architecture_conformance.md §4 rule 2).
func DanglingRefProblems(packets []gates.TaskPacket, exists func(rel string) bool, blockers BlockerDoc) []string {
	byID := map[string]bool{}
	for _, p := range packets {
		byID[p.ID] = true
	}
	var problems []string
	for _, p := range packets {
		for _, d := range p.DependsOn {
			if !byID[d] {
				problems = append(problems, p.ID+": depends_on "+d+" is not a packet")
			}
		}
		for _, s := range p.Specs {
			rel := specRel(s)
			if rel == "" {
				problems = append(problems, p.ID+": spec "+s+" escapes docs/")
				continue
			}
			if exists != nil && !exists(rel) {
				problems = append(problems, p.ID+": spec "+s+" does not resolve ("+rel+")")
			}
		}
		for _, a := range p.ADRs {
			rel := adrDir + "/" + a
			if exists != nil && !exists(rel) {
				problems = append(problems, p.ID+": adr "+a+" does not resolve ("+rel+")")
			}
		}
		if p.Status == "BLOCKED" && p.BlockedBy != "" {
			if !isOpenBlockerRef(blockers, p.BlockedBy) {
				problems = append(problems, p.ID+": blocked_by "+p.BlockedBy+" names no open BLK/OPS entry or REVERT-<sha>")
			}
		}
	}
	return dedup(problems)
}

var revertRefRe = regexp.MustCompile(`^REVERT-[0-9a-f]{7,40}$`)

// isOpenBlockerRef reports whether ref names an open BLK/OPS entry or a
// post-merge-guard REVERT-<sha> marker.
func isOpenBlockerRef(blockers BlockerDoc, ref string) bool {
	if _, ok := blockers.Open[ref]; ok {
		return true
	}
	return revertRefRe.MatchString(ref)
}

// specRel resolves a packet spec entry relative to docs/10_implementation/;
// "" when the result would escape docs/.
func specRel(s string) string {
	rel := joinSlash(specDir, s)
	for strings.HasPrefix(rel, "../") {
		rel = strings.TrimPrefix(rel, "../")
	}
	if !strings.HasPrefix(rel, "docs/") {
		return ""
	}
	return rel
}

// joinSlash joins slash paths without OS dependence (repo paths are always
// forward-slash).
func joinSlash(base, rel string) string {
	parts := strings.Split(base+"/"+rel, "/")
	var out []string
	for _, p := range parts {
		switch p {
		case "", ".":
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			} else {
				out = append(out, "..")
			}
		default:
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

// StatusProblems enforces the status enum and the claim-field schema
// (agent_execution_protocol.md §3): IN_PROGRESS/BLOCKED/DONE packets must
// carry claimed_by/branch/claimed_at; BLOCKED must carry blocked_by.
func StatusProblems(packets []gates.TaskPacket) []string {
	var problems []string
	for _, p := range packets {
		switch p.Status {
		case "NOT_STARTED", "IN_PROGRESS", "BLOCKED", "DONE":
		default:
			problems = append(problems, p.ID+": invalid status "+p.Status)
			continue
		}
		switch p.Status {
		case "IN_PROGRESS", "BLOCKED", "DONE":
			if p.ClaimedBy == "" {
				problems = append(problems, p.ID+": "+p.Status+" without claimed_by")
			}
			if p.Branch == "" {
				problems = append(problems, p.ID+": "+p.Status+" without branch")
			}
			if !claimedAtRe.MatchString(p.ClaimedAt) {
				problems = append(problems, p.ID+": "+p.Status+" without a claimed_at timestamp")
			}
		}
		if p.Status == "BLOCKED" && p.BlockedBy == "" {
			problems = append(problems, p.ID+": BLOCKED without blocked_by")
		}
	}
	return dedup(problems)
}

var claimedAtRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// cycleProblems runs a three-colour DFS over depends_on and reports each
// elementary cycle once as "a -> b -> a".
func cycleProblems(packets []gates.TaskPacket, byID map[string]gates.TaskPacket) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var cycles []string

	var visit func(id string)
	visit = func(id string) {
		color[id] = gray
		stack = append(stack, id)
		for _, dep := range byID[id].DependsOn {
			if _, ok := byID[dep]; !ok {
				continue // dangling: reported by DanglingRefProblems
			}
			switch color[dep] {
			case gray:
				// found a back edge: report the cycle segment.
				start := 0
				for i, s := range stack {
					if s == dep {
						start = i
						break
					}
				}
				cyc := append(append([]string{}, stack[start:]...), dep)
				cycles = append(cycles, "dependency cycle: "+strings.Join(cyc, " -> "))
			case white:
				visit(dep)
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
	}
	var ids []string
	for _, p := range packets {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == white {
			visit(id)
		}
	}
	return cycles
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
