package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"thinhthan/internal/conformance/style"
)

// asmdefAssembly is the layout's assembly list: name -> directory (the 13
// mandatory assemblies of repository_layout.md, owned by IMP-000).
var asmdefDirs = map[string]string{
	"ThinhThan.Core":                     "client/Assets/Scripts/Core",
	"ThinhThan.Net":                      "client/Assets/Scripts/Net",
	"ThinhThan.Systems":                  "client/Assets/Scripts/Systems",
	"ThinhThan.UI":                       "client/Assets/Scripts/UI",
	"ThinhThan.App":                      "client/Assets/Scripts/App",
	"ThinhThan.Protocol":                 "client/Assets/Scripts/Protocol",
	"ThinhThan.Core.Assets":              "client/Assets/Scripts/Core/Assets",
	"ThinhThan.Core.Assets.Editor":       "client/Assets/Scripts/Core/Assets/Editor",
	"ThinhThan.Core.Localization":        "client/Assets/Scripts/Core/Localization",
	"ThinhThan.Core.Localization.Editor": "client/Assets/Scripts/Core/Localization/Editor",
	"ThinhThan.Core.Geometry.Editor":     "client/Assets/Scripts/Core/Geometry/Editor",
	"ThinhThan.Tests.EditMode":           "client/Assets/Tests/EditMode",
	"ThinhThan.Tests.PlayMode":           "client/Assets/Tests/PlayMode",
}

// asmdefRefsRe captures the "references" array body of an asmdef JSON.
var asmdefRefsRe = regexp.MustCompile(`"references"\s*:\s*\[([^\]]*)\]`)
var asmdefStrRe = regexp.MustCompile(`"([^"]+)"`)

// CheckQ4 — Architecture conformance (owner IMP-000 base + IMP-083 client API
// fence / canonical-implementation + IMP-068 ratchet).
func CheckQ4(root string, e *Env) []Check {
	const gate = "Q4"
	id := func(s string) string { return gate + "." + s }
	var checks []Check

	// Production mains: the single server binary plus the tool binaries the
	// spec allows (one binary = thinhthan-server; tools per AGENTS.md).
	allowedMains := map[string]bool{
		"server/cmd/server/main.go":   true,
		"server/cmd/compiler/main.go": true,
		"server/cmd/verify/main.go":   true,
		"server/cmd/migrate/main.go":  true,
	}
	var mains []string
	_ = filepath.WalkDir(filepath.Join(root, "server", "cmd"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "main.go" {
			mains = append(mains, filepath.ToSlash(mustRel(root, p)))
		}
		return nil
	})
	var badMains []string
	for _, m := range mains {
		if !allowedMains[m] {
			badMains = append(badMains, m)
		}
	}
	checks = append(checks, statusCheck(id("one_production_main"), badMains))

	// asmdef acyclicity + mandatory presence + Protocol isolation + the
	// mandatory set is closed (no extra .asmdef files).
	checks = append(checks, checkAsmdefGraph(root)...)

	// C# style (all committed *.cs).
	checks = append(checks, checkCSharpStyle(root)...)

	// .editorconfig + .gitattributes canonical keys.
	checks = append(checks, checkRootConfigFiles(root)...)

	// Generated boundary: server/internal/protocol and
	// client/Assets/Scripts/Protocol hold only generated outputs (no
	// hand-written sources besides the IMP-000 asmdef).
	checks = append(checks, checkGeneratedBoundary(root))

	return checks
}

// checkAsmdefGraph parses every *.asmdef under client/, builds the
// reference graph, asserts acyclic and every mandatory assembly declared.
func checkAsmdefGraph(root string) []Check {
	id := "Q4.asmdefs"
	var problems []string
	refs := map[string][]string{}
	for name, dir := range asmdefDirs {
		path := filepath.Join(root, filepath.FromSlash(dir), name+".asmdef")
		data, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, "missing asmdef "+name+" at "+dir)
			continue
		}
		if m := asmdefRefsRe.FindStringSubmatch(string(data)); m != nil {
			for _, r := range asmdefStrRe.FindAllStringSubmatch(m[1], -1) {
				refs[name] = append(refs[name], r[1])
			}
		}
	}
	// acyclic
	color := map[string]int{}
	var stack []string
	var cycle []string
	var visit func(n string) bool
	visit = func(n string) bool {
		if color[n] == 1 {
			cycle = append(stack, n)
			return true
		}
		if color[n] == 2 {
			return false
		}
		color[n] = 1
		stack = append(stack, n)
		for _, d := range refs[n] {
			if _, known := asmdefDirs[d]; !known {
				problems = append(problems, fmt.Sprintf("%s references unknown assembly %s", n, d))
				continue
			}
			if visit(d) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = 2
		return false
	}
	for n := range refs {
		if visit(n) {
			break
		}
	}
	if len(cycle) > 0 {
		problems = append(problems, "assembly reference cycle: "+strings.Join(cycle, " -> "))
	}
	// Protocol references no project assembly.
	for _, r := range refs["ThinhThan.Protocol"] {
		if strings.HasPrefix(r, "ThinhThan.") && r != "ThinhThan.Protocol" {
			problems = append(problems, "ThinhThan.Protocol references "+r+" (generated code may not depend on project assemblies)")
		}
	}
	// The mandatory set is closed: no .asmdef outside the 13 may exist.
	expected := map[string]bool{}
	for name, dir := range asmdefDirs {
		expected[dir+"/"+name+".asmdef"] = true
	}
	if out, err := gitDir(root, "ls-files", "--", "client/**/*.asmdef"); err == nil {
		for _, rel := range splitLines(out) {
			if !expected[rel] {
				problems = append(problems, "unexpected asmdef "+rel+" (mandatory assembly set is closed)")
			}
		}
	}
	return []Check{statusCheck(id, problems)}
}

