package gates

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Q0 — Task/spec integrity (audit_gates.md Q0-Q6 Contract): DAG, links,
// states, transitions, claim fields, control-file diff rules, requirement-ID
// coverage, evidence schema.
func CheckQ0(root string, e *Env) []Check {
	const gate = "Q0"
	id := func(s string) string { return gate + "." + s }
	var checks []Check

	packets, summary, err := ParseTaskQueue(root)
	if err != nil {
		return []Check{Fail(id("task_queue.parse"), fmt.Sprintf("cannot parse task_queue.md: %v", err))}
	}
	if len(packets) == 0 {
		return []Check{Fail(id("task_queue.packets"), "no IMP packets found")}
	}
	checks = append(checks, Pass(id("task_queue.parse"), fmt.Sprintf("%d packets parsed", len(packets))))

	byID := map[string]TaskPacket{}
	ids := map[string]bool{}
	var dup []string
	for _, p := range packets {
		if ids[p.ID] {
			dup = append(dup, p.ID)
		}
		ids[p.ID] = true
		byID[p.ID] = p
	}
	if len(dup) > 0 {
		checks = append(checks, Fail(id("packet.unique_ids"), "duplicate packet ids: "+strings.Join(dup, ", ")))
	} else {
		checks = append(checks, Pass(id("packet.unique_ids"), ""))
	}

	var badStatus, missingFields []string
	for _, p := range packets {
		if !validStatus[p.Status] {
			badStatus = append(badStatus, p.ID+"="+p.Status)
		}
		if len(p.Specs) == 0 || len(p.OwnedPaths) == 0 || p.EvidenceLocation == "" ||
			strings.TrimSpace(p.ChangeBody) == "" || strings.TrimSpace(p.AcceptanceBody) == "" ||
			strings.TrimSpace(p.TestsBody) == "" {
			missingFields = append(missingFields, p.ID)
		}
	}
	checks = append(checks, statusCheck(id("packet.status_enum"), badStatus))
	checks = append(checks, statusCheck(id("packet.schema"), missingFields))

	var claimErr []string
	for _, p := range packets {
		switch p.Status {
		case "IN_PROGRESS", "DONE":
			if p.ClaimedBy == "" || p.Branch == "" || p.ClaimedAt == "" {
				claimErr = append(claimErr, p.ID+" is "+p.Status+" without claim fields")
			}
		case "BLOCKED":
			if p.BlockedBy == "" {
				claimErr = append(claimErr, p.ID+" is BLOCKED without blocked_by")
			}
		}
	}
	checks = append(checks, statusCheck(id("packet.claim_fields"), claimErr))

	var summaryMismatch []string
	for _, p := range packets {
		s, ok := summary[p.ID]
		if !ok {
			summaryMismatch = append(summaryMismatch, p.ID+": absent from summary table")
			continue
		}
		if s != p.Status {
			summaryMismatch = append(summaryMismatch, fmt.Sprintf("%s: table=%s packet=%s", p.ID, s, p.Status))
		}
	}
	for sid := range summary {
		if !ids[sid] {
			summaryMismatch = append(summaryMismatch, sid+": table row without packet")
		}
	}
	checks = append(checks, statusCheck(id("packet.summary_parity"), summaryMismatch))

	var missingDep []string
	for _, p := range packets {
		for _, d := range p.DependsOn {
			if !ids[d] {
				missingDep = append(missingDep, fmt.Sprintf("%s -> %s", p.ID, d))
			}
		}
	}
	checks = append(checks, statusCheck(id("dag.depends_on_exists"), missingDep))

	if cyc := findCycle(packets, ids); cyc != "" {
		checks = append(checks, Fail(id("dag.acyclic"), "dependency cycle: "+cyc))
	} else {
		checks = append(checks, Pass(id("dag.acyclic"), ""))
	}

	var ownedForbidden []string
	for _, p := range packets {
		for _, o := range p.OwnedPaths {
			for _, f := range p.ForbiddenPaths {
				if o == f || strings.HasPrefix(o, strings.TrimSuffix(f, "/")+"/") ||
					strings.HasPrefix(f, strings.TrimSuffix(o, "/")+"/") {
					ownedForbidden = append(ownedForbidden, fmt.Sprintf("%s: %s vs %s", p.ID, o, f))
				}
			}
		}
	}
	checks = append(checks, statusCheck(id("paths.owned_not_forbidden"), ownedForbidden))

	reaches := depReachability(packets, ids)
	var overlapViol []string
	for i := 0; i < len(packets); i++ {
		for j := i + 1; j < len(packets); j++ {
			a, b := packets[i], packets[j]
			for _, oa := range a.OwnedPaths {
				for _, ob := range b.OwnedPaths {
					if pathsOverlap(oa, ob) && !reaches[a.ID][b.ID] && !reaches[b.ID][a.ID] {
						overlapViol = append(overlapViol,
							fmt.Sprintf("%s and %s share %s ~ %s with no dependency between them", a.ID, b.ID, oa, ob))
					}
				}
			}
		}
	}
	checks = append(checks, statusCheck(id("paths.ownership_overlap"), dedup(overlapViol)))

	var missingSpec, missingADR []string
	for _, p := range packets {
		for _, s := range p.Specs {
			full := filepath.Join(root, "docs", "10_implementation", filepath.FromSlash(s))
			if st, err := os.Stat(full); err != nil || st.IsDir() {
				missingSpec = append(missingSpec, fmt.Sprintf("%s: %s", p.ID, s))
			}
		}
		for _, a := range p.ADRs {
			if _, err := os.Stat(filepath.Join(root, "docs", "11_decisions", filepath.FromSlash(a))); err != nil {
				missingADR = append(missingADR, fmt.Sprintf("%s: %s", p.ID, a))
			}
		}
	}
	checks = append(checks, statusCheck(id("specs.resolve"), missingSpec))
	checks = append(checks, statusCheck(id("adrs.resolve"), missingADR))

	var gateViol []string
	for _, p := range packets {
		if p.Status != "IN_PROGRESS" && p.Status != "DONE" {
			continue
		}
		for _, d := range p.DependsOn {
			if dep, ok := byID[d]; ok && dep.Status != "DONE" {
				gateViol = append(gateViol, fmt.Sprintf("%s is %s but depends_on %s (%s)", p.ID, p.Status, d, dep.Status))
			}
		}
	}
	checks = append(checks, statusCheck(id("dag.done_gating"), gateViol))

	checks = append(checks, checkLayerMilestoneOrder(root, packets, byID)...)
	checks = append(checks, checkTraceability(root, packets, byID)...)
	checks = append(checks, checkRequirementCoverage(root, packets)...)
	checks = append(checks, checkOpenBlockerGating(root, byID)...)
	checks = append(checks, checkBootstrapAbsence(root))

	checks = append(checks, checkEvidenceOnDisk(root, packets)...)
	checks = append(checks, checkDoneManifestRule(root, e, packets)...)

	// Control-file diff rules (PR-scoped; defers locally).
	checks = append(checks, checkControlFileDiff(root, e, byID)...)

	return checks
}

