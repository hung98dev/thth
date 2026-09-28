package caching

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"thinhthan/internal/conformance/gates"
	"thinhthan/internal/stackpin"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not in a repo checkout: %v", err)
	}
	return root
}

func workflowSteps(t *testing.T) []Step {
	t.Helper()
	steps, err := ParseSteps(WorkflowPath(repoRoot(t)))
	if err != nil {
		t.Fatalf("parse verify.yml: %v", err)
	}
	return steps
}

func cacheSteps(t *testing.T) []Step {
	t.Helper()
	cs := CacheSteps(workflowSteps(t))
	if len(cs) == 0 {
		t.Fatal("verify.yml has no actions/cache steps")
	}
	return cs
}

func workflowText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(WorkflowPath(repoRoot(t)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// jobOS returns the runner OS segment a job's cache keys must carry.
func jobOS(t *testing.T, job string) string {
	t.Helper()
	switch job {
	case "verify-linux":
		return "Linux"
	case "verify-windows":
		return "Windows"
	default:
		t.Fatalf("unexpected job %q carries a cache step", job)
		return ""
	}
}

// CI-001: every cache mechanism is the pinned actions/cache commit.
func TestCacheActionPinnedSha(t *testing.T) {
	want := "actions/cache@" + stackpin.GitHubActions["actions/cache"].SHA
	if stackpin.GitHubActions["actions/cache"].Tag != "v6.1.0" {
		t.Fatalf("stackpin actions/cache tag drifted: %v", stackpin.GitHubActions["actions/cache"])
	}
	for _, s := range cacheSteps(t) {
		if s.Uses != want {
			t.Errorf("%s/%s: uses %q, want %q", s.Job, s.Name, s.Uses, want)
		}
	}
	// The cache must exist for every packet-mandated scope on both jobs.
	var found []string
	for _, s := range cacheSteps(t) {
		found = append(found, s.Job+":"+s.Key)
	}
	for _, need := range []string{
		"verify-linux:go-build-", "verify-windows:go-build-",
		"verify-windows:unity-editor-", "verify-windows:cli-tools-",
		"verify-linux:unity-library-", "verify-windows:unity-library-",
		"verify-windows:edb-",
	} {
		ok := false
		for _, f := range found {
			if strings.HasPrefix(f[strings.Index(f, ":")+1:], need[strings.Index(need, ":")+1:]) && strings.HasPrefix(f, need[:strings.Index(need, ":")+1]) {
				ok = true
			}
		}
		if !ok {
			t.Errorf("missing cache step for scope %q (have: %v)", need, found)
		}
	}
}

// CI-001: every cache key hashes all its pin inputs (OS + version/digest +
// content lockfiles), and the *_IMAGE_DIGEST env pins cannot drift from the
// image pins they mirror.
func TestCacheKeysCoverPinInputs(t *testing.T) {
	text := workflowText(t)
	// ADR-0073: no image cache; the Windows editor is a native install pinned
	// by the stackpin installer URL + sha256.
	if strings.Contains(text, "unity-image-") || strings.Contains(text, "UNITY_WINDOWS_IMAGE") {
		t.Error("verify.yml must not cache unity images or reference a Windows unityci image (ADR-0073)")
	}
	ed := stackpin.UnityWindowsInstallers["editor"]
	for _, need := range []string{
		"UNITY_WINDOWS_EDITOR_URL: '" + ed.URL + "'",
		"UNITY_WINDOWS_EDITOR_SHA256: '" + ed.SHA256 + "'",
	} {
		if !strings.Contains(text, need) {
			t.Errorf("verify.yml env must pin %s (stackpin.UnityWindowsInstallers)", need)
		}
	}
	// Digest env vars must equal the @sha256: suffix of the image env pins,
	// and *_IMAGE_TAG must equal the image's repo:tag prefix — docker run
	// resolves tag refs after pull+tag, digest refs are not used by run.
	for _, pair := range [][2]string{
		{"UNITY_LINUX_IMAGE", "UNITY_LINUX_IMAGE_DIGEST"},
	} {
		imgRe := regexp.MustCompile(pair[0] + `:\s*'([^'@]+)@sha256:([0-9a-f]{64})'`)
		digRe := regexp.MustCompile(pair[1] + `:\s*'(sha256:[0-9a-f]{64}|[0-9a-f]{64})'`)
		tagRe := regexp.MustCompile(pair[0] + `_TAG:\s*'([^']+)'`)
		im, dm := imgRe.FindStringSubmatch(text), digRe.FindStringSubmatch(text)
		tm := tagRe.FindStringSubmatch(text)
		if im == nil || dm == nil || tm == nil {
			t.Fatalf("verify.yml env must define %s, %s and %s_TAG", pair[0], pair[1], pair[0])
		}
		if strings.TrimPrefix(dm[1], "sha256:") != im[2] {
			t.Errorf("%s digest %q != %s image digest %q", pair[1], dm[1], pair[0], im[2])
		}
		if tm[1] != im[1] {
			t.Errorf("%s_TAG %q != %s repo:tag %q", pair[0], tm[1], pair[0], im[1])
		}
	}
	// Containers must run the tag ref (docker load restores RepoTags, not
	// RepoDigests); the digest ref may only feed the pull path via the
	// `UNITY_IMAGE:` step-env mapping in the load/pull steps.
	for _, line := range strings.Split(text, "\n") {
		for _, img := range []string{"UNITY_LINUX_IMAGE"} {
			if strings.Contains(line, "${{ env."+img+" }}") && !strings.Contains(line, "UNITY_IMAGE: ${{") {
				t.Errorf("digest image ref outside pull mapping — use %s_TAG (docker load drops digest refs): %s", img, strings.TrimSpace(line))
			}
		}
	}

	for _, s := range cacheSteps(t) {
		if s.Key == "" {
			t.Errorf("%s/%s: cache step has no key", s.Job, s.Name)
			continue
		}
		if !strings.Contains(s.Key, "runner.os") {
			t.Errorf("%s/%s: key %q lacks ${{ runner.os }}", s.Job, s.Name, s.Key)
		}
		switch {
		case strings.HasPrefix(s.Key, "go-build-"):
			for _, need := range []string{"env.GO_VERSION", "hashFiles('server/go.sum'"} {
				if !strings.Contains(s.Key, need) {
					t.Errorf("%s: go-build key %q missing %q", s.Job, s.Key, need)
				}
			}
		case strings.HasPrefix(s.Key, "unity-editor-"):
			if s.Job != "verify-windows" || !strings.Contains(s.Key, "env.UNITY_WINDOWS_EDITOR_SHA256") {
				t.Errorf("%s: unity-editor cache is Windows-only and keyed by env.UNITY_WINDOWS_EDITOR_SHA256, got %q", s.Job, s.Key)
			}
		case strings.HasPrefix(s.Key, "cli-tools-"):
			if s.Job != "verify-windows" {
				t.Errorf("cli-tools cache must not exist on %s", s.Job)
			}
			for _, a := range []stackpin.CliAsset{
				stackpin.CliAssets["pwsh-windows"], stackpin.CliAssets["jq-windows"],
				stackpin.CliAssets["gh-windows"], stackpin.CliAssets["git-lfs-windows"],
			} {
				if !strings.Contains(s.Key, a.SHA256[:8]) {
					t.Errorf("cli-tools key %q missing pin sha prefix %s", s.Key, a.SHA256[:8])
				}
			}
			for _, v := range []string{stackpin.Pwsh, stackpin.Jq, stackpin.GhCli, stackpin.GitLfs} {
				if !strings.Contains(s.Key, v) {
					t.Errorf("cli-tools key %q missing version pin %s", s.Key, v)
				}
			}
		case strings.HasPrefix(s.Key, "unity-library-"):
			// Editor pin per OS: Linux image digest, Windows native installer.
			want := "env.UNITY_LINUX_IMAGE_DIGEST"
			if jobOS(t, s.Job) == "Windows" {
				want = "env.UNITY_WINDOWS_EDITOR_SHA256"
			}
			if !strings.Contains(s.Key, want) {
				t.Errorf("%s: unity-library key %q missing editor pin %q", s.Job, s.Key, want)
			}
			if !strings.Contains(s.Key, "hashFiles(") ||
				!strings.Contains(s.Key, "manifest.json") ||
				!strings.Contains(s.Key, "ProjectSettings") {
				t.Errorf("%s: unity-library key %q must hash manifest.json + ProjectSettings", s.Job, s.Key)
			}
		case strings.HasPrefix(s.Key, "edb-"):
			if s.Job != "verify-windows" {
				t.Errorf("edb cache must not exist on %s", s.Job)
			}
			if !strings.Contains(s.Key, stackpin.PostgreSQL) {
				t.Errorf("edb key %q missing postgres version pin %s", s.Key, stackpin.PostgreSQL)
			}
			if !strings.Contains(s.Key, "env.EDB_ZIP_SHA256") &&
				!strings.Contains(s.Key, stackpin.EdbZipSHA256) {
				t.Errorf("edb key %q missing zip sha256 pin", s.Key)
			}
		default:
			t.Errorf("%s/%s: unrecognized cache key scope %q", s.Job, s.Name, s.Key)
		}
	}
}

// CI-001: restore-keys must be literal prefixes of the key that keep the OS
// and every pin token — a fallback may only roll the content hash, never
// substitute a different pinned version or OS.
func TestRestoreKeysNeverCrossPinOrOs(t *testing.T) {
	for _, s := range cacheSteps(t) {
		for _, rk := range s.RestoreKeys {
			if !strings.Contains(rk, "runner.os") {
				t.Errorf("%s/%s: restore-key %q drops the OS segment", s.Job, s.Name, rk)
			}
			if !strings.HasPrefix(s.Key, strings.TrimSuffix(rk, "-")) {
				t.Errorf("%s/%s: restore-key %q is not a prefix of key %q", s.Job, s.Name, rk, s.Key)
			}
			for _, pin := range PinTokens(s.Key) {
				if !strings.Contains(rk, pin) {
					t.Errorf("%s/%s: restore-key %q drops pin %q from key %q", s.Job, s.Name, rk, pin, s.Key)
				}
			}
			if strings.Contains(rk, "hashFiles(") {
				t.Errorf("%s/%s: restore-key %q must not pin a content hash", s.Job, s.Name, rk)
			}
		}
	}
}

// CI-001/BLK-005: unity-library stores content derived from the hashed inputs
// (PackageCache/ScriptAssemblies are a function of manifest/lock/ProjectSettings/
// compiler flags), so a restore-keys prefix hit is a silent wrong-content
// restore — exact key only.
func TestLibraryCacheExactKeyOnly(t *testing.T) {
	for _, s := range cacheSteps(t) {
		if strings.HasPrefix(s.Key, "unity-library-") && len(s.RestoreKeys) != 0 {
			t.Errorf("%s/%s: unity-library must not set restore-keys (exact key only; a prefix restore returns a Library built from different inputs — BLK-005)", s.Job, s.Name)
		}
	}
}

// CI-002: no gate, precondition, materialization step or licence activation
// may be conditioned on a cache outcome.
func TestNoGateSkippedOnCacheHit(t *testing.T) {
	for _, s := range workflowSteps(t) {
		if strings.Contains(s.If, "cache-hit") || strings.Contains(s.If, "steps.cache") {
			t.Errorf("%s/%s: step gated on cache-hit (%q)", s.Job, s.Name, s.If)
		}
	}
	required := []string{
		"Fork guard", "Merge freeze guard",
		"Unity materialization (licence retry <=5)",
		"Unity materialized drift check", "Run Q0-Q6 verifier",
	}
	steps := workflowSteps(t)
	for _, job := range []string{"verify-linux", "verify-windows"} {
		for _, name := range required {
			found := false
			for _, s := range steps {
				if s.Job == job && s.Name == name {
					found = true
					if strings.Contains(strings.ToLower(s.If), "cache") {
						t.Errorf("%s/%s: required step conditioned on cache (%q)", job, name, s.If)
					}
				}
			}
			if !found {
				t.Errorf("%s: required step %q missing", job, name)
			}
		}
	}
	// Licence activation still runs inside the test step every attempt.
	if !strings.Contains(workflowText(t), "Unity_lic.ulf") {
		t.Error("verify.yml no longer mounts/checks Unity_lic.ulf — licence state must stay live")
	}
}

// CI-002: the §4b materialized-drift commit loop must be intact on both jobs:
// drift step + unconditional upload of unity-materialized-<os>.
func TestMaterializeCommitStillRequiredOnHit(t *testing.T) {
	steps := workflowSteps(t)
	for _, job := range []string{"verify-linux", "verify-windows"} {
		var drift, upload bool
		for _, s := range steps {
			if s.Job != job {
				continue
			}
			if s.Name == "Unity materialized drift check" {
				drift = true
				if strings.Contains(strings.ToLower(s.If), "cache") {
					t.Errorf("%s: drift check conditioned on cache", job)
				}
			}
			if s.Name == "Upload unity-materialized drift" &&
				strings.Contains(s.Uses, "actions/upload-artifact@") &&
				strings.Contains(s.If, "drift") {
				upload = true
			}
		}
		if !drift || !upload {
			t.Errorf("%s: materialization drift-commit loop incomplete (drift=%v upload=%v)", job, drift, upload)
		}
	}
	if n := strings.Count(workflowText(t), "commit unity-materialized"); n < 2 {
		t.Errorf("'commit unity-materialized' found %d times, want >= 2 (one per job)", n)
	}
}

// CI-002: licence and credential state must never be under a cache path/key.
func TestLicenceStateNeverCached(t *testing.T) {
	needles := []string{
		"unity-lic", "unity_lic", ".ulf", "programdata", "unity3d",
		"licence", "license", "licen",
	}
	for _, s := range cacheSteps(t) {
		for _, field := range append(append(append([]string{s.Key}, s.Path...), s.RestoreKeys...), s.Name) {
			l := strings.ToLower(field)
			for _, n := range needles {
				if strings.Contains(l, n) {
					t.Errorf("%s/%s: cache field %q references licence state %q", s.Job, s.Name, field, n)
				}
			}
		}
	}
}

// CI-003: every cache step has a telemetry emission in the same job, and the
// merge into verify-report.json produces hit|miss + wall_seconds entries.
func TestWallTimeFieldsRecorded(t *testing.T) {
	text := workflowText(t)
	root := repoRoot(t)
	for _, helper := range []string{"cache_telemetry.sh", "cache_telemetry.ps1"} {
		data, err := os.ReadFile(filepath.Join(root, ".devin", "scripts", helper))
		if err != nil {
			t.Fatalf("telemetry helper %s missing: %v", helper, err)
		}
		if !strings.Contains(string(data), "RUNNER_TEMP") {
			t.Errorf("%s must write to RUNNER_TEMP (outside the workspace, Q6)", helper)
		}
	}
	for _, entry := range []string{"go-build", "unity-editor-image", "unity-library"} {
		if !strings.Contains(text, entry) {
			t.Errorf("verify.yml emits no %q telemetry entry", entry)
		}
	}
	if !strings.Contains(text, "edb-postgres") {
		t.Error("verify.yml emits no edb-postgres telemetry entry")
	}
	ps, err := os.ReadFile(filepath.Join(root, "scripts", "verify.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, need := range []string{"cachemerge", "THINHTHAN_CACHE_HIT_GO", "Write-CacheTelemetry"} {
		if !strings.Contains(string(ps), need) {
			t.Errorf("verify.ps1 lacks %q — cached_steps would never reach the report", need)
		}
	}

	// Functional: merge two entries into a report and read them back.
	dir := t.TempDir()
	report := filepath.Join(dir, "verify-report.json")
	tel := filepath.Join(dir, "cache-telemetry.jsonl")
	base := `{"schema":"verify-report-v1","result":"PASSED","os":"linux","gates":[]}`
	if err := os.WriteFile(report, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	telData := "{\"step\":\"go-build\",\"result\":\"hit\",\"wall_seconds\":3.5}\n" +
		"{\"step\":\"unity-editor-image\",\"result\":\"miss\",\"wall_seconds\":240}\n"
	if err := os.WriteFile(tel, []byte(telData), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MergeTelemetryIntoReport(report, tel); err != nil {
		t.Fatalf("merge: %v", err)
	}
	var obj struct {
		Result      string  `json:"result"`
		CachedSteps []Entry `json:"cached_steps"`
	}
	data, _ := os.ReadFile(report)
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("merged report invalid: %v", err)
	}
	if obj.Result != "PASSED" {
		t.Error("merge corrupted existing report fields")
	}
	if len(obj.CachedSteps) != 2 || obj.CachedSteps[0].Result != "hit" ||
		obj.CachedSteps[0].WallSeconds != 3.5 || obj.CachedSteps[1].WallSeconds != 240 {
		t.Errorf("cached_steps wrong: %+v", obj.CachedSteps)
	}
}

// CI-004: a cold and a warm run must yield identical evidence — cached_steps
// never reaches the manifest, and no cache path pollutes the source tree.
func TestEvidenceIdentityIndependentOfCache(t *testing.T) {
	rep := gates.VerifyReport{
		Schema: "verify-report-v1", Result: "PASSED", Go: "go1.27.1",
		RanAt: "2026-09-26T00:00:00Z", OS: "linux",
		Commands: []string{"go test ./..."},
		Gates:    []gates.Gate{{ID: "Q0", Title: "x", Checks: []gates.Check{{ID: "Q0.x", Status: gates.StatusPass}}}},
	}
	coldJSON, _ := json.Marshal(rep)

	dir := t.TempDir()
	warmPath := filepath.Join(dir, "verify-report.json")
	if err := os.WriteFile(warmPath, coldJSON, 0o644); err != nil {
		t.Fatal(err)
	}
	telPath := filepath.Join(dir, "cache-telemetry.jsonl")
	if err := os.WriteFile(telPath, []byte(`{"step":"verify","result":"hit","wall_seconds":31}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MergeTelemetryIntoReport(warmPath, telPath); err != nil {
		t.Fatal(err)
	}
	warmJSON, _ := os.ReadFile(warmPath)

	// The manifest consumes reports via non-strict decode into VerifyReport —
	// identical decoded values => identical manifests (CI-004).
	var coldRep, warmRep gates.VerifyReport
	if err := json.Unmarshal(coldJSON, &coldRep); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(warmJSON, &warmRep); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(coldRep, warmRep) {
		t.Error("cached_steps changed the decoded VerifyReport")
	}
	coldM, err := gates.MergeReports("IMP-106", "tree", "run", 1,
		map[string]gates.VerifyReport{"linux": coldRep, "windows": coldRep})
	if err != nil {
		t.Fatal(err)
	}
	warmM, err := gates.MergeReports("IMP-106", "tree", "run", 1,
		map[string]gates.VerifyReport{"linux": warmRep, "windows": warmRep})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(coldM, warmM) {
		t.Error("evidence manifest differs between cold and warm cache runs")
	}

	// No cached path may be a tracked source-tree file: Library is gitignored
	// build output; everything else lives in runner.temp or the home dir.
	gitignore, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gitignore), "client/Library") {
		t.Error(".gitignore must cover client/Library — a cached path would dirty source_tree_hash")
	}
	for _, s := range cacheSteps(t) {
		for _, p := range s.Path {
			np := strings.ReplaceAll(p, "/", "\\")
			ok := strings.Contains(p, "runner.temp") || strings.HasPrefix(p, "~/") ||
				p == "client/Library" || strings.Contains(np, "AppData")
			if !ok {
				t.Errorf("%s/%s: cache path %q lands inside the source tree", s.Job, s.Name, p)
			}
		}
	}
}

// ADR-0073: cache_warm.yml saves the pure-pin caches in the main scope; each
// of its cache steps must be byte-identical (key + path) to a verify.yml cache
// step, and it never touches secrets or licences.
func TestCacheWarmMirrorsVerifyCaches(t *testing.T) {
	root := repoRoot(t)
	warm, err := ParseSteps(filepath.Join(root, ".github", "workflows", "cache_warm.yml"))
	if err != nil {
		t.Fatalf("parse cache_warm.yml: %v", err)
	}
	verify := cacheSteps(t)
	warmCaches := CacheSteps(warm)
	if len(warmCaches) == 0 {
		t.Fatal("cache_warm.yml has no cache steps")
	}
	for _, w := range warmCaches {
		if strings.HasPrefix(w.Key, "unity-library-") {
			t.Errorf("%s: unity-library is content-derived and needs a licence; not warmed here", w.Name)
		}
		match := false
		for _, v := range verify {
			if v.Key == w.Key && reflect.DeepEqual(v.Path, w.Path) && jobOSMatches(v.Job, w.Job) {
				match = true
			}
		}
		if !match {
			t.Errorf("cache_warm %s/%s (key %q) has no identical verify.yml cache step", w.Job, w.Name, w.Key)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "cache_warm.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"secrets.", "UNITY_SERIAL", "Unity_lic"} {
		if strings.Contains(string(data), bad) {
			t.Errorf("cache_warm.yml must not reference %q", bad)
		}
	}
}

func jobOSMatches(verifyJob, warmJob string) bool {
	return (verifyJob == "verify-linux" && warmJob == "warm-linux") ||
		(verifyJob == "verify-windows" && warmJob == "warm-windows")
}

// BLK-008: every Unity editor invocation (materialization, licence probe,
// EditMode/PlayMode test containers) keeps network egress — the licensing
// client's access-token refresh needs license.unity3d.com, and a
// --network=none container reproducibly self-SIGKILLed the editor during
// early init. The workflow must never pass a --network flag to docker.
func TestUnityContainersKeepNetworkEgress(t *testing.T) {
	wf := workflowText(t)
	if strings.Contains(wf, "--network") {
		t.Error("verify.yml must not pass --network to any docker run — Unity licensing needs egress (BLK-008)")
	}
	if strings.Contains(wf, "TNET") || strings.Contains(wf, "NET=") {
		t.Error("verify.yml still carries the NET/TNET offline conditional (BLK-008)")
	}
}

// BLK-009: every containerized Unity run bind-mounts the licensing state
// dirs writable — unity-lic at /root/.local/share/unity3d, unity-cfg at
// /root/.config/unity3d AND unity-cache at /root/.cache. Without the
// cache dir the licensing client cannot persist its token/state
// (CreateDirectory '/root/.cache/unity3d' failed) and self-terminates
// the editor's process group ~25s into startup.
func TestUnityContainersMountLicensingDirs(t *testing.T) {
	wf := workflowText(t)
	lic := strings.Count(wf, "unity-lic:/root/.local/share/unity3d")
	cfg := strings.Count(wf, "unity-cfg:/root/.config/unity3d")
	cache := strings.Count(wf, "unity-cache:/root/.cache")
	if lic == 0 || cfg == 0 {
		t.Fatal("verify.yml lost the unity-lic/unity-cfg bind mounts entirely")
	}
	if lic != cfg {
		t.Errorf("unity-lic mounts (%d) and unity-cfg mounts (%d) diverge — every licensing container needs both", lic, cfg)
	}
	if cache != lic {
		t.Errorf("unity-cache:/root/.cache mounts (%d) != unity-lic mounts (%d) — every container running the Unity editor needs the licensing cache dir writable (BLK-009)", cache, lic)
	}
	if !strings.Contains(wf, `mkdir -p "$RUNNER_TEMP/unity-lic" "$RUNNER_TEMP/unity-cache"`) &&
		!strings.Contains(wf, `mkdir -p "$RUNNER_TEMP/unity-cache" "$RUNNER_TEMP/unity-lic"`) &&
		!strings.Contains(wf, "unity-cache\"") {
		t.Error("verify.yml must mkdir the unity-cache runner dir before mounting it")
	}
}

// BLK-009 (primary fix): every `unity-editor` invocation in the Linux job
// runs with -batchmode. Without it the editor runs headed under Xvfb and
// the auto-quit/fatal path issues killpg on its own process group —
// deterministic SIGKILL ~15ms into assembly registration.
func TestUnityRunsBatchmode(t *testing.T) {
	wf := workflowText(t)
	for _, line := range strings.Split(wf, "\n") {
		s := strings.TrimSpace(line)
		// covered below via the wrapped bash -c payloads too: match any
		// `unity-editor` invocation that runs the editor (not just the
		// wrapper's -version probe is exempt — it never opens the editor).
		if strings.Contains(s, "unity-editor") &&
			(strings.Contains(s, "-runTests") || strings.Contains(s, "-projectPath") ||
				strings.Contains(s, "-serial") || strings.Contains(s, "-quit")) &&
			!strings.Contains(s, "-batchmode") {
			t.Errorf("unity-editor invocation missing -batchmode (headed under Xvfb → killpg, BLK-009): %s", s)
		}
	}
}

// BLK-010: `-runTests` invocations must not touch the GL/Xvfb path at all and
// must log to a real file — the editor's in-process abort fired inside
// GfxDevice init (GLX over Xvfb+llvmpipe) even under -batchmode, and
// `-logFile -` loses the buffered tail containing the fatal line on SIGKILL.
func TestUnityTestRunsHeadless(t *testing.T) {
	wf := workflowText(t)
	lines := strings.Split(wf, "\n")
	for i, line := range lines {
		// an editor invocation may span continued lines — join the block
		if !strings.Contains(line, "unity-editor") {
			continue
		}
		inv := line
		for strings.HasSuffix(strings.TrimSpace(inv), "\\") && i+1 < len(lines) {
			i++
			inv += " " + lines[i]
		}
		if !strings.Contains(inv, "-runTests") {
			continue
		}
		if !strings.Contains(inv, "-nographics") {
			t.Errorf("-runTests invocation missing -nographics (GLX/Xvfb abort surface, BLK-010): %s", strings.TrimSpace(inv))
		}
		m := regexp.MustCompile(`-logFile\s+(\S+)`).FindStringSubmatch(inv)
		if m == nil || m[1] == "-" {
			t.Errorf("-runTests invocation must write -logFile to a real file (stdout tail is lost on SIGKILL, BLK-010): %s", strings.TrimSpace(inv))
		}
	}
}

// BLK-013: the editor's watchdog SIGKILLs its own process group during
// shutdown ~5s AFTER 'Test run completed. Exiting with code 0' — rc=137 is
// not a test failure once the verdict was written. The Linux -runTests pass
// condition must gate on the completion line in the real -logFile plus a
// Passed results.xml, never on the container exit code alone.
func TestUnityTestPassGatesOnCompletionLine(t *testing.T) {
	wf := workflowText(t)
	if !strings.Contains(wf, "Test run completed. Exiting with code 0") {
		t.Error("verify.yml Linux test step must treat a post-completion rc=137 as PASS via the editor-log completion line (BLK-013)")
	}
	// the pass branch must still require a Passed results.xml — completion
	// alone must not mask a real test failure that happened to print the line
	idx := strings.Index(wf, "Test run completed. Exiting with code 0")
	if idx < 0 {
		return
	}
	tail := wf[idx:]
	if !strings.Contains(tail, `result="Passed"`) || !strings.Contains(tail, "results.xml") {
		t.Error("BLK-013 pass condition must still require a Passed results.xml alongside the completion line")
	}
}

// BLK-014: when the watchdog killpg fires mid-run (before the completion
// line), the exit-code gate never sees the verdict — the editor must run
// under setsid so the kill only destroys the editor's own process group
// and the wrapping shell (container PID 1) survives to exit 0 when a
// Passed results.xml was already committed.
func TestUnityEditorRunsUnderSetsid(t *testing.T) {
	wf := workflowText(t)
	if !strings.Contains(wf, "setsid -w unity-editor") {
		t.Error("verify.yml -runTests must wrap unity-editor in setsid so killpg cannot take down the container entrypoint (BLK-014)")
	}
	idx := strings.Index(wf, "setsid -w unity-editor")
	if idx < 0 {
		return
	}
	tail := wf[idx:]
	if !strings.Contains(tail, `result="Passed"`) || !strings.Contains(tail, "exit 0") {
		t.Error("setsid wrapper must exit 0 when a Passed results.xml exists (post-verdict kill is not a failure)")
	}
}

// BLK-017: a Linux test container must not inherit editor state left by a
// previous editor that ended in its SIGKILL exit (user config, analytics and
// metrics DBs, client/Temp). Each -runTests container mounts per-attempt
// licensing/config/cache dirs seeded only with the activated licence file.
func TestUnityTestContainersStartFromFreshState(t *testing.T) {
	wf := workflowText(t)
	idx := strings.Index(wf, "setsid -w unity-editor")
	if idx < 0 {
		t.Fatal("test container invocation not found")
	}
	head := wf[:idx]
	start := strings.LastIndex(head, `att="$RUNNER_TEMP/unity-att"`)
	if start < 0 {
		t.Fatal("test containers must use a per-attempt state dir (BLK-017)")
	}
	block := wf[start:idx]
	for _, want := range []string{
		`sudo rm -rf "$att" client/Temp`,
		`cp "$RUNNER_TEMP/unity-lic/Unity/Unity_lic.ulf" "$att/unity-lic/Unity/"`,
		`-v "$att/unity-lic:/root/.local/share/unity3d"`,
		`-v "$att/unity-cfg:/root/.config/unity3d"`,
		`-v "$att/unity-cache:/root/.cache"`,
	} {
		if !strings.Contains(block, want) {
			t.Errorf("test container setup missing %q (BLK-017)", want)
		}
	}
}
