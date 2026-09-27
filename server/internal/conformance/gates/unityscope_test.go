package gates

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnityRelevantPaths(t *testing.T) {
	cases := []struct {
		file string
		want bool
	}{
		{"client/Assets/Scripts/Core/X.cs", true},
		{"proto/thinhthan/v1/common.proto", true},
		{"scripts/codegen.ps1", true},
		{"scripts/verify.ps1", true},
		{".github/workflows/verify.yml", true},
		{"server/cmd/verify/main.go", true},
		{"server/internal/conformance/gates/q3.go", true},
		{"docs/00_context/technology_versions.md", true},
		{"server/internal/sim/tick.go", false},
		{"server/internal/core/id/id.go", false},
		{"docs/10_implementation/task_queue.md", false},
		{"docs/01_gameplay/combat.md", false},
		{"scripts/verify.ps1.bak", false},
	}
	for _, c := range cases {
		if got := UnityRelevant([]string{c.file}); got != c.want {
			t.Errorf("UnityRelevant(%q) = %v, want %v", c.file, got, c.want)
		}
	}
}

func TestResolveUnityScope(t *testing.T) {
	pr := &Env{EventName: "pull_request", HeadRef: "imp/IMP-007-currency"}
	server := []string{"server/internal/durable/x.go"}

	if skip, p := ResolveUnityScope(pr, UnityScopeFull, server, nil); skip || p != "" {
		t.Fatalf("full scope: skip=%v problem=%q", skip, p)
	}
	if skip, p := ResolveUnityScope(pr, UnityScopeNoClientChange, server, nil); !skip || p != "" {
		t.Fatalf("server-only PR must skip: skip=%v problem=%q", skip, p)
	}
	if skip, p := ResolveUnityScope(pr, UnityScopeNoClientChange, []string{"client/a.cs"}, nil); skip || p == "" {
		t.Fatal("skip declared on a client diff must be a problem")
	}
	if skip, p := ResolveUnityScope(pr, UnityScopeNoClientChange, nil, errors.New("git")); skip || p == "" {
		t.Fatal("unverifiable diff must be a problem")
	}
	push := &Env{EventName: "push", HeadRef: ""}
	if skip, p := ResolveUnityScope(push, UnityScopeNoClientChange, server, nil); skip || p == "" {
		t.Fatal("no-client-change must never be valid on push")
	}
	for _, b := range []string{"imp/IMP-068-trusted-ci", "imp/IMP-005-done"} {
		e := &Env{EventName: "pull_request", HeadRef: b}
		if skip, p := ResolveUnityScope(e, UnityScopeNoClientChange, server, nil); skip || p == "" {
			t.Fatalf("branch %s must run Unity in full", b)
		}
	}
}

// verify.yml's Unity scope step must carry UnityRelevantPattern verbatim on
// both jobs, and gate every Unity step on its output (ADR-0073).
func TestWorkflowUnityScopeMatchesVerifier(t *testing.T) {
	wf, err := ParseWorkflow(filepath.Join(repoRoot(t), ".github", "workflows", "verify.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := wf.JobByName(name)
		if j == nil {
			t.Fatalf("job %s missing", name)
		}
		var scope *WorkflowStep
		for i := range j.Steps {
			if j.Steps[i].Name == "Unity scope" {
				scope = &j.Steps[i]
			}
		}
		if scope == nil {
			t.Fatalf("%s: Unity scope step missing", name)
		}
		if !strings.Contains(scope.Run, UnityRelevantPattern) {
			t.Fatalf("%s: Unity scope pattern differs from gates.UnityRelevantPattern", name)
		}
		for _, need := range []string{"THINHTHAN_UNITY_SCOPE", "no-client-change", "imp/IMP-068-", "-done"} {
			if !strings.Contains(scope.Run, need) {
				t.Fatalf("%s: Unity scope step lacks %q", name, need)
			}
		}
		for _, s := range j.Steps {
			if strings.HasPrefix(s.Name, "Unity materialization") || strings.HasPrefix(s.Name, "Unity EditMode") {
				if !strings.Contains(s.If, "steps.unity-scope.outputs.run == 'true'") {
					t.Fatalf("%s/%s: must be gated on the Unity scope output, got if=%q", name, s.Name, s.If)
				}
				if s.Index < scope.Index {
					t.Fatalf("%s/%s runs before the Unity scope step", name, s.Name)
				}
			}
		}
	}
}