func statusCheck(id string, problems []string) Check {
	if len(problems) > 0 {
		return Fail(id, strings.Join(dedup(problems), "; "))
	}
	return Pass(id, "")
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

// findCycle detects a depends_on cycle, returning the path or "".
func findCycle(packets []TaskPacket, ids map[string]bool) string {
	byID := map[string]TaskPacket{}
	for _, p := range packets {
		byID[p.ID] = p
	}
	color := map[string]int{}
	var stack []string
	var cycle string
	var visit func(n string) bool
	visit = func(n string) bool {
		if color[n] == 1 {
			cycle = strings.Join(append(stack, n), " -> ")
			return true
		}
		if color[n] == 2 {
			return false
		}
		color[n] = 1
		stack = append(stack, n)
		for _, d := range byID[n].DependsOn {
			if ids[d] && visit(d) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = 2
		return false
	}
	for _, p := range packets {
		if visit(p.ID) {
			break
		}
	}
	return cycle
}

// depReachability computes transitive depends_on reachability.
func depReachability(packets []TaskPacket, ids map[string]bool) map[string]map[string]bool {
	deps := map[string][]string{}
	for _, p := range packets {
		deps[p.ID] = p.DependsOn
	}
	reaches := map[string]map[string]bool{}
	var visit func(from, cur string, seen map[string]bool)
	visit = func(from, cur string, seen map[string]bool) {
		for _, d := range deps[cur] {
			if !ids[d] || seen[d] {
				continue
			}
			seen[d] = true
			reaches[from][d] = true
			visit(from, d, seen)
		}
	}
	for _, p := range packets {
		reaches[p.ID] = map[string]bool{}
		visit(p.ID, p.ID, map[string]bool{p.ID: true})
	}
	return reaches
}

// checkLayerMilestoneOrder asserts a task's layer (dependency_graph.md) and
// milestone (milestones.md) are never lower than those of its depends_on.
func checkLayerMilestoneOrder(root string, packets []TaskPacket, byID map[string]TaskPacket) []Check {
	const id = "Q0.dag.layer_milestone_order"
	layers, err := parseIDNumberTable(filepath.Join(root, "docs", "10_implementation", "dependency_graph.md"), "Layer")
	if err != nil {
		return []Check{Fail(id, "dependency_graph.md layer table missing: "+err.Error())}
	}
	milestones, err := parseIDNumberTable(filepath.Join(root, "docs", "10_implementation", "milestones.md"), "M")
	if err != nil {
		return []Check{Fail(id, "milestones.md milestone table missing: "+err.Error())}
	}
	var problems []string
	for _, p := range packets {
		for _, d := range p.DependsOn {
			if _, ok := byID[d]; !ok {
				continue
			}
			if l, okL := layers[p.ID]; okL {
				if ld, okD := layers[d]; okD && l < ld {
					problems = append(problems, fmt.Sprintf("layer inversion: %s (layer %d) depends on %s (layer %d)", p.ID, l, d, ld))
				}
			}
			if ms, okM := milestones[p.ID]; okM {
				if md, okD := milestones[d]; okD && ms < md {
					problems = append(problems, fmt.Sprintf("milestone inversion: %s (M%d) depends on %s (M%d)", p.ID, ms, d, md))
				}
			}
		}
	}
	return []Check{statusCheck(id, problems)}
}

// parseIDNumberTable reads markdown tables mapping IMP ids to a numbered
// value. kind "Layer" parses both `| 0 | name | IMP-x, ... |` group rows
// and `| IMP-000 | 0 |` task rows; any other kind parses
// `| <kind><n> | IMP-x, ... |` membership rows (e.g. "| M0 | IMP-000 |").
func parseIDNumberTable(path, kind string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	taskRe := regexp.MustCompile(`\bIMP-\d+\b`)
	var grpRe, mapRe *regexp.Regexp
	if kind == "Layer" {
		grpRe = regexp.MustCompile(`^\|\s*(\d+)\s*\|[^|]*\|\s*([^|]+)\|`)
		mapRe = regexp.MustCompile(`^\|\s*(IMP-\d+)\s*\|\s*(\d+)\s*\|`)
	} else {
		grpRe = regexp.MustCompile(`^\|\s*` + kind + `(\d+)\s*\|\s*([^|]+)\|`)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if m := grpRe.FindStringSubmatch(line); m != nil {
			var n int
			fmt.Sscanf(m[1], "%d", &n)
			for _, t := range taskRe.FindAllString(m[2], -1) {
				out[t] = n
			}
			continue
		}
		if mapRe != nil {
			if m := mapRe.FindStringSubmatch(line); m != nil {
				var n int
				fmt.Sscanf(m[2], "%d", &n)
				out[m[1]] = n
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no %s table rows parsed", kind)
	}
	return out, nil
}

var reqIDRe = regexp.MustCompile("`([A-Z]{2,6}-[0-9]{3})`")

// checkRequirementCoverage (Gate B): every requirement ID declared in a
// "Requirement IDs" table of docs/00_context..09_testing or
// engineering_conventions.md must be named in at least one packet's
// ## Acceptance AND ## Tests of the same packet.
func checkRequirementCoverage(root string, packets []TaskPacket) []Check {
	const id = "Q0.coverage.requirement_ids"
	declared := map[string]string{} // id -> declaring file
	dirs := []string{
		filepath.Join(root, "docs", "00_context"),
		filepath.Join(root, "docs", "01_gameplay"),
		filepath.Join(root, "docs", "02_world"),
		filepath.Join(root, "docs", "03_systems"),
		filepath.Join(root, "docs", "04_architecture"),
		filepath.Join(root, "docs", "05_network"),
		filepath.Join(root, "docs", "06_data"),
		filepath.Join(root, "docs", "07_security"),
		filepath.Join(root, "docs", "07_content"),
		filepath.Join(root, "docs", "08_scale_ops"),
		filepath.Join(root, "docs", "09_testing"),
	}
	dirs = append(dirs, filepath.Join(root, "docs", "10_implementation", "engineering_conventions.md"))
	for _, d := range dirs {
		st, err := os.Stat(d)
		if err != nil {
			continue
		}
		var files []string
		if st.IsDir() {
			entries, _ := os.ReadDir(d)
			for _, en := range entries {
				if !en.IsDir() && strings.HasSuffix(en.Name(), ".md") {
					files = append(files, filepath.Join(d, en.Name()))
				}
			}
		} else {
			files = []string{d}
		}
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			section := requirementIDsSection(string(data))
			if section == "" {
				continue
			}
			for _, m := range reqIDRe.FindAllStringSubmatch(section, -1) {
				declared[m[1]] = filepath.ToSlash(mustRel(root, f))
			}
		}
	}
	var problems []string
	var idsSorted []string
	for idv := range declared {
		idsSorted = append(idsSorted, idv)
	}
	sort.Strings(idsSorted)
	for _, idv := range idsSorted {
		covered := false
		for _, p := range packets {
			if strings.Contains(p.AcceptanceBody, idv) && strings.Contains(p.TestsBody, idv) {
				covered = true
				break
			}
		}
		if !covered {
			problems = append(problems, fmt.Sprintf("%s (declared in %s) is named in no packet's Acceptance+Tests", idv, declared[idv]))
		}
	}
	return []Check{statusCheck(id, problems)}
}

// requirementIDsSection extracts the body of a "Requirement IDs" section
// (##/### heading) up to the next same-or-higher heading; "" when absent.
var reqIDsHeadingRe = regexp.MustCompile(`(?m)^#{2,3} Requirement IDs?\s*$`)
var nextHeadingRe = regexp.MustCompile(`(?m)^#{1,3} `)

func requirementIDsSection(text string) string {
	m := reqIDsHeadingRe.FindStringIndex(text)
	if m == nil {
		return ""
	}
	rest := text[m[1]:]
	if n := nextHeadingRe.FindStringIndex(rest); n != nil {
		return rest[:n[0]]
	}
	return rest
}

func mustRel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return r
}

// checkTraceability enforces spec_traceability.md parity with the packets:
// every spec file under docs/ (except 11_decisions/, evidence, this index,
// task_queue.md, known_blockers.md, templates) has a consuming-task row, and
// every listed task actually names the spec/ADR in its packet.
func checkTraceability(root string, packets []TaskPacket, byID map[string]TaskPacket) []Check {
	const gate = "Q0.traceability"
	var problems []string

	// Spec coverage: every docs/**/*.md (except 11_decisions/, evidence/,
	// task_queue.md, known_blockers.md, spec_traceability.md, templates/) must
	// appear in the index's Spec → Tasks table.
	indexPath := filepath.Join(root, "docs", "10_implementation", "spec_traceability.md")
	idxData, err := os.ReadFile(indexPath)
	if err != nil {
		return []Check{Fail(gate+".index_missing", "spec_traceability.md missing")}
	}
	idx := string(idxData)

	docsRoot := filepath.Join(root, "docs")
	var uncovered []string
	_ = filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(docsRoot, path))
		switch {
		case strings.HasPrefix(rel, "11_decisions/"),
			strings.HasPrefix(rel, "10_implementation/evidence/"),
			rel == "10_implementation/task_queue.md",
			rel == "10_implementation/known_blockers.md",
			rel == "10_implementation/spec_traceability.md",
			strings.HasPrefix(rel, "templates/"):
			return nil
		}
		if !strings.Contains(idx, "`"+rel+"`") {
			uncovered = append(uncovered, rel)
		}
		return nil
	})
	for _, rel := range uncovered {
		problems = append(problems, "spec file has no consuming task row: "+rel)
	}

	// ADR coverage: every 11_decisions/*.md must appear in the index.
	adrDir := filepath.Join(root, "docs", "11_decisions")
	adrEntries, _ := os.ReadDir(adrDir)
	for _, en := range adrEntries {
		if en.IsDir() || !strings.HasSuffix(en.Name(), ".md") {
			continue
		}
		if !strings.Contains(idx, "`"+en.Name()+"`") && !strings.Contains(idx, "11_decisions/"+en.Name()) {
			problems = append(problems, "ADR has no consuming task row: "+en.Name())
		}
	}

	// Index↔packet parity: a spec listed in a packet's specs: must be a real
	// row, and every index row must match packets (loose parity: every
	// packet-listed spec appears in the index).
	for _, p := range packets {
		for _, s := range p.Specs {
			norm := strings.TrimPrefix(s, "../")
			if strings.HasPrefix(s, "../") {
				if !strings.Contains(idx, "`"+norm+"`") {
					problems = append(problems, fmt.Sprintf("%s specs: %s not in traceability index", p.ID, s))
				}
			} else {
				if !strings.Contains(idx, "`10_implementation/"+s+"`") && !strings.Contains(idx, "`"+s+"`") {
					problems = append(problems, fmt.Sprintf("%s specs: %s not in traceability index", p.ID, s))
				}
			}
		}
	}

	return []Check{statusCheck(gate+".parity", problems)}
}