// checkCSharpStyle runs the style checker on every tracked first-party .cs
// under client/. Generated files under Assets/Scripts/Protocol/ are exempt —
// they are governed by CODE-004 (generated header + byte-determinism), not
// by the §2.7 hand-written style rules.
func checkCSharpStyle(root string) []Check {
	const id = "Q4.csharp_style"
	out, err := gitDir(root, "ls-files", "--", "client/**/*.cs")
	if err != nil {
		return []Check{Fail(id, "git ls-files: "+err.Error())}
	}
	var problems []string
	for _, rel := range splitLines(out) {
		if strings.HasPrefix(rel, "client/Assets/Scripts/Protocol/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		for _, v := range style.CheckFile(rel, string(data), expectedNamespace(rel)) {
			problems = append(problems, rel+": "+v)
		}
	}
	return []Check{statusCheck(id, problems)}
}

// expectedNamespace computes `ThinhThan.<Assembly>[.<subfolders>]` from the
// file's position under its assembly root (the dir holding its .asmdef).
func expectedNamespace(rel string) string {
	best := ""
	asm := ""
	for name, dir := range asmdefDirs {
		dirSlash := strings.TrimSuffix(dir, "/") + "/"
		if strings.HasPrefix(rel, dirSlash) && len(dirSlash) > len(best) {
			best = dirSlash
			asm = name
		}
	}
	if best == "" {
		return ""
	}
	rest := strings.TrimPrefix(rel, best)
	if idx := strings.LastIndexByte(rest, '/'); idx >= 0 {
		return asm + "." + strings.ReplaceAll(rest[:idx], "/", ".")
	}
	return asm
}

// checkRootConfigFiles asserts canonical .editorconfig/.gitattributes keys.
func checkRootConfigFiles(root string) []Check {
	var problems []string
	ec, err := os.ReadFile(filepath.Join(root, ".editorconfig"))
	if err != nil {
		problems = append(problems, ".editorconfig missing")
	} else {
		for _, want := range []string{
			"charset = utf-8", "end_of_line = lf", "insert_final_newline = true",
			"indent_style = space", "indent_size = 4", "indent_style = tab",
		} {
			if !strings.Contains(string(ec), want) {
				problems = append(problems, ".editorconfig missing `"+want+"`")
			}
		}
	}
	ga, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		problems = append(problems, ".gitattributes missing")
	} else {
		for _, want := range []string{
			"* text=auto eol=lf",
			"filter=lfs diff=lfs merge=lfs -text",
			"merge=unityyamlmerge",
		} {
			if !strings.Contains(string(ga), want) {
				problems = append(problems, ".gitattributes missing `"+want+"`")
			}
		}
		// Unity YAML files are never LFS.
		for _, line := range strings.Split(string(ga), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || strings.HasPrefix(fields[0], "*.") == false {
				continue
			}
			pat := fields[0]
			isLFS := strings.Contains(line, "filter=lfs")
			if isLFS && (pat == "*.unity" || pat == "*.prefab" || pat == "*.asset" || pat == "*.meta" || pat == "*.mat") {
				problems = append(problems, ".gitattributes puts "+pat+" in LFS (must use unityyamlmerge, not LFS)")
			}
		}
	}
	return []Check{statusCheck("Q4.root_config", problems)}
}

// checkGeneratedBoundary: generated dirs contain only generated files. A
// generated C# file must carry the CODE-004 header (`#nullable disable` +
// `#pragma warning disable`); a .cs without it is hand-written and flagged.
func checkGeneratedBoundary(root string) Check {
	const id = "Q4.generated_boundary"
	var problems []string
	for _, dir := range []string{
		"server/internal/protocol",
		"client/Assets/Scripts/Protocol",
	} {
		full := filepath.Join(root, filepath.FromSlash(dir))
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		for _, en := range entries {
			if en.IsDir() {
				continue
			}
			name := en.Name()
			switch {
			case strings.HasSuffix(name, ".asmdef"), strings.HasSuffix(name, ".meta"):
			case strings.HasSuffix(name, ".pb.go"):
			case strings.HasSuffix(name, ".cs"):
				data, err := os.ReadFile(filepath.Join(full, name))
				if err != nil || !hasGeneratedCSharpHeader(string(data)) {
					problems = append(problems, dir+"/"+name+" lacks the CODE-004 generated header — hand-written files are forbidden in generated dirs")
				}
			default:
				problems = append(problems, dir+"/"+name+" unexpected")
			}
		}
	}
	return statusCheck(id, problems)
}

// hasGeneratedCSharpHeader reports whether a .cs file begins with the
// CODE-004 generated header: within the first lines, `#nullable disable`
// and `#pragma warning disable` (an `// <auto-generated>` comment line may
// precede them).
func hasGeneratedCSharpHeader(text string) bool {
	lines := strings.SplitN(text, "\n", 12)
	head := strings.Join(lines[:min(8, len(lines))], "\n")
	first := ""
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			first = t
			break
		}
	}
	if !strings.HasPrefix(first, "#nullable disable") &&
		!strings.HasPrefix(first, "#pragma warning disable") &&
		!strings.HasPrefix(first, "//") {
		return false
	}
	return strings.Contains(head, "#nullable disable") && strings.Contains(head, "#pragma warning disable")
}
