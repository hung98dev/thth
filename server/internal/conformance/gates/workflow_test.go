package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// verifyWf loads the repo verify.yml once per test.
func verifyWf(t *testing.T) *WorkflowFile {
	t.Helper()
	root := repoRoot(t)
	wf, err := ParseWorkflow(filepath.Join(root, ".github", "workflows", "verify.yml"))
	if err != nil {
		t.Fatalf("parse verify.yml: %v", err)
	}
	return wf
}

func jobNamed(t *testing.T, wf *WorkflowFile, name string) *WorkflowJob {
	t.Helper()
	j := wf.JobByName(name)
	if j == nil {
		t.Fatalf("verify.yml: job %q missing", name)
	}
	return j
}

func stepNamed(t *testing.T, j *WorkflowJob, name string) *WorkflowStep {
	t.Helper()
	for i := range j.Steps {
		if j.Steps[i].Name == name {
			return &j.Steps[i]
		}
	}
	t.Fatalf("job %q: step %q missing", j.Name, name)
	return nil
}

func TestLinuxAndWindowsJobsRequired(t *testing.T) {
	wf := verifyWf(t)
	linux := jobNamed(t, wf, "verify-linux")
	win := jobNamed(t, wf, "verify-windows")
	if linux.RunsOn != "ubuntu-24.04" || win.RunsOn != "windows-2022" {
		t.Fatalf("runner labels must be ubuntu-24.04 / windows-2022 (ADR-0058), got %q / %q", linux.RunsOn, win.RunsOn)
	}
	for _, j := range wf.Jobs {
		if strings.HasSuffix(j.RunsOn, "-latest") || j.RunsOn == "" {
			t.Fatalf("job %q uses forbidden or empty runs-on %q", j.Name, j.RunsOn)
		}
	}
}

func TestForkGuardIsFirstStep(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows", "evidence-manifest"} {
		j := jobNamed(t, wf, name)
		if len(j.Steps) == 0 || j.Steps[0].Name != "Fork guard" {
			t.Fatalf("job %q: first step is %q, want Fork guard", name, j.Steps[0].Name)
		}
	}
}

func TestNoJobLevelIfOnRequiredJobs(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		if j.HasJobIf {
			t.Fatalf("required job %q has job-level if: %s", name, j.If)
		}
	}
}

func TestSecretsOnlyAfterForkGuard(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows", "evidence-manifest"} {
		j := jobNamed(t, wf, name)
		seenFork := false
		for _, s := range j.Steps {
			if s.Name == "Fork guard" {
				seenFork = true
				continue
			}
			if !seenFork && strings.Contains(s.Run, "secrets.") {
				t.Fatalf("job %q step %q touches secrets before fork guard", name, s.Name)
			}
		}
	}
}

func TestCliToolsFromPinnedReleaseAssets(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		found := map[string]bool{}
		for _, s := range j.Steps {
			low := strings.ToLower(s.Run)
			if strings.Contains(low, "releases/download") {
				for _, tool := range []string{"powershell", "jq", "gh", "git-lfs"} {
					if strings.Contains(low, tool) {
						found[tool] = true
					}
				}
			}
		}
		for _, tool := range []string{"powershell", "jq", "gh", "git-lfs"} {
			if !found[tool] {
				t.Fatalf("job %q: %s not installed from pinned release assets", name, tool)
			}
		}
	}
	// Every pinned download is hash-verified: linux asserts via sha256sum,
	// windows via the Get-Pinned helper (one Get-FileHash per call).
	if c := strings.Count(wf.Raw, "sha256sum -c"); c < 5 {
		t.Fatalf("linux sha256sum verification missing (%d sites)", c)
	}
	if c := strings.Count(wf.Raw, "Get-Pinned '"); c < 4 {
		t.Fatalf("windows Get-Pinned verification missing (%d calls)", c)
	}
}

func TestLinuxUnityRunsHeadless(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "verify-linux")
	found := false
	for i := range j.Steps {
		if strings.Contains(j.Steps[i].Run, `"/opt/unity/Editor/Unity" -batchmode`) {
			found = true
			if !strings.Contains(j.Steps[i].Run, "-nographics") {
				t.Fatalf("linux unity step %q must run headless (-nographics)", j.Steps[i].Name)
			}
		}
	}
	if !found {
		t.Fatal(`linux job: no "/opt/unity/Editor/Unity" -batchmode step`)
	}
}

func TestVisualReviewArtifactUpload(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "verify-linux")
	s := stepNamed(t, j, "Upload visual-review")
	if !strings.Contains(s.Uses, "upload-artifact") {
		t.Fatal("visual-review step must use actions/upload-artifact")
	}
}

func TestPullRequestTriggerBeforeCutover(t *testing.T) {
	wf := verifyWf(t)
	if !wf.Triggers["pull_request"] {
		t.Fatal("verify.yml must trigger on pull_request")
	}
	for _, tr := range []string{"pull_request_target", "workflow_dispatch", "merge_group", "schedule"} {
		if wf.Triggers[tr] {
			t.Fatalf("verify.yml must not trigger on %s before the IMP-068 cutover", tr)
		}
	}
}

