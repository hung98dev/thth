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
	for _, name := range []string{"verify-linux", "verify-windows", "unity-linux", "unity-windows"} {
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
	for _, name := range []string{"verify-linux", "verify-windows", "unity-linux", "unity-windows"} {
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
	j := jobNamed(t, wf, "unity-linux")
	found := false
	for i := range j.Steps {
		run := j.Steps[i].Run
		if !strings.Contains(run, "-projectPath") {
			continue
		}
		// unity-editor wraps Unity under xvfb (virtual display) - headless.
		// A direct editor binary invocation must pass -nographics instead.
		if strings.Contains(run, "unity-editor") {
			found = true
			continue
		}
		if strings.Contains(run, "/opt/unity/Editor/Unity") {
			if !strings.Contains(run, "-nographics") {
				t.Fatalf("linux unity step %q invokes the editor without -nographics", j.Steps[i].Name)
			}
			found = true
		}
	}
	if !found {
		t.Fatal(`linux job: no headless unity-editor/-nographics -projectPath step`)
	}
}

func TestVisualReviewArtifactUpload(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "unity-linux")
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
	for unity, name := range map[string]string{"unity-linux": "verify-linux", "unity-windows": "verify-windows"} {
		stepName := "Unity materialization (licence retry <=5)"
		if unity == "unity-windows" {
			// ADR-0077: Windows materialization is a detached background
			// process started by the launcher step; the join is the gate.
			stepName = "Start Unity materialization (background)"
		}
		s := stepNamed(t, jobNamed(t, wf, unity), stepName)
		// Never gated on gate activation (owner-not-done): its only condition
		// is the ADR-0073 path scope, validated by the verifier.
		if s.If != "steps.unity-scope.outputs.run == 'true'" {
			t.Fatalf("job %q materialization may only be conditioned on the Unity scope, got %q", name, s.If)
		}
		// ADR-0075: the required job joins the Unity job before its verifier.
		j := jobNamed(t, wf, name)
		wait := stepNamed(t, j, "Wait for Unity job")
		ver := stepNamed(t, j, "Run Q0-Q6 verifier")
		if wait.Index >= ver.Index {
			t.Fatalf("job %q: the Unity job must be joined before the verifier", name)
		}
	}
}

func TestMaterializedArtifactPerOsFailsJob(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"unity-linux", "unity-windows"} {
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

// BLK-018: the drift gate may reset only the derived Addressables
// m_currentHash line, and must do so before drift is computed.
func TestDriftGateResetsOnlyAddressablesDerivedHash(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"unity-linux", "unity-windows"} {
		run := stepNamed(t, jobNamed(t, wf, name), "Unity materialized drift check").Run
		for _, want := range []string{
			"'client/Assets/AddressableAssetsData/AddressableAssetSettings.asset'",
			"'^[+-]    Hash: [0-9a-f]{32}$'",
			"checkout -- $aas",
		} {
			if !strings.Contains(run, want) {
				t.Fatalf("job %q: drift step missing %q", name, want)
			}
		}
		if strings.Count(run, "checkout --") != 1 {
			t.Fatalf("job %q: drift step may restore exactly one file", name)
		}
		if strings.Index(run, "checkout -- $aas") > strings.Index(run, "status --porcelain") {
			t.Fatalf("job %q: derived-hash reset must precede the drift computation", name)
		}
	}
}

