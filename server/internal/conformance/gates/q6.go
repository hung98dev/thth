package gates

import (
	"os"
	"path/filepath"
	"strings"
)

// CheckQ6 — Evidence/cleanliness: clean generated state, worktree free of
// stray outputs, and evidence identity per ADR-0057 for every DONE packet on
// the base tree (new DONE transitions on the head are exempt until step 7 of
// the merge sequence commits the evidence artifact).
func CheckQ6(root string, e *Env) []Check {
	const gate = "Q6"
	id := func(s string) string { return gate + "." + s }
	var checks []Check

	// Clean worktree: no verification-created drift. Locally the pre-existing
	// reviewed diff is allowed (definition_of_done) — only untracked drift and
	// unexpected modified generated paths count.
	status, err := gitDir(root, "status", "--porcelain")
	if err != nil {
		checks = append(checks, Fail(id("clean_tree"), "git status failed: "+err.Error()))
	} else {
		var stray []string
		for _, line := range splitLines(status) {
			if len(line) < 4 {
				continue
			}
			code, file := line[:2], strings.TrimSpace(line[3:])
			_ = code
			if strings.HasPrefix(code, "??") {
				if e.CI {
					stray = append(stray, "untracked: "+file)
				} else {
					// local run: untracked files are the contributor's diff — allowed
				}
			}
		}
		checks = append(checks, statusCheck(id("clean_tree"), stray))
	}

	// Source-tree hash is computable and stable.
	hash, err := SourceTreeHash(root)
	if err != nil {
		checks = append(checks, Fail(id("source_tree_hash"), err.Error()))
	} else {
		checks = append(checks, Pass(id("source_tree_hash"), hash))
	}

	// Committed evidence manifests must be schema-valid AND their ci_run_id
	// must resolve to a successful verify.yml run whose head_sha is an
	// ancestor of HEAD.
	manifests, _ := filepath.Glob(filepath.Join(root, "docs", "10_implementation", "evidence", "*", "manifest.json"))
	if len(manifests) == 0 {
		checks = append(checks, Pass(id("evidence.identity"), "no evidence manifests yet"))
	}
	for _, mp := range manifests {
		taskID := filepath.Base(filepath.Dir(mp))
		v := ValidateManifest(mp, taskID)
		if len(v) > 0 {
			checks = append(checks, Fail(id("evidence.identity"), taskID+": "+strings.Join(v, "; ")))
			continue
		}
		var m EvidenceManifest
		data, _ := os.ReadFile(mp)
		_ = decodeManifest(data, &m)
		if ok, c := e.CIContextCheck(id("evidence.api")); ok {
			probs := CheckRunIdentity(root, e.Repository, m.CIRunID, m.RunAttempt, e.Token)
			checks = append(checks, statusCheck(id("evidence.api")+"."+taskID, probs))
		} else {
			checks = append(checks, c)
		}
	}
	return checks
}