func TestForkGuardOnlyOnPullRequestEvents(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		s := stepNamed(t, jobNamed(t, wf, name), "Fork guard")
		if !strings.Contains(s.If, "pull_request") {
			t.Fatalf("job %q fork guard if must restrict to pull_request events, got %q", name, s.If)
		}
	}
}

func TestForkGuardSkippedOnPush(t *testing.T) {
	wf := verifyWf(t)
	s := stepNamed(t, jobNamed(t, wf, "verify-linux"), "Fork guard")
	if strings.Contains(s.If, "push") {
		t.Fatalf("fork guard must be skipped on push events, got if=%q", s.If)
	}
	if !strings.Contains(s.Run, "external PRs not accepted") {
		t.Fatal("fork guard must fail with 'external PRs not accepted'")
	}
}

func TestFreezeFailsExceptRevertAndOps(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		s := stepNamed(t, jobNamed(t, wf, name), "Merge freeze guard")
		for _, want := range []string{"AUTO_MERGE_FROZEN", "revert/", "ops/"} {
			if !strings.Contains(s.Run, want) {
				t.Fatalf("job %q freeze guard missing %q", name, want)
			}
		}
	}
}

func TestUnityMaterializeRunsWhenUnityGatesSkip(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		s := stepNamed(t, j, "Unity materialization (licence retry <=5)")
		if strings.Contains(s.If, "unity") || strings.Contains(s.If, "gate") {
			t.Fatalf("job %q materialization must run unconditionally (no gate if), got %q", name, s.If)
		}
		// It must come before the verifier step.
		ver := stepNamed(t, j, "Run Q0-Q6 verifier")
		if s.Index >= ver.Index {
			t.Fatalf("job %q: materialization must precede the verifier", name)
		}
	}
}

func TestMaterializedArtifactPerOsFailsJob(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		drift := stepNamed(t, j, "Unity materialized drift check")
		if !strings.Contains(drift.Run, "commit unity-materialized") {
			t.Fatalf("job %q: drift step must fail with 'commit unity-materialized'", name)
		}
		found := false
		for _, s := range j.Steps {
			if strings.Contains(s.Run, "unity-materialized") {
				found = true
			}
		}
		if !found {
			t.Fatalf("job %q: unity-materialized-<os> artifact upload missing", name)
		}
	}
}

func TestLicenceActivationRetriedFiveTimes(t *testing.T) {
	wf := verifyWf(t)
	bashLoop := regexp.MustCompile(`for\s+\w+\s+in\s+([0-9 ]+);`)
	pwshLoop := regexp.MustCompile(`foreach\s*\(\$\w+\s+in\s+1\.\.(\d+)\)`)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		s := stepNamed(t, jobNamed(t, wf, name), "Unity materialization (licence retry <=5)")
		bounds := 0
		for _, m := range bashLoop.FindAllStringSubmatch(s.Run, -1) {
			f := strings.Fields(m[1])
			last, _ := strconv.Atoi(f[len(f)-1])
			if last > 5 {
				t.Fatalf("job %q: retry loop bound %d exceeds licence-retry limit 5", name, last)
			}
			bounds++
		}
		for _, m := range pwshLoop.FindAllStringSubmatch(s.Run, -1) {
			n, _ := strconv.Atoi(m[1])
			if n > 5 {
				t.Fatalf("job %q: retry loop bound %d exceeds licence-retry limit 5", name, n)
			}
			bounds++
		}
		if bounds == 0 {
			t.Fatalf("job %q: licence retry loop not found", name)
		}
		if !strings.Contains(s.Run, "60") {
			t.Fatalf("job %q: licence retry loop must retry 60s apart", name)
		}
	}
}

func TestCheckoutLfsAndPinnedGitLfs(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		found := false
		for _, s := range j.Steps {
			if strings.Contains(s.Uses, "actions/checkout") {
				found = true
			}
		}
		if !found {
			t.Fatalf("job %q: actions/checkout missing", name)
		}
		// checkout uses lfs: true — the raw file carries it next to each uses.
	}
	if strings.Count(wf.Raw, "lfs: true") < 3 {
		t.Fatal("every checkout must set lfs: true")
	}
	if !strings.Contains(wf.Raw, "git-lfs-linux-amd64-v3.8.0") || !strings.Contains(wf.Raw, "git-lfs-windows-amd64-v3.8.0") {
		t.Fatal("pinned git-lfs 3.8.0 release assets missing")
	}
}

func TestRaceOnLinuxJobOnly(t *testing.T) {
	wf := verifyWf(t)
	// The verifier enables -race when RUNNER_OS == Linux; assert the flag
	// never appears on the Windows job.
	win := jobNamed(t, wf, "verify-windows")
	for _, s := range win.Steps {
		if strings.Contains(s.Run, "-race") {
			t.Fatalf("windows job step %q must not enable -race", s.Name)
		}
	}
	// The wiring: cmd/verify gates -race on the runner OS.
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "server", "cmd", "verify", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `RunnerOS == "Linux"`) {
		t.Fatal("verifier must gate -race on RUNNER_OS == Linux")
	}
}

func TestEvidenceJobUsesPinnedDownloadArtifact(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "evidence-manifest")
	found := false
	for _, s := range j.Steps {
		if strings.Contains(s.Uses, "actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdf5c8093") {
			found = true
		}
	}
	if !found {
		t.Fatal("evidence job must download reports via pinned actions/download-artifact v4.3.0")
	}
}
