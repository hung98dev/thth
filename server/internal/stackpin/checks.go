package stackpin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Finding is one Q1 pin-check outcome: ID plus zero or more problems.
// An empty Problems slice means the pin is satisfied.
type Finding struct {
	ID       string
	Problems []string
}

func ok(id string) Finding                      { return Finding{ID: id} }
func bad(id string, problems ...string) Finding { return Finding{ID: id, Problems: problems} }

// Run executes every Q1 pin check against the repository at root.
func Run(root string) []Finding {
	return []Finding{
		checkGoMod(root),
		checkUnity(root),
		checkProtobufCSharp(root),
		checkCscRsp(root),
		checkGitHubActions(root),
		checkRunnerLabels(root),
		checkNoPythonVerify(root),
	}
}

var forbiddenVersionTokens = []string{"latest", "nightly", "preview", "beta", "rc", "HEAD"}

// floatingToken reports a forbidden version token inside a version string.
func floatingToken(v string) string {
	lv := strings.ToLower(v)
	for _, t := range forbiddenVersionTokens {
		if strings.Contains(lv, t) {
			return t
		}
	}
	if strings.Contains(v, "*") || strings.Contains(v, ">=") || strings.HasSuffix(v, ".x") {
		return "floating range"
	}
	return ""
}

var goDirectiveRe = regexp.MustCompile(`^go\s+([0-9.]+)\s*$`)
var requireRe = regexp.MustCompile(`^([^\s]+)\s+(v[0-9][^\s]*)$`)

func checkGoMod(root string) Finding {
	const id = "Q1.go_mod"
	path := filepath.Join(root, "server", "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return bad(id, "server/go.mod missing")
	}
	var problems []string
	var goDir, moduleName string
	inRequire := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "module ") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
		if m := goDirectiveRe.FindStringSubmatch(line); m != nil {
			goDir = m[1]
		}
		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		var mod, ver string
		switch {
		case inRequire:
			if m := requireRe.FindStringSubmatch(line); m != nil {
				mod, ver = m[1], m[2]
			}
		case strings.HasPrefix(line, "require "):
			if m := requireRe.FindStringSubmatch(strings.TrimPrefix(line, "require ")); m != nil {
				mod, ver = m[1], m[2]
			}
		}
		if mod == "" {
			continue
		}
		for _, f := range ForbiddenModules {
			if mod == f || strings.HasPrefix(mod, f+"/") {
				problems = append(problems, "forbidden dependency "+mod)
			}
		}
		if want, ok := GoModulePins[mod]; !ok {
			problems = append(problems, fmt.Sprintf("unlisted module %s %s", mod, ver))
		} else if want != ver {
			problems = append(problems, fmt.Sprintf("%s pinned %s, go.mod has %s", mod, want, ver))
		}
	}
	if moduleName != "thinhthan" {
		problems = append(problems, "module name must be thinhthan, got "+moduleName)
	}
	if goDir != GoVersion {
		problems = append(problems, fmt.Sprintf("go directive %q != %s", goDir, GoVersion))
	}
	if _, err := os.Stat(filepath.Join(root, "server", "go.sum")); err != nil {
		problems = append(problems, "server/go.sum missing — committed lockfile required")
	}
	if len(problems) > 0 {
		return bad(id, problems...)
	}
	return ok(id)
}

type unityManifest struct {
	Dependencies map[string]string `json:"dependencies"`
}

type unityLock struct {
	Dependencies map[string]struct {
		Version string `json:"version"`
	} `json:"dependencies"`
}

func checkUnity(root string) Finding {
	const id = "Q1.unity"
	var problems []string

	pv, err := os.ReadFile(filepath.Join(root, "client", "ProjectSettings", "ProjectVersion.txt"))
	if err != nil {
		problems = append(problems, "ProjectVersion.txt missing")
	} else {
		s := string(pv)
		if !strings.Contains(s, "m_EditorVersion: "+UnityEditor) {
			problems = append(problems, "ProjectVersion.txt does not pin "+UnityEditor)
		}
		if !strings.Contains(s, "m_EditorVersionWithRevision: "+UnityEditor+" (7efac9f6c10e)") {
			problems = append(problems, "ProjectVersion.txt missing revision 6000.6.1f1 (7efac9f6c10e)")
		}
	}

	manifestPath := filepath.Join(root, "client", "Packages", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		problems = append(problems, "manifest.json missing")
	} else {
		var m unityManifest
		if json.Unmarshal(data, &m) != nil {
			problems = append(problems, "manifest.json invalid JSON")
		} else {
			for name, want := range UnityPackages {
				got, ok := m.Dependencies[name]
				if !ok {
					problems = append(problems, name+" missing from manifest.json")
				} else if got != want {
					problems = append(problems, fmt.Sprintf("%s pinned %s, manifest has %s", name, want, got))
				}
			}
			for name, ver := range m.Dependencies {
				if strings.HasPrefix(name, "com.unity.modules.") {
					if ver != "1.0.0" {
						problems = append(problems, name+" module version must be 1.0.0")
					}
					continue
				}
				if _, ok := UnityPackages[name]; !ok {
					problems = append(problems, "unlisted Unity package "+name+" "+ver)
				}
				if t := floatingToken(ver); t != "" {
					problems = append(problems, fmt.Sprintf("%s uses forbidden version token %q", name, t))
				}
			}
		}
	}

	lockPath := filepath.Join(root, "client", "Packages", "packages-lock.json")
	if data, err := os.ReadFile(lockPath); err == nil {
		var l unityLock
		if json.Unmarshal(data, &l) == nil {
			for name, want := range UnityPackages {
				if e, ok := l.Dependencies[name]; ok && e.Version != want {
					problems = append(problems, fmt.Sprintf("%s lock %s != pin %s", name, e.Version, want))
				}
			}
		}
	}

	if len(problems) > 0 {
		return bad(id, problems...)
	}
	return ok(id)
}