// checkOpenBlockerGating: a task named by a Blocks: line under an open
// blocker may only be NOT_STARTED or BLOCKED.
func checkOpenBlockerGating(root string, byID map[string]TaskPacket) []Check {
	blockedTasks := openBlockedTasks(filepath.Join(root, "docs", "10_implementation", "known_blockers.md"))
	var viol []string
	for blk, tasks := range blockedTasks {
		for _, t := range tasks {
			p, ok := byID[t]
			if !ok {
				continue
			}
			if p.Status != "NOT_STARTED" && p.Status != "BLOCKED" {
				viol = append(viol, fmt.Sprintf("%s blocks %s but %s is %s", blk, t, t, p.Status))
			}
		}
	}
	return []Check{statusCheck("Q0.blockers.open_gating", viol)}
}

var blocksLineRe = regexp.MustCompile(`^(?:(?:[-*+]|\d+[.)])\s*)?Blocks:\s*(.*)$`)
var openBlockerRe = regexp.MustCompile("^###\\s+`?(BLK-[0-9]+|OPS-[0-9]+)`?")

// openBlockedTasks returns task ids named by `Blocks:` lines under each open
// blocker of known_blockers.md.
func openBlockedTasks(path string) map[string][]string {
	out := map[string][]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	inOpen := false
	curBLK := ""
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "## ") {
			inOpen = strings.Contains(line, "Open Blockers")
			curBLK = ""
			continue
		}
		if !inOpen {
			continue
		}
		if m := openBlockerRe.FindStringSubmatch(line); m != nil {
			curBLK = m[1]
			continue
		}
		if curBLK == "" {
			continue
		}
		if m := blocksLineRe.FindStringSubmatch(line); m != nil {
			for _, ref := range taskIDRefRe.FindAllString(m[1], -1) {
				out[curBLK] = append(out[curBLK], ref)
			}
		}
	}
	return out
}

