// Package architecture implements the IMP-083-owned Q4 architecture
// conformance fences (docs/10_implementation/architecture_conformance.md §2–
// §4): Go import direction between the internal layers, the single production
// main, the forbidden dependency/schema list, the generated-code boundary,
// and the client API fence (engineering_conventions.md §2.5, CODE-005 /
// PERF-020) with the protected allowlist client_api_allowlist.txt plus the
// canonical-implementation check of §2.6 (CODE-006).
//
// Rule functions return problem strings; CheckQ4/CheckQ4Client adapt them to
// []gates.Check for the verifier.
package architecture

import (
	"strings"

	"thinhthan/internal/conformance/gates"
)

// CheckQ4 assembles the Go-side Q4 fences as gates.Check values.
func CheckQ4(root string, e *gates.Env) []gates.Check {
	files, err := repoFiles(root)
	if err != nil {
		return []gates.Check{gates.Fail("Q4.arch.file_list", err.Error())}
	}
	id := func(s string) string { return "Q4.arch." + s }
	return []gates.Check{
		statusCheck(id("import_direction"), ImportDirectionProblems(root, files)),
		statusCheck(id("one_production_main"), ProductionMainProblems(root, files)),
		statusCheck(id("forbidden_dependencies"), ForbiddenDependencyProblems(root, files)),
		statusCheck(id("schema_prohibitions"), SchemaProblems(root, files)),
		statusCheck(id("generated_boundary"), GeneratedBoundaryProblems(root, files)),
		statusCheck(id("canonical_implementations"), GoCanonicalProblems(root, files)),
	}
}

// CheckQ4Client assembles the client-side Q4 fences (CODE-005, PERF-020,
// CODE-006) as gates.Check values.
func CheckQ4Client(root string, e *gates.Env) []gates.Check {
	files, err := repoFiles(root)
	if err != nil {
		return []gates.Check{gates.Fail("Q4.client.file_list", err.Error())}
	}
	allow, allowProblems := LoadAllowlist(root)
	var problems []string
	problems = append(problems, allowProblems...)
	problems = append(problems, ClientFenceProblems(root, files, allow)...)
	return []gates.Check{
		statusCheck("Q4.client.api_fence", problems),
		statusCheck("Q4.client.canonical", CanonicalImplementationProblems(root, files)),
	}
}

func statusCheck(id string, problems []string) gates.Check {
	if len(problems) == 0 {
		return gates.Pass(id, "")
	}
	if len(problems) > 8 {
		problems = append(problems[:8], "...")
	}
	return gates.Fail(id, strings.Join(problems, "; "))
}

// repoFiles lists repo-relative slash paths of committed files via
// git ls-files.
func repoFiles(root string) ([]string, error) {
	out, err := gitFiles(root)
	if err != nil {
		return nil, err
	}
	return out, nil
}
