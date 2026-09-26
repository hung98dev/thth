package gates

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"thinhthan/internal/stackpin"
)

// SourceTreeHashExclusions are excluded from the ADR-0057 source tree hash:
// control/status files and evidence, so the hash of a -done status PR equals
// its tested implementation tree.
var SourceTreeHashExclusions = []string{
	"docs/10_implementation/evidence/",
	"docs/10_implementation/task_queue.md",
	"docs/10_implementation/known_blockers.md",
}

// SourceTreeHashExcluded reports whether path is excluded from the hash.
func SourceTreeHashExcluded(path string) bool {
	for _, e := range SourceTreeHashExclusions {
		if path == strings.TrimSuffix(e, "/") || strings.HasPrefix(path, e) {
			return true
		}
	}
	return false
}

// SourceTreeHash computes the ADR-0057 hash: SHA-256 over sorted
// "path\0git-blob-sha\n" lines of `git ls-files` minus the exclusions. The
// blob SHA-1 makes the digest identical across OSes and line-ending settings.
func SourceTreeHash(root string) (string, error) {
	out, err := gitDir(root, "ls-files", "-s")
	if err != nil {
		return "", fmt.Errorf("git ls-files: %v", err)
	}
	type entry struct{ path, blob string }
	var entries []entry
	for _, line := range strings.Split(out, "\n") {
		// format: "<mode> <sha1> <stage>\t<path>"
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		meta := strings.Fields(line[:tab])
		if len(meta) < 2 {
			continue
		}
		path := strings.Trim(line[tab+1:], `"`)
		if SourceTreeHashExcluded(path) {
			continue
		}
		entries = append(entries, entry{path: path, blob: meta[1]})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	h := sha256.New()
	for _, e := range entries {
		io.WriteString(h, e.path+"\x00"+e.blob+"\n")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// EvidenceManifest mirrors docs/09_testing/test_and_release_evidence.md
// schema-v2 (ADR-0057): fixed toolchain pins, commands, test summary,
// per-OS job results, worktree cleanliness, ci_run_id + run_attempt.
type EvidenceManifest struct {
	SchemaVersion  int    `json:"schema_version"`
	TaskID         string `json:"task_id"`
	SourceTreeHash string `json:"source_tree_hash"`
	Toolchain      struct {
		Go          string `json:"go"`
		Unity       string `json:"unity"`
		PostgreSQL  string `json:"postgresql"`
		Protoc      string `json:"protoc"`
		ProtocGenGo string `json:"protoc_gen_go"`
	} `json:"toolchain"`
	Commands    []string `json:"commands"`
	TestSummary struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Skipped int `json:"skipped"`
	} `json:"test_summary"`
	SkippedReasons []struct {
		ID      string `json:"id"`
		Reason  string `json:"reason"`
		Allowed bool   `json:"allowed"`
	} `json:"skipped_reasons"`
	ContentRevision string `json:"content_revision"`
	CIRunID         string `json:"ci_run_id"`
	RunAttempt      int    `json:"run_attempt"`
	Jobs            []struct {
		Name   string `json:"name"`
		OS     string `json:"os"`
		Result string `json:"result"`
	} `json:"jobs"`
	WorktreeClean bool   `json:"worktree_clean"`
	Result        string `json:"result"`
}

var sha256HexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// decodeManifest is strict JSON decode into an EvidenceManifest.
func decodeManifest(data []byte, m *EvidenceManifest) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(m)
}

// ValidateManifest returns schema violations for a schema-v2 evidence
// manifest at path (empty list = valid). A missing file is a violation.
func ValidateManifest(path, taskID string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("evidence manifest missing: %s", path)}
	}
	var m EvidenceManifest
	if err := decodeManifest(data, &m); err != nil {
		return []string{fmt.Sprintf("evidence manifest invalid JSON/schema: %v", err)}
	}
	var problems []string
	if m.SchemaVersion != 2 {
		problems = append(problems, "schema_version must be 2")
	}
	if m.TaskID != taskID {
		problems = append(problems, fmt.Sprintf("task_id %q != %q", m.TaskID, taskID))
	}
	if !sha256HexRe.MatchString(m.SourceTreeHash) {
		problems = append(problems, "source_tree_hash is not SHA-256 hex")
	}
	if m.Toolchain.Go != stackpin.GoVersion ||
		m.Toolchain.Unity != stackpin.UnityEditor ||
		m.Toolchain.PostgreSQL != stackpin.PostgreSQL ||
		m.Toolchain.Protoc != stackpin.Protoc ||
		m.Toolchain.ProtocGenGo != stackpin.ProtocGenGo {
		problems = append(problems, "toolchain fields differ from the canonical matrix")
	}
	if len(m.Commands) == 0 {
		problems = append(problems, "commands[] must be non-empty")
	}
	ts := m.TestSummary
	if ts.Total != ts.Passed+ts.Failed+ts.Skipped {
		problems = append(problems, "test_summary totals inconsistent")
	}
	if ts.Failed != 0 {
		problems = append(problems, "test_summary.failed must be 0")
	}
	for _, sr := range m.SkippedReasons {
		if sr.ID == "" || sr.Reason == "" || !sr.Allowed {
			problems = append(problems, "skipped_reasons entries need id + reason + allowed")
		}
	}
	if m.ContentRevision == "" {
		problems = append(problems, "content_revision missing")
	}
	if m.CIRunID == "" {
		problems = append(problems, "ci_run_id empty — DONE requires CI evidence")
	}
	if m.RunAttempt < 1 {
		problems = append(problems, "run_attempt must be >= 1")
	}
	for _, j := range m.Jobs {
		if j.Name == "" || (j.OS != "linux" && j.OS != "windows") || j.Result != "PASSED" {
			problems = append(problems, fmt.Sprintf("jobs[] entry invalid: %+v", j))
		}
	}
	if !m.WorktreeClean {
		problems = append(problems, "worktree_clean must be true")
	}
	if m.Result != "PASSED" {
		problems = append(problems, "result must be PASSED")
	}
	return problems
}

