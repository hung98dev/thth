package gates

import (
	"fmt"
	"os"
	"strings"
)

// Env is the verifier's environment contract. In CI every field that the
// gates need is populated by verify.yml from github context; locally the
// fields are empty and the PR-context checks defer instead of failing.
type Env struct {
	CI              bool   // running under CI (GITHUB_ACTIONS=true)
	LocalDefer      bool   // -LocalDeferMissing was passed to verify.ps1
	EventName       string // pull_request | pull_request_target | push
	BaseSHA         string
	HeadSHA         string
	BaseRef         string
	HeadRef         string // PR branch name, e.g. imp/IMP-000-bootstrap
	Repository      string // owner/repo the run belongs to
	HeadRepo        string // owner/repo of the PR head (== Repository for in-repo PRs)
	RunID           string
	RunAttempt      string
	Token           string   // GITHUB_TOKEN for the Q6 run-id API check
	RunnerOS        string   // Linux | Windows | macOS
	UnityResultsDir string   // directory holding Unity test-results XML files
	Worktree        string   // repo root being verified
	MainRef         string   // ref used for "status on main", default "origin/main"
	PRFiles         []string // diff file list base..head (when PR context exists)
	prFilesLoaded   bool
}

// LoadEnv builds the Env from OS environment variables.
func LoadEnv(localDefer bool, unityResultsDir string) *Env {
	e := &Env{
		CI:              strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true"),
		LocalDefer:      localDefer,
		EventName:       os.Getenv("GITHUB_EVENT_NAME"),
		BaseSHA:         os.Getenv("THINHTHAN_BASE_SHA"),
		HeadSHA:         os.Getenv("THINHTHAN_HEAD_SHA"),
		BaseRef:         os.Getenv("GITHUB_BASE_REF"),
		HeadRef:         os.Getenv("GITHUB_HEAD_REF"),
		Repository:      os.Getenv("GITHUB_REPOSITORY"),
		RunID:           os.Getenv("GITHUB_RUN_ID"),
		RunAttempt:      os.Getenv("GITHUB_RUN_ATTEMPT"),
		Token:           os.Getenv("GH_TOKEN"),
		RunnerOS:        os.Getenv("RUNNER_OS"),
		UnityResultsDir: unityResultsDir,
		MainRef:         "origin/main",
	}
	e.HeadRepo = e.Repository // in-repo PRs only; fork guard blocks forks earlier
	if os.Getenv("THINHTHAN_MAIN_REF") != "" {
		e.MainRef = os.Getenv("THINHTHAN_MAIN_REF")
	}
	return e
}

// IsPR reports whether the run is a pull_request* event.
func (e *Env) IsPR() bool {
	return e.EventName == "pull_request" || e.EventName == "pull_request_target"
}

// IsPush reports whether the run is a push event.
func (e *Env) IsPush() bool {
	return e.EventName == "push"
}

// HasPRContext reports whether base/head SHAs and refs are available.
func (e *Env) HasPRContext() bool {
	return e.IsPR() && e.BaseSHA != "" && e.HeadSHA != "" && e.HeadRef != ""
}

// PRContextCheck is the shared preamble for checks that need PR context:
// a FAIL in CI when context is missing, otherwise a DEFERRED marker.
func (e *Env) PRContextCheck(id string) (ok bool, c Check) {
	if e.HasPRContext() {
		return true, Check{}
	}
	if e.CI {
		return false, Fail(id, fmt.Sprintf("PR context missing (event=%q base=%q head=%q ref=%q)",
			e.EventName, e.BaseSHA, e.HeadSHA, e.HeadRef))
	}
	return false, Deferred(id, "PR context (GITHUB_* env) absent — not a PR run")
}

// missingCheck returns a FAIL in CI or DEFERRED(local-missing) locally when a
// required tool/runtime is absent.
func (e *Env) missingCheck(id, detail string) Check {
	if e.CI || !e.LocalDefer {
		return Fail(id, detail+" (required in CI; locally defer with -LocalDeferMissing)")
	}
	return Deferred(id, detail)
}

// CIContextCheck is the preamble for checks that only make sense in CI (run id
// / api checks): local runs defer, CI runs require the context.
func (e *Env) CIContextCheck(id string) (ok bool, c Check) {
	if e.CI && e.RunID != "" && e.RunAttempt != "" {
		return true, Check{}
	}
	if e.CI {
		return false, Fail(id, fmt.Sprintf("CI context missing (run_id=%q attempt=%q)", e.RunID, e.RunAttempt))
	}
	return false, Deferred(id, "not a CI run")
}
