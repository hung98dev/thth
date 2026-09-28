package taskgraph

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"thinhthan/internal/conformance/gates"
)

var (
	reqIDRe         = regexp.MustCompile("`([A-Z]{2,6}-[0-9]{3})`")
	reqIDsHeadingRe = regexp.MustCompile(`(?m)^#{2,3} Requirement IDs?\s*$`)
	nextHeadingRe   = regexp.MustCompile(`(?m)^#{1,3} `)
)

// specScanDirs are the directories whose "Requirement IDs" tables declare the
// normative requirement ids every packet must cover (Gate B of
// audit_gates.md).
var specScanDirs = []string{
	"docs/00_context",
	"docs/01_gameplay",
	"docs/02_world",
	"docs/03_systems",
	"docs/04_architecture",
	"docs/05_network",
	"docs/06_data",
	"docs/07_security",
	"docs/07_content",
	"docs/08_scale_ops",
	"docs/09_testing",
}

// specScanFiles are singleton spec files scanned in addition to the dirs.
var specScanFiles = []string{
	"docs/10_implementation/engineering_conventions.md",
}

// RequirementCoverageProblems implements Gate B: every `AAA-NNN` requirement
// id declared in a spec "Requirement IDs" table must appear in at least one
// packet's ## Acceptance AND the same packet's ## Tests.
func RequirementCoverageProblems(root string, packets []gates.TaskPacket) []string {
	declared := map[string]string{} // id -> declaring file (repo-relative)
	scanFile := func(rel string) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return
		}
		section := requirementIDsSection(string(data))
		if section == "" {
			return
		}
		for _, m := range reqIDRe.FindAllStringSubmatch(section, -1) {
			if _, ok := declared[m[1]]; !ok {
				declared[m[1]] = rel
			}
		}
	}
	for _, dir := range specScanDirs {
		full := filepath.Join(root, filepath.FromSlash(dir))
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		for _, en := range entries {
			if en.IsDir() || !strings.HasSuffix(en.Name(), ".md") {
				continue
			}
			scanFile(dir + "/" + en.Name())
		}
	}
	for _, f := range specScanFiles {
		scanFile(f)
	}
	var problems []string
	var ids []string
	for id := range declared {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		covered := false
		for _, p := range packets {
			if strings.Contains(p.AcceptanceBody, id) && strings.Contains(p.TestsBody, id) {
				covered = true
				break
			}
		}
		if !covered {
			problems = append(problems, fmt.Sprintf(
				"%s (declared in %s) is named in no packet's Acceptance+Tests", id, declared[id]))
		}
	}
	return problems
}

// requirementIDsSection extracts the body of a "Requirement IDs" section
// (##/### heading) up to the next same-or-higher heading; "" when absent.
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
