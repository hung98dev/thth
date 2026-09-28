// Package taskgraph implements the IMP-083-owned Q0 task-graph conformance
// rules (docs/10_implementation/architecture_conformance.md §4, audit_gates.md
// § Protected Paths): DAG acyclicity, reference resolution, status/claim-field
// schema, per-role status transitions, owned/forbidden path overlap ordered
// by depends_on, test-path ownership, requirement-ID coverage and the
// control-file diff rules for spec//claim//ops//imp//block//revert/ PRs.
//
// Rule functions return problem strings so mutation fixtures can pin
// fail-closed behaviour without git; CheckQ0 adapts them to []gates.Check
// for the verifier (audit_gates.md § Gate Activation).
package taskgraph

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"thinhthan/internal/conformance/gates"
)

const (
	taskQueuePath     = "docs/10_implementation/task_queue.md"
	knownBlockersPath = "docs/10_implementation/known_blockers.md"
	specDir           = "docs/10_implementation"
	adrDir            = "docs/11_decisions"
)

// LoadPackets parses docs/10_implementation/task_queue.md into an id-keyed map.
func LoadPackets(root string) (map[string]gates.TaskPacket, error) {
	packets, _, err := gates.ParseTaskQueue(root)
	if err != nil {
		return nil, err
	}
	m := map[string]gates.TaskPacket{}
	for _, p := range packets {
		m[p.ID] = p
	}
	return m, nil
}

// LoadBlockers parses docs/10_implementation/known_blockers.md.
func LoadBlockers(root string) BlockerDoc {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(knownBlockersPath)))
	if err != nil {
		return BlockerDoc{Open: map[string]BlockerEntry{}, Resolved: map[string]BlockerEntry{}}
	}
	return ParseBlockers(string(data))
}

// TrackedFiles lists repo-relative slash paths of committed files
// (git ls-files), falling back to a filesystem walk outside git worktrees.
func TrackedFiles(root string) ([]string, error) {
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err == nil {
		var files []string
		for _, f := range strings.Split(string(out), "\x00") {
			if f != "" {
				files = append(files, filepath.ToSlash(f))
			}
		}
		return files, nil
	}
	var files []string
	werr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, werr
}

// fileExistsFunc returns whether repo-relative slash paths exist on disk.
func fileExistsFunc(root string) func(string) bool {
	return func(p string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		return err == nil
	}
}