func checkProtobufCSharp(root string) Finding {
	const id = "Q1.protobuf_csharp"
	dir := filepath.Join(root, "client", "Assets", "Plugins", "Google.Protobuf")
	dll := filepath.Join(dir, "Google.Protobuf.dll")
	if _, err := os.Stat(dll); err != nil {
		return bad(id, "vendored Google.Protobuf.dll missing (from verified nupkg lib/netstandard2.0)")
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || !strings.Contains(string(readme), ProtobufCSharp) {
		return bad(id, "plugin README does not record pin "+ProtobufCSharp)
	}
	return ok(id)
}

// checkCscRsp enforces CODE-001: client/Assets/csc.rsp is exactly
// -warnaserror+ and -nullable:enable.
func checkCscRsp(root string) Finding {
	const id = "Q1.csc_rsp"
	data, err := os.ReadFile(filepath.Join(root, "client", "Assets", "csc.rsp"))
	if err != nil {
		return bad(id, "client/Assets/csc.rsp missing")
	}
	var flags []string
	for _, line := range strings.Split(string(data), "\n") {
		if t := strings.TrimSpace(line); t != "" {
			flags = append(flags, t)
		}
	}
	want := []string{"-warnaserror+", "-nullable:enable"}
	if strings.Join(flags, " ") != strings.Join(want, " ") {
		return bad(id, fmt.Sprintf("csc.rsp = %q, want %q", strings.Join(flags, " "), strings.Join(want, " ")))
	}
	return ok(id)
}

var usesRe = regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*([^\s@#]+)@([^\s#]+)`)
var runsOnRe = regexp.MustCompile(`(?m)^\s*runs-on:\s*([^\s#]+)`)

func checkGitHubActions(root string) Finding {
	const id = "Q1.github_actions"
	wfDir := filepath.Join(root, ".github", "workflows")
	var entries []string
	for _, ext := range []string{"*.yml", "*.yaml"} {
		entries = append(entries, globOrNil(filepath.Join(wfDir, ext))...)
	}
	if len(entries) == 0 {
		return bad(id, "no workflow files under .github/workflows/")
	}
	var problems []string
	for _, wf := range entries {
		data, err := os.ReadFile(wf)
		if err != nil {
			continue
		}
		base := filepath.Base(wf)
		for _, m := range usesRe.FindAllStringSubmatch(string(data), -1) {
			action, ref := m[1], m[2]
			pin, ok := GitHubActions[action]
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: unlisted action %s@%s", base, action, ref))
				continue
			}
			if ref != pin.SHA {
				problems = append(problems, fmt.Sprintf("%s: %s must pin SHA %s (%s), got %s", base, action, pin.SHA, pin.Tag, ref))
			}
		}
	}
	if len(problems) > 0 {
		return bad(id, problems...)
	}
	return ok(id)
}

func checkRunnerLabels(root string) Finding {
	const id = "Q1.runner_labels"
	wfDir := filepath.Join(root, ".github", "workflows")
	var entries []string
	for _, ext := range []string{"*.yml", "*.yaml"} {
		entries = append(entries, globOrNil(filepath.Join(wfDir, ext))...)
	}
	var problems []string
	for _, wf := range entries {
		data, err := os.ReadFile(wf)
		if err != nil {
			continue
		}
		for _, m := range runsOnRe.FindAllStringSubmatch(string(data), -1) {
			label := strings.Trim(strings.TrimSpace(m[1]), `"'`)
			if !RunnerLabels[label] {
				problems = append(problems, fmt.Sprintf("%s: forbidden runs-on %q (ADR-0058)", filepath.Base(wf), label))
			}
		}
	}
	if len(problems) > 0 {
		return bad(id, problems...)
	}
	return ok(id)
}

// checkNoPythonVerify enforces the matrix rule that scripts/verify.ps1 must
// not call python.
func checkNoPythonVerify(root string) Finding {
	const id = "Q1.verify_ps1_no_python"
	data, err := os.ReadFile(filepath.Join(root, "scripts", "verify.ps1"))
	if err != nil {
		return bad(id, "scripts/verify.ps1 missing")
	}
	for i, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if strings.Contains(t, "python") {
			return bad(id, fmt.Sprintf("verify.ps1 line %d calls python: %s", i+1, t))
		}
	}
	return ok(id)
}

// SortedIDs lists finding IDs deterministically (test helper).
func SortedIDs(findings []Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
	}
	sort.Strings(ids)
	return ids
}

func globOrNil(pattern string) []string {
	m, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return m
}
