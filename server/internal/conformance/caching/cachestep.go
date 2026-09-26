// Package caching audits the actions/cache wiring in verify.yml (IMP-106,
// CI-001..004) and merges per-step cache telemetry into verify-report.json.
//
// gates.ParseWorkflow does not capture with: blocks, so this package parses
// the verify.yml text with a small indentation-aware scanner — no YAML
// dependency is added for this. The grammar is deliberately narrow: jobs,
// step boundaries, and the with: fields the cache policy constrains.
package caching

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Step is one parsed step of a verify.yml job.
type Step struct {
	Job         string   // job id, e.g. "verify-linux"
	Name        string   // - name: ...
	ID          string   // id: ...
	Uses        string   // uses: <action@ref>, trailing comment stripped
	If          string   // if: expression
	Path        []string // with:path (inline value or block list)
	Key         string   // with:key
	RestoreKeys []string // with:restore-keys block list
}

// CacheSteps returns the steps using actions/cache.
func CacheSteps(steps []Step) []Step {
	var out []Step
	for _, s := range steps {
		if strings.HasPrefix(s.Uses, "actions/cache@") {
			out = append(out, s)
		}
	}
	return out
}

var (
	jobRe       = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	stepStartRe = regexp.MustCompile(`^      -\s+(\S.*)$`)
	fieldRe     = regexp.MustCompile(`^        ([A-Za-z_-]+):\s*(.*)$`)
	withFieldRe = regexp.MustCompile(`^          ([A-Za-z_-]+):\s*(.*)$`)
	withItemRe  = regexp.MustCompile(`^            +\S`)
)

// stripComment removes a trailing " # ..." YAML comment and trims space.
func stripComment(v string) string {
	if i := strings.Index(v, " #"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

func isBlockScalar(v string) bool {
	return v == "|" || v == "|-" || v == "|+" || v == ">" || v == ">-" || v == ">+"
}

// ParseSteps parses every step of every job in the workflow file at path.
func ParseSteps(path string) ([]Step, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var steps []Step
	var cur *Step
	job := ""
	inWith := false
	blockField := "" // current multi-line with: field ("path" | "restore-keys")

	flush := func() {
		if cur != nil {
			steps = append(steps, *cur)
			cur = nil
		}
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if m := jobRe.FindStringSubmatch(line); m != nil {
			flush()
			job, inWith, blockField = m[1], false, ""
			continue
		}
		if m := stepStartRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Step{Job: job}
			inWith, blockField = false, ""
			// The step header itself may carry a field: `- name: X`.
			h := m[1]
			if strings.HasPrefix(h, "name:") {
				cur.Name = stripComment(strings.TrimPrefix(h, "name:"))
			} else if strings.HasPrefix(h, "uses:") {
				cur.Uses = stripComment(strings.TrimPrefix(h, "uses:"))
			} else if strings.HasPrefix(h, "id:") {
				cur.ID = stripComment(strings.TrimPrefix(h, "id:"))
			}
			continue
		}
		if cur == nil {
			continue
		}
		if m := fieldRe.FindStringSubmatch(line); m != nil {
			blockField = ""
			key, val := m[1], m[2]
			switch key {
			case "id":
				cur.ID = stripComment(val)
			case "if":
				cur.If = strings.TrimSpace(val)
			case "uses":
				cur.Uses = stripComment(val)
			case "name":
				if cur.Name == "" {
					cur.Name = stripComment(val)
				}
			case "with":
				inWith = true
			default:
				inWith = false // run:, env:, shell:, timeout-minutes:, ...
			}
			continue
		}
		if !inWith {
			continue
		}
		if m := withFieldRe.FindStringSubmatch(line); m != nil {
			key, val := m[1], m[2]
			blockField = ""
			switch key {
			case "path":
				if isBlockScalar(val) {
					blockField = "path"
				} else {
					cur.Path = append(cur.Path, stripComment(val))
				}
			case "key":
				cur.Key = stripComment(val)
			case "restore-keys":
				if isBlockScalar(val) {
					blockField = "restore-keys"
				} else if v := stripComment(val); v != "" {
					cur.RestoreKeys = append(cur.RestoreKeys, v)
				}
			}
			continue
		}
		if blockField != "" && withItemRe.MatchString(line) {
			item := stripComment(strings.TrimSpace(line))
			if item != "" {
				switch blockField {
				case "path":
					cur.Path = append(cur.Path, item)
				case "restore-keys":
					cur.RestoreKeys = append(cur.RestoreKeys, item)
				}
			}
			continue
		}
		// Anything else (deeper indent, unrelated) leaves with: context only
		// when a line at step-field depth arrives — handled above.
	}
	flush()
	return steps, nil
}

// PinSegment returns the fragment of a cache key that pins the toolchain or
// payload version: a ${{ env.<NAME> }} reference or an embedded 64-hex digest.
// Restore-keys must keep this segment verbatim.
var envRefRe = regexp.MustCompile(`\$\{\{\s*env\.([A-Z0-9_]+)\s*\}\}`)
var hex64Re = regexp.MustCompile(`[0-9a-f]{64}`)

// PinTokens returns every pin-ish token in key: env refs and embedded digests.
func PinTokens(key string) []string {
	var out []string
	out = append(out, envRefRe.FindAllString(key, -1)...)
	out = append(out, hex64Re.FindAllString(key, -1)...)
	return out
}

// WorkflowPath resolves <root>/.github/workflows/verify.yml.
func WorkflowPath(root string) string {
	return filepath.Join(root, ".github", "workflows", "verify.yml")
}
