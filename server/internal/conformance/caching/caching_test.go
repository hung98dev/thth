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
		"verify-linux:unity-image-", "verify-windows:unity-image-",
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
	// Digest env vars must equal the @sha256: suffix of the image env pins.
	for _, pair := range [][2]string{
		{"UNITY_LINUX_IMAGE", "UNITY_LINUX_IMAGE_DIGEST"},
		{"UNITY_WINDOWS_IMAGE", "UNITY_WINDOWS_IMAGE_DIGEST"},
	} {
		imgRe := regexp.MustCompile(pair[0] + `:\s*'[^'@]+@sha256:([0-9a-f]{64})'`)
		digRe := regexp.MustCompile(pair[1] + `:\s*'(sha256:[0-9a-f]{64}|[0-9a-f]{64})'`)
		im, dm := imgRe.FindStringSubmatch(text), digRe.FindStringSubmatch(text)
		if im == nil || dm == nil {
			t.Fatalf("verify.yml env must define %s and %s", pair[0], pair[1])
		}
		if strings.TrimPrefix(dm[1], "sha256:") != im[1] {
			t.Errorf("%s digest %q != %s image digest %q", pair[1], dm[1], pair[0], im[1])
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
		case strings.HasPrefix(s.Key, "unity-image-"):
			want := "env.UNITY_" + strings.ToUpper(jobOS(t, s.Job)) + "_IMAGE_DIGEST"
			if !strings.Contains(s.Key, want) {
				t.Errorf("%s: unity-image key %q missing %q", s.Job, s.Key, want)
			}
		case strings.HasPrefix(s.Key, "unity-library-"):
			want := "env.UNITY_" + strings.ToUpper(jobOS(t, s.Job)) + "_IMAGE_DIGEST"
			if !strings.Contains(s.Key, want) {
				t.Errorf("%s: unity-library key %q missing image digest pin %q", s.Job, s.Key, want)
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
	for _, entry := range []string{"go-toolchain", "unity-editor-image", "unity-library"} {
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
	telData := "{\"step\":\"go-toolchain\",\"result\":\"hit\",\"wall_seconds\":3.5}\n" +
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