// runAPI is one GitHub Actions run record relevant to the Q6 check.
type runAPI struct {
	WorkflowPath string `json:"path"`
	Conclusion   string `json:"conclusion"`
	HeadSHA      string `json:"head_sha"`
	RunAttempt   int    `json:"run_attempt"`
}

// CheckRunIdentity calls the GitHub REST API for run ciRunID of repo and
// requires it to be a successful verify.yml run whose head_sha is an ancestor
// of (or equal to) the current HEAD — the evidence belongs to this PR's own
// verify.yml run history (ADR-0068).
func CheckRunIdentity(root, repo, runID string, runAttempt int, token string) []string {
	var problems []string
	if repo == "" || runID == "" || token == "" {
		return []string{"missing repo/run_id/token for the run API check"}
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%s", repo, runID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []string{fmt.Sprintf("run API request failed: %v", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return []string{fmt.Sprintf("run %s not found as a run of this repo (HTTP %d)", runID, resp.StatusCode)}
	}
	var r runAPI
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return []string{fmt.Sprintf("run API response invalid: %v", err)}
	}
	if r.WorkflowPath != ".github/workflows/verify.yml" {
		problems = append(problems, fmt.Sprintf("run %s is workflow %q, not verify.yml", runID, r.WorkflowPath))
	}
	if r.Conclusion != "success" {
		problems = append(problems, fmt.Sprintf("run %s conclusion %q != success", runID, r.Conclusion))
	}
	if runAttempt > 0 && r.RunAttempt != runAttempt {
		problems = append(problems, fmt.Sprintf("run_attempt %d != API %d", runAttempt, r.RunAttempt))
	}
	// The evidence run must be an ancestor of the current head (own run). A
	// squash/rebase merge never makes the PR head sha an ancestor — fall back
	// to the merged PR's merge_commit_sha, which is.
	if r.HeadSHA != "" {
		if out, err := gitDir(root, "merge-base", "--is-ancestor", r.HeadSHA, "HEAD"); err != nil {
			if !landedViaMerge(root, mergedPRSHAs(repo, r.HeadSHA, token, client)) {
				problems = append(problems, fmt.Sprintf("run head_sha %s is not an ancestor of HEAD: %v (%s)", r.HeadSHA, err, out))
			}
		}
	}
	return problems
}