// checkBootstrapAbsence enforces IMP-000's acceptance rule: proto/,
// migrations, generated outputs and feature paths remain absent until their
// owning task.
func checkBootstrapAbsence(root string) Check {
	const id = "Q0.bootstrap.absent_paths"
	var problems []string
	for _, rel := range []string{
		"proto",
		"server/migrations",
		"server/cmd/server",
		"client/Assets/Scenes",
		"client/Assets/Prefabs",
		"client/Assets/Art",
		"client/Assets/AddressableAssetsData",
		"client/Assets/Localization",
		"client/Assets/Settings",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			problems = append(problems, rel+" must not exist before its owning task")
		}
	}
	// Generated protocol dirs may exist holding only their asmdef (IMP-000
	// owns the asmdef; generated .cs/.go come from IMP-061).
	for _, dir := range []string{
		"client/Assets/Scripts/Protocol",
		"server/internal/protocol",
	} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			continue
		}
		for _, en := range entries {
			if en.IsDir() {
				continue
			}
			name := en.Name()
			if !strings.HasSuffix(name, ".asmdef") && !strings.HasSuffix(name, ".meta") {
				problems = append(problems, dir+"/"+name+" present before its owning task")
			}
		}
	}
	return statusCheck(id, problems)
}

// checkEvidenceOnDisk validates every committed evidence manifest schema-v2.
func checkEvidenceOnDisk(root string, packets []TaskPacket) []Check {
	const id = "Q0.evidence.manifests"
	var problems []string
	evRoot := filepath.Join(root, "docs", "10_implementation", "evidence")
	manifests, _ := filepath.Glob(filepath.Join(evRoot, "*", "manifest.json"))
	for _, mp := range manifests {
		taskID := filepath.Base(filepath.Dir(mp))
		for _, p := range ValidateManifest(mp, taskID) {
			problems = append(problems, taskID+": "+p)
		}
	}
	if len(manifests) == 0 && len(problems) == 0 {
		return []Check{Pass(id, "no manifests yet")}
	}
	return []Check{statusCheck(id, problems)}
}