func TestLicenceActivationRetriedFiveTimes(t *testing.T) {
	wf := verifyWf(t)
	bashLoop := regexp.MustCompile(`for\s+\w+\s+in\s+([0-9 ]+);`)
	pwshLoop := regexp.MustCompile(`foreach\s*\(\$\w+\s+in\s+1\.\.(\d+)\)`)
	for _, name := range []string{"unity-linux", "unity-windows"} {
		var src string
		if name == "unity-windows" {
			// ADR-0077: the Windows retry loop is inside the detached
			// materialization script the launcher runs.
			data, err := os.ReadFile(filepath.Join(repoRoot(t), ".devin/scripts/unity_materialize.ps1"))
			if err != nil {
				t.Fatalf("read unity_materialize.ps1: %v", err)
			}
			src = string(data)
		} else {
			src = stepNamed(t, jobNamed(t, wf, name), "Unity materialization (licence retry <=5)").Run
		}
		s := &WorkflowStep{Run: src}
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
	for _, name := range []string{"verify-linux", "verify-windows", "unity-linux", "unity-windows"} {
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
	if strings.Count(wf.Raw, "lfs: true") < 4 {
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
	for _, name := range []string{"verify-windows", "unity-windows"} {
		for _, s := range jobNamed(t, wf, name).Steps {
			if strings.Contains(s.Run, "-race") {
				t.Fatalf("windows job %q step %q must not enable -race", name, s.Name)
			}
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
	j := jobNamed(t, wf, "verify-linux")
	found := false
	for _, s := range j.Steps {
		if strings.Contains(s.Uses, "actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdf5c8093") {
			found = true
		}
	}
	if !found {
		t.Fatal("the Linux job must download the Windows report via pinned actions/download-artifact v4.3.0")
	}
}

// BLK-017: the verifier runs whenever the tree was checked out, so a failed
// Unity step yields a report with a Q3 FAIL instead of a missing
// verify-report.json that cascades into the evidence job.
func TestVerifierRunsAfterEarlierStepFailure(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"verify-linux", "verify-windows"} {
		j := jobNamed(t, wf, name)
		ver := stepNamed(t, j, "Run Q0-Q6 verifier")
		if !strings.Contains(ver.If, "!cancelled()") || !strings.Contains(ver.If, "steps.checkout.outcome == 'success'") {
			t.Fatalf("job %q: verifier must run after earlier failures once checkout succeeded, got if=%q", name, ver.If)
		}
	}
}

// ADR-0075: Unity runs in a parallel job per OS; the required job joins it
// (wait -> download results -> verifier -> Unity job result) so a failed Unity
// job fails the required check even when every Unity gate is skipped.
func TestRequiredJobsJoinUnityJobs(t *testing.T) {
	wf := verifyWf(t)
	for req, unity := range map[string]string{"verify-linux": "Unity (Linux)", "verify-windows": "Unity (Windows)"} {
		j := jobNamed(t, wf, req)
		if j.HasJobIf {
			t.Fatalf("required job %q must not have a job-level if", req)
		}
		wait := stepNamed(t, j, "Wait for Unity job")
		if !strings.Contains(wf.Raw, "WAIT_JOB: '"+unity+"'") || !strings.Contains(wait.Run, "wait_job.sh") {
			t.Fatalf("job %q must wait for %q via wait_job.sh", req, unity)
		}
		dl := stepNamed(t, j, "Download Unity test results")
		ver := stepNamed(t, j, "Run Q0-Q6 verifier")
		res := stepNamed(t, j, "Unity job result")
		if !(wait.Index < dl.Index && dl.Index < ver.Index && ver.Index < res.Index) {
			t.Fatalf("job %q: order must be wait < download < verifier < Unity job result", req)
		}
		if !strings.Contains(res.Run, `"$CONCLUSION" != success`) || !strings.Contains(res.If, "!cancelled()") {
			t.Fatalf("job %q: Unity job result must fail on any non-success conclusion", req)
		}
	}
}

// ADR-0075: the Unity job launches only the modes the verifier plans.
func TestUnityTestModesFromVerifierPlan(t *testing.T) {
	wf := verifyWf(t)
	for _, name := range []string{"unity-linux", "unity-windows"} {
		j := jobNamed(t, wf, name)
		plan := stepNamed(t, j, "Unity mode plan")
		if !strings.Contains(plan.Run, "./cmd/verify -plan-unity") {
			t.Fatalf("job %q: the mode plan must come from the verifier", name)
		}
		tests := stepNamed(t, j, "Unity EditMode+PlayMode tests")
		if plan.Index >= tests.Index || !strings.Contains(tests.If, "steps.unity-plan.outputs.modes != ''") {
			t.Fatalf("job %q: tests must follow the plan and run only when modes are planned", name)
		}
		if !strings.Contains(tests.Run, "UNITY_MODES") || strings.Contains(tests.Run, "EditMode PlayMode;") || strings.Contains(tests.Run, "'EditMode', 'PlayMode'") {
			t.Fatalf("job %q: tests must iterate the planned modes, not a fixed list", name)
		}
	}
}

// ADR-0075: no separate evidence job; the Linux required job merges both
// reports into the `evidence` artifact.
func TestEvidenceBuiltInLinuxRequiredJob(t *testing.T) {
	wf := verifyWf(t)
	for _, j := range wf.Jobs {
		if j.Name == "evidence-manifest" {
			t.Fatal("evidence-manifest job must not exist (ADR-0075)")
		}
	}
	j := jobNamed(t, wf, "verify-linux")
	build := stepNamed(t, j, "Build evidence manifest")
	up := stepNamed(t, j, "Upload evidence manifest")
	if !strings.Contains(build.Run, "-MergeReports") || build.Index >= up.Index {
		t.Fatal("verify-linux must merge the reports before uploading the evidence artifact")
	}
	if !strings.Contains(wf.Raw, "WAIT_JOB: 'Q0-Q6 verify (Windows)'") {
		t.Fatal("verify-linux must wait for the Windows report before merging")
	}
}

// stepBlock returns the raw YAML of the step named name inside job (keys the
// structural parser does not model, e.g. continue-on-error, env).
func stepBlock(t *testing.T, wf *WorkflowFile, job, name string) string {
	t.Helper()
	loc := regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(job) + `:\s*$`).FindStringIndex(wf.Raw)
	if loc == nil {
		t.Fatalf("job %q not found", job)
	}
	rest := wf.Raw[loc[1]:]
	if next := regexp.MustCompile(`(?m)^  [A-Za-z0-9_-]+:\s*$`).FindStringIndex(rest); next != nil {
		rest = rest[:next[0]]
	}
	start := strings.Index(rest, "      - name: "+name+"\n")
	if start < 0 {
		t.Fatalf("job %q: step %q not found", job, name)
	}
	block := rest[start+1:]
	if end := strings.Index(block, "\n      - name:"); end >= 0 {
		block = block[:end]
	}
	return block
}

// ADR-0077: the digest-pinned Linux image pull runs in the background from
// right after the preconditions and is joined before the first editor run,
// with a foreground pull of the same digest as the fallback.
func TestUnityImagePullOverlapsSetup(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "unity-linux")
	freeze := stepNamed(t, j, "Merge freeze guard")
	start := stepNamed(t, j, "Start Unity image pull (background)")
	if start.Index != freeze.Index+1 || start.If != "" {
		t.Fatal("the background pull must start unconditionally right after the freeze guard")
	}
	block := stepBlock(t, wf, "unity-linux", "Start Unity image pull (background)")
	if !strings.Contains(start.Run, "nohup") || !strings.Contains(start.Run, `docker pull "$1"`) || strings.Contains(block, "secrets.") {
		t.Fatal("background pull must nohup a docker pull of the pinned image without secrets")
	}
	join := stepNamed(t, j, "Pull Unity image (pinned)")
	lib := stepNamed(t, j, "Cache Unity client/Library (pinned)")
	mat := stepNamed(t, j, "Unity materialization (licence retry <=5)")
	if !(lib.Index < join.Index && join.Index < mat.Index) {
		t.Fatal("the Library restore must overlap the pull; the join must precede materialization")
	}
	for _, want := range []string{`kill -0 "$pid"`, `docker pull "$UNITY_IMAGE"`, "docker image inspect"} {
		if !strings.Contains(join.Run, want) {
			t.Fatalf("pull join step missing %q", want)
		}
	}
}

// ADR-0077: the Windows materialization runs detached while the foreground
// installs Go, restores the Go cache and computes the mode plan; the join is
// the only consumer-side wait and must precede -runTests.
func TestWindowsMaterializationOverlapsSetup(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "unity-windows")
	lib := stepNamed(t, j, "Cache Unity client/Library (pinned)")
	editor := stepNamed(t, j, "Install Unity editor (pinned)")
	start := stepNamed(t, j, "Start Unity materialization (background)")
	if !(editor.Index < lib.Index && lib.Index < start.Index) {
		t.Fatal("materialization needs the installed editor and the restored Library before it starts")
	}
	if start.If != "steps.unity-scope.outputs.run == 'true'" {
		t.Fatalf("background materialization may only be conditioned on the Unity scope, got %q", start.If)
	}
	goInstall := stepNamed(t, j, "Install Go (pinned)")
	goCache := stepNamed(t, j, "Cache Go modules + build (pinned)")
	plan := stepNamed(t, j, "Unity mode plan")
	join := stepNamed(t, j, "Join Unity materialization")
	tests := stepNamed(t, j, "Unity EditMode+PlayMode tests")
	if !(start.Index < goInstall.Index && goInstall.Index < goCache.Index && goCache.Index < plan.Index && plan.Index < join.Index && join.Index < tests.Index) {
		t.Fatal("want order: start-bg < Install Go < Cache Go < mode plan < join < tests")
	}
	for _, want := range []string{"Start-Process", "unity_materialize.ps1", "pid.txt", "secrets.UNITY_SERIAL"} {
		if !strings.Contains(stepBlock(t, wf, "unity-windows", "Start Unity materialization (background)"), want) {
			t.Fatalf("background launcher missing %q", want)
		}
	}
	for _, want := range []string{"exitcode.txt", "unity_materialize.ps1", "$rc -ne 0"} {
		if !strings.Contains(join.Run, want) {
			t.Fatalf("join step missing %q", want)
		}
	}
	if join.If != "steps.unity-scope.outputs.run == 'true'" {
		t.Fatalf("join may only be conditioned on the Unity scope, got %q", join.If)
	}
}

// ADR-0077: the kill-probe (auditd/tracefs/bpftrace + editor version probes)
// is an investigation tool, not a gate: it runs only when the repository
// variable UNITY_KILL_PROBE is 'true'.
func TestKillProbeOptIn(t *testing.T) {
	wf := verifyWf(t)
	j := jobNamed(t, wf, "unity-linux")
	for _, name := range []string{"Arm kill-probe (SIGKILL sender diagnostic)", "Dump kill-probe diagnostics"} {
		if s := stepNamed(t, j, name); !strings.Contains(s.If, "vars.UNITY_KILL_PROBE == 'true'") {
			t.Fatalf("step %q must be opt-in via vars.UNITY_KILL_PROBE, got if=%q", name, s.If)
		}
	}
	run := stepNamed(t, j, "Unity materialization (licence retry <=5)").Run
	guard := strings.Index(run, `if [ "${UNITY_KILL_PROBE:-}" = "true" ]; then`)
	probe := strings.Index(run, "unity-editor -version")
	loop := strings.Index(run, "for i in 1 2 3 4 5; do")
	if guard < 0 || probe < guard || loop < probe || strings.Count(run, "direct-version probe") != 1 {
		t.Fatal("editor version probes must run only inside the UNITY_KILL_PROBE guard")
	}
}

// ADR-0077: materialization snapshots the cache-restored Library; a retry
// after an editor kill restarts warm from it (never from the killed editor's
// Library, BLK-017), compiler errors stop when they repeat, and only other
// (licence/infra) failures wait 60 s.
func TestMaterializationWarmRetry(t *testing.T) {
	wf := verifyWf(t)
	run := stepNamed(t, jobNamed(t, wf, "unity-linux"), "Unity materialization (licence retry <=5)").Run
	snap := strings.Index(run, `sudo cp -a client/Library "$mat_snap"`)
	loop := strings.Index(run, "for i in 1 2 3 4 5; do")
	if snap < 0 || snap > loop {
		t.Fatal("the restored Library must be snapshotted before the first editor run")
	}
	for _, want := range []string{
		`if [ "$prev" = killed ] && [ "$i" -le 3 ] && [ -d "$mat_snap" ]; then`,
		`sudo rm -rf client/Library client/Temp`,
		`sudo cp -a "$mat_snap" client/Library`,
		`elif [ "$rc" -eq 137 ]; then`,
		"compiler errors on two consecutive attempts",
		`[ "$i" -lt 5 ] && [ "$prev" = other ] && sleep 60`,
	} {
		if !strings.Contains(run, want) {
			t.Fatalf("linux materialization missing %q", want)
		}
	}
	winData, err := os.ReadFile(filepath.Join(repoRoot(t), ".devin/scripts/unity_materialize.ps1"))
	if err != nil {
		t.Fatalf("read unity_materialize.ps1: %v", err)
	}
	win := string(winData)
	if !strings.Contains(win, "$compile -and $prevCompile") || !strings.Contains(win, "-not $compile") {
		t.Fatal("windows materialization must stop on repeated compiler errors and skip the 60 s wait for them")
	}
}

// ADR-0077: a Linux -runTests attempt that completed with a non-Passed
// results XML is a final verdict (no retry); retries are reserved for editor
// kills. The step still restores the clean Library snapshot before failing.
func TestLinuxTestFailedVerdictIsFinal(t *testing.T) {
	wf := verifyWf(t)
	run := stepNamed(t, jobNamed(t, wf, "unity-linux"), "Unity EditMode+PlayMode tests").Run
	pass := strings.Index(run, "Test run completed. Exiting with code 0")
	final := strings.Index(run, "ok=final; break")
	retry := strings.Index(run, `tests attempt $i failed (rc=$rc); retrying`)
	if pass < 0 || final < pass || retry < final {
		t.Fatal("a completed failing verdict must be final and checked before the retry path")
	}
	restore := strings.LastIndex(run, `sudo cp -a "$lib_snap" client/Library`)
	fail := strings.Index(run, `if [ -n "$verdict_failed" ]; then`)
	if fail < restore {
		t.Fatal("the step must restore the clean Library snapshot before failing on a verdict")
	}
}

// ADR-0077: each required job runs the Unity-independent verifier phase
// before joining its Unity job; the final verifier reuses that file and adds
// the Unity gates. The pre phase never fails the job on its own.
func TestRequiredJobsRunPreUnityPhase(t *testing.T) {
	wf := verifyWf(t)
	after := map[string][]string{
		"verify-linux":   {"Verifier environment", "Unity scope", "Install protoc + Go tool binaries (pinned)"},
		"verify-windows": {"Verifier environment", "Unity scope", "Unity editor path", "PostgreSQL test server (EDB pinned)", "Install protoc + Go tool binaries (pinned)"},
	}
	for job, deps := range after {
		j := jobNamed(t, wf, job)
		pre := stepNamed(t, j, "Q0-Q6 verifier (pre-Unity phase)")
		wait := stepNamed(t, j, "Wait for Unity job")
		ver := stepNamed(t, j, "Run Q0-Q6 verifier")
		if pre.Index >= wait.Index {
			t.Fatalf("job %q: the pre-Unity phase must run before the Unity join", job)
		}
		for _, d := range deps {
			if stepNamed(t, j, d).Index >= pre.Index {
				t.Fatalf("job %q: pre-Unity phase must follow %q", job, d)
			}
		}
		if !strings.Contains(pre.Run, "-Phase pre-unity") || !strings.Contains(pre.Run, `-ReportOut "$env:RUNNER_TEMP/verify-pre-unity.json"`) {
			t.Fatalf("job %q: pre-Unity phase must write RUNNER_TEMP/verify-pre-unity.json", job)
		}
		if !strings.Contains(ver.Run, "-Phase unity") || !strings.Contains(ver.Run, `-PreReport "$env:RUNNER_TEMP/verify-pre-unity.json"`) {
			t.Fatalf("job %q: the final verifier must consume the pre-Unity file", job)
		}
		if !strings.Contains(pre.If, "!cancelled()") || !strings.Contains(stepBlock(t, wf, job, pre.Name), "continue-on-error: true") {
			t.Fatalf("job %q: the pre-Unity phase must run after earlier failures and never fail the job itself", job)
		}
	}
	ps, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "verify.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"'-phase', $Phase", "'-pre-report', $PreReport", "'verify-pre-unity'"} {
		if !strings.Contains(string(ps), want) {
			t.Fatalf("verify.ps1 missing %q", want)
		}
	}
}