// prSHAs is the subset of the /commits/{sha}/pulls record used here.
type prSHAs struct {
	MergeCommitSHA string  `json:"merge_commit_sha"`
	MergedAt       *string `json:"merged_at"`
	Head           struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// mergedPRSHAs returns the merge_commit_sha of every merged PR whose head
// commit is sha — /commits/{sha}/pulls also lists PRs where sha is only an
// intermediate commit, which must not satisfy the fallback.
func mergedPRSHAs(repo, sha, token string, client *http.Client) []string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/pulls", repo, sha)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var prs []prSHAs
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil
	}
	var shas []string
	for _, pr := range prs {
		if pr.MergedAt != nil && pr.Head.SHA == sha && pr.MergeCommitSHA != "" {
			shas = append(shas, pr.MergeCommitSHA)
		}
	}
	return shas
}

// landedViaMerge reports whether any candidate commit is an ancestor of HEAD.
func landedViaMerge(root string, candidates []string) bool {
	for _, c := range candidates {
		if _, err := gitDir(root, "merge-base", "--is-ancestor", c, "HEAD"); err == nil {
			return true
		}
	}
	return false
}

// VerifyReport is the per-job report written by cmd/verify next to
// verify-report.json.
type VerifyReport struct {
	Schema   string   `json:"schema"`
	Result   string   `json:"result"`
	Go       string   `json:"go"`
	RanAt    string   `json:"ran_at"`
	OS       string   `json:"os"`
	Commands []string `json:"commands"`
	Gates    []Gate   `json:"gates"`
}

// MergeReports builds the schema-v2 evidence manifest from the linux and
// windows verify reports plus run metadata. os -> job name.
func MergeReports(taskID, treeHash, runID string, runAttempt int, reports map[string]VerifyReport) (*EvidenceManifest, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id required")
	}
	m := &EvidenceManifest{
		SchemaVersion:   2,
		TaskID:          taskID,
		SourceTreeHash:  treeHash,
		ContentRevision: "none",
		CIRunID:         runID,
		RunAttempt:      runAttempt,
		WorktreeClean:   true,
		Result:          "PASSED",
	}
	m.Toolchain.Go = stackpin.GoVersion
	m.Toolchain.Unity = stackpin.UnityEditor
	m.Toolchain.PostgreSQL = stackpin.PostgreSQL
	m.Toolchain.Protoc = stackpin.Protoc
	m.Toolchain.ProtocGenGo = stackpin.ProtocGenGo
	for _, osName := range []string{"linux", "windows"} {
		rep, ok := reports[osName]
		if !ok {
			return nil, fmt.Errorf("missing %s verify report", osName)
		}
		if rep.Result != "PASSED" {
			return nil, fmt.Errorf("%s report result %q != PASSED", osName, rep.Result)
		}
		jobName := map[string]string{"linux": "Q0-Q6 verify (Linux)", "windows": "Q0-Q6 verify (Windows)"}[osName]
		m.Jobs = append(m.Jobs, struct {
			Name   string `json:"name"`
			OS     string `json:"os"`
			Result string `json:"result"`
		}{Name: jobName, OS: osName, Result: "PASSED"})
		m.Commands = append(m.Commands, rep.Commands...)
		for _, g := range rep.Gates {
			for _, c := range g.Checks {
				m.TestSummary.Total++
				switch c.Status {
				case StatusPass:
					m.TestSummary.Passed++
				case StatusFail:
					m.TestSummary.Failed++
				case StatusSkip, StatusDeferred:
					m.TestSummary.Skipped++
					m.SkippedReasons = append(m.SkippedReasons, struct {
						ID      string `json:"id"`
						Reason  string `json:"reason"`
						Allowed bool   `json:"allowed"`
					}{ID: c.ID, Reason: c.Detail, Allowed: true})
				}
			}
		}
	}
	return m, nil
}