// checkDoneManifestRule enforces ADR-0072: a packet DONE on the base (already
// merged) must carry its evidence manifest on this tree; a head that newly
// sets DONE without the manifest is allowed (step 7 commits it before merge).
// required_evidence: "none" exempts a task.
func checkDoneManifestRule(root string, e *Env, packets []TaskPacket) []Check {
	const id = "Q0.evidence.done_manifest"
	if ok, c := e.PRContextCheck(id); !ok {
		return []Check{c}
	}
	basePackets := map[string]TaskPacket{}
	if data, err := RefFile(root, e.BaseSHA, "docs/10_implementation/task_queue.md"); err == nil {
		if ps, _, err := ParseTaskQueueText(data); err == nil {
			for _, p := range ps {
				basePackets[p.ID] = p
			}
		}
	}
	var problems []string
	for _, p := range packets {
		if p.Status != "DONE" || p.RequiredEvidence == "none" {
			continue
		}
		manifestPath := filepath.Join(root, "docs", "10_implementation", "evidence", p.ID, "manifest.json")
		if _, err := os.Stat(manifestPath); err == nil {
			continue // manifest present on this head
		}
		if bp, ok := basePackets[p.ID]; ok && bp.Status != "DONE" {
			continue // fresh DONE transition on this head — allowed until step 7
		}
		problems = append(problems, fmt.Sprintf("%s is DONE without evidence/%s/manifest.json on this tree", p.ID, p.ID))
	}
	return []Check{statusCheck(id, problems)}
}
