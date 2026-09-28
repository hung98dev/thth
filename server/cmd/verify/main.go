// Command verify runs the Q0-Q6 conformance gates. verify.ps1 is the only
// caller; CI invokes it on both GitHub-hosted OSes and the evidence job merges
// the two per-job reports into the schema-v2 manifest.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"thinhthan/internal/conformance/gates"
)

// Verifier phases (ADR-0077). The default runs every gate at once. CI splits
// the run so the Unity-independent gates execute while the parallel Unity job
// is still running: `pre-unity` writes those gates to an intermediate file,
// `unity` adds the Unity gates and writes the final verify-report.json, which
// is identical in content to a single full run.
const (
	phaseFull     = ""
	phasePreUnity = "pre-unity"
	phaseUnity    = "unity"

	preUnitySchema = "verify-pre-unity-v1"
)

type cliFlags struct {
	localDefer      bool
	unityResultsDir string
	reportOut       string
	mergeDir        string
	printHash       bool
	taskID          string
	planUnity       bool
	phase           string
	preReport       string
}

func main() {
	var f cliFlags
	flag.BoolVar(&f.localDefer, "local-defer", false, "defer checks needing CI/Unity/PG context")
	flag.StringVar(&f.unityResultsDir, "unity-results-dir", "", "directory of Unity test-results XML")
	flag.StringVar(&f.reportOut, "report-out", "", "verify-report.json path")
	flag.StringVar(&f.mergeDir, "merge-reports", "", "merge linux+windows verify reports into a manifest")
	flag.BoolVar(&f.printHash, "print-source-tree-hash", false, "print the ADR-0057 source tree hash and exit")
	flag.StringVar(&f.taskID, "task", "", "task id for merge mode (e.g. IMP-000)")
	flag.BoolVar(&f.planUnity, "plan-unity", false, "print the Unity test modes whose Q3 gate is active (GITHUB_OUTPUT lines) and exit")
	flag.StringVar(&f.phase, "phase", phaseFull, "verifier phase: empty (all gates), pre-unity or unity (ADR-0077)")
	flag.StringVar(&f.preReport, "pre-report", "", "pre-unity phase output consumed by -phase unity")
	flag.Parse()

	switch f.phase {
	case phaseFull, phasePreUnity, phaseUnity:
	default:
		fatal(fmt.Errorf("unknown -phase %q (want empty, %s or %s)", f.phase, phasePreUnity, phaseUnity))
	}

	cwd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	root, err := gates.RepoRoot(cwd)
	if err != nil {
		fatal(err)
	}

	switch {
	case f.printHash:
		h, err := gates.SourceTreeHash(root)
		if err != nil {
			fatal(err)
		}
		fmt.Println(h)
	case f.planUnity:
		if err := planUnity(root); err != nil {
			fatal(err)
		}
	case f.mergeDir != "":
		if err := mergeMode(root, f); err != nil {
			fatal(err)
		}
	default:
		os.Exit(run(root, f))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "verify: "+err.Error())
	os.Exit(2)
}

// mergeMode builds the schema-v2 manifest from
// <mergeDir>/{linux,windows}-verify-report.json into
// docs/10_implementation/evidence/<task>/manifest.json.
func mergeMode(root string, f cliFlags) error {
	reports := map[string]gates.VerifyReport{}
	for _, osName := range []string{"linux", "windows"} {
		data, err := os.ReadFile(filepath.Join(f.mergeDir, osName+"-verify-report.json"))
		if err != nil {
			return fmt.Errorf("read %s report: %w", osName, err)
		}
		var rep gates.VerifyReport
		if err := json.Unmarshal(data, &rep); err != nil {
			return fmt.Errorf("parse %s report: %w", osName, err)
		}
		reports[osName] = rep
	}
	hash, err := gates.SourceTreeHash(root)
	if err != nil {
		return err
	}
	runID := os.Getenv("GITHUB_RUN_ID")
	attempt, _ := strconv.Atoi(os.Getenv("GITHUB_RUN_ATTEMPT"))
	taskID := regexp.MustCompile(`IMP-[0-9]+`).FindString(f.taskID)
	if taskID == "" {
		// spec/, ops/, revert/ and other non-task branches carry no IMP id;
		// the PR is not a task evidence run, so there is no manifest to merge.
		fmt.Fprintf(os.Stderr, "verify: %q names no IMP task; evidence manifest skipped\n", f.taskID)
		return nil
	}
	m, err := gates.MergeReports(taskID, hash, runID, attempt, reports)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	dest := filepath.Join(root, "docs/10_implementation/evidence", taskID, "manifest.json")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, append(out, '\n'), 0o644)
}

// activation returns the gate-activation predicate (owner task DONE on main
// or in the head) and whether the PR is status-only.
func activation(root string, e *gates.Env) (func(string) bool, bool, error) {
	packets, _, err := gates.ParseTaskQueue(root)
	if err != nil {
		return nil, false, err
	}
	headStatus := map[string]string{}
	for _, p := range packets {
		headStatus[p.ID] = p.Status
	}
	mainStatus := headStatus
	if e.IsPR() && e.BaseSHA != "" {
		mainStatus = map[string]string{}
		if data, err := gates.RefFile(root, e.BaseSHA, "docs/10_implementation/task_queue.md"); err == nil {
			if bps, _, err := gates.ParseTaskQueueText(data); err == nil {
				for _, p := range bps {
					mainStatus[p.ID] = p.Status
				}
			}
		}
	}
	required := func(owner string) bool {
		return mainStatus[owner] == "DONE" || headStatus[owner] == "DONE"
	}

	var statusOnly bool
	if e.HasPRContext() {
		if cl, err := gates.ClassifyPR(root, e); err == nil {
			statusOnly = cl.StatusOnly
		}
	}
	return required, statusOnly, nil
}

// planUnity prints the Unity test modes whose Q3 gate is active as
// GITHUB_OUTPUT lines (ADR-0074).
func planUnity(root string) error {
	required, statusOnly, err := activation(root, gates.LoadEnv(false, ""))
	if err != nil {
		return fmt.Errorf("parse task queue: %w", err)
	}
	fmt.Print(gates.PlanUnityModes(required, statusOnly).GithubOutput())
	return nil
}

// gateSet accumulates gates under the shared activation rules: a gate whose
// owner task is not DONE reports SKIP(owner-not-done), and a status-only PR
// runs Q0 only.
type gateSet struct {
	required   func(string) bool
	statusOnly bool
	list       []gates.Gate
}

func (s *gateSet) add(id, title string, owners []string, checks []gates.Check) {
	if owner := gates.FirstUnmetOwner(owners, s.required); owner != "" && id != "Q0" {
		checks = []gates.Check{gates.SkipOwnerNotDone(id, owner)}
	}
	if s.statusOnly && id != "Q0" {
		checks = []gates.Check{gates.SkipStatusOnly(id)}
	}
	s.list = append(s.list, gates.Gate{ID: id, Title: title, Checks: checks})
}

// active reports whether a gate owned by owner executes (not skipped).
func (s *gateSet) active(owner string) bool {
	return s.required(owner) && !s.statusOnly
}

// gatesBeforeUnity returns Q0, Q1, Q2 and Q3 (Go) — the gates reported ahead
// of the Unity gates. None of them reads Unity results.
func gatesBeforeUnity(root string, e *gates.Env, required func(string) bool, statusOnly bool, rep *gates.RunReport) []gates.Gate {
	s := &gateSet{required: required, statusOnly: statusOnly}
	s.add("Q0", "Task/spec integrity", []string{"IMP-000"}, gates.CheckQ0(root, e))

	var q1 []gates.Check
	if s.active("IMP-000") {
		q1 = gates.CheckQ1(root, e)
	}
	s.add("Q1", "Version reproducibility", []string{"IMP-000"}, q1)

	var q2 []gates.Check
	if s.active("IMP-061") {
		q2 = gates.CheckQ2(root, e, "")
	}
	s.add("Q2", "Code generation drift", []string{"IMP-061"}, q2)

	var q3Go []gates.Check
	if s.active("IMP-000") {
		q3Go = gates.CheckQ3Go(root, e, "", e.RunnerOS == "Linux", rep)
	}
	s.add("Q3", "Test suites (Go)", []string{"IMP-000"}, q3Go)
	return s.list
}

// unityGates returns the Q3 Unity EditMode/PlayMode gates, the only gates
// that read the Unity job's results.
func unityGates(root string, e *gates.Env, required func(string) bool, statusOnly bool) []gates.Gate {
	s := &gateSet{required: required, statusOnly: statusOnly}

	// ADR-0073: the workflow may skip Unity on a PR whose diff touches no
	// Unity-relevant path; the verifier re-derives the diff and fails the
	// Unity gates when the declared skip is not legitimate.
	unitySkip, unityProblem := false, ""
	if declared := os.Getenv("THINHTHAN_UNITY_SCOPE"); declared == gates.UnityScopeNoClientChange {
		var files []string
		var ferr error
		if e.HasPRContext() {
			files, ferr = gates.DiffFiles(root, e.BaseSHA, e.HeadSHA)
		} else {
			ferr = fmt.Errorf("PR context absent")
		}
		unitySkip, unityProblem = gates.ResolveUnityScope(e, declared, files, ferr)
	}
	unityChecks := func(suite string) []gates.Check {
		id := "Q3.unity." + strings.ToLower(suite)
		switch {
		case unityProblem != "":
			return []gates.Check{gates.Fail(id, unityProblem)}
		case unitySkip:
			return []gates.Check{gates.SkipNoClientChange(id)}
		}
		return gates.CheckUnityResults(e.UnityResultsDir, "Q3", suite, e)
	}

	var q3Edit, q3Play []gates.Check
	if s.active(gates.UnityEditModeOwner) {
		q3Edit = unityChecks("EditMode")
	}
	if s.active(gates.UnityPlayModeOwner) {
		q3Play = unityChecks("PlayMode")
	}
	s.add("Q3", "Test suites (Unity EditMode)", []string{gates.UnityEditModeOwner}, q3Edit)
	s.add("Q3", "Test suites (Unity PlayMode)", []string{gates.UnityPlayModeOwner}, q3Play)
	return s.list
}

// gatesAfterUnity returns Q4, Q5 and Q6 — reported after the Unity gates.
// None of them reads Unity results.
func gatesAfterUnity(root string, e *gates.Env, required func(string) bool, statusOnly bool) []gates.Gate {
	s := &gateSet{required: required, statusOnly: statusOnly}

	var q4Base, q4Client []gates.Check
	if s.active("IMP-000") {
		q4Base = gates.CheckQ4(root, e)
	}
	if s.active("IMP-083") {
		q4Client = gates.CheckQ4Client(root, e)
	}
	s.add("Q4", "Architecture conformance (base)", []string{"IMP-000"}, q4Base)
	s.add("Q4", "Architecture conformance (client API fence)", []string{"IMP-083"}, q4Client)

	q5Owners := []string{"IMP-005", "IMP-003", "IMP-004"}
	q5Any := false
	for _, o := range q5Owners {
		if required(o) {
			q5Any = true
		}
	}
	var q5 []gates.Check
	if q5Any && !statusOnly {
		q5 = gates.CheckQ5(root, e, "")
	}
	s.add("Q5", "Data/content integrity", q5Owners, q5)

	var q6 []gates.Check
	if s.active("IMP-000") {
		q6 = gates.CheckQ6(root, e)
	}
	s.add("Q6", "Evidence/cleanliness", []string{"IMP-000"}, q6)
	return s.list
}

// spliceGates returns the canonical report order: gates before Unity, the
// Unity gates, then the gates after Unity.
func spliceGates(before, unity, after []gates.Gate) []gates.Gate {
	out := make([]gates.Gate, 0, len(before)+len(unity)+len(after))
	out = append(out, before...)
	out = append(out, unity...)
	return append(out, after...)
}

// preUnityReport is the intermediate file written by -phase pre-unity and
// consumed by -phase unity. It never leaves the runner (RUNNER_TEMP).
type preUnityReport struct {
	Schema      string       `json:"schema"`
	Head        string       `json:"head"`
	Commands    []string     `json:"commands"`
	BeforeUnity []gates.Gate `json:"before_unity"`
	AfterUnity  []gates.Gate `json:"after_unity"`
}

// readPreReport loads a pre-unity phase file and rejects one that was not
// produced by this verifier for the same checked-out commit.
func readPreReport(path, head string) (*preUnityReport, error) {
	if path == "" {
		return nil, fmt.Errorf("no -pre-report given")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p preUnityReport
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if p.Schema != preUnitySchema {
		return nil, fmt.Errorf("%s: schema %q != %q", path, p.Schema, preUnitySchema)
	}
	if head == "" || p.Head != head {
		return nil, fmt.Errorf("%s: produced for HEAD %q, checkout is %q", path, p.Head, head)
	}
	if len(p.BeforeUnity) == 0 || len(p.AfterUnity) == 0 {
		return nil, fmt.Errorf("%s: incomplete gate set", path)
	}
	return &p, nil
}

// headCommit returns the checked-out commit; a pre-unity file is only valid
// for the tree it was computed on.
func headCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// run executes the gate set of the requested phase and writes its report.
func run(root string, f cliFlags) int {
	e := gates.LoadEnv(f.localDefer, f.unityResultsDir)
	rep := &gates.RunReport{}

	required, statusOnly, err := activation(root, e)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify: parse task queue: "+err.Error())
		return 2
	}

	if f.phase == phasePreUnity {
		return writePreUnity(root, f, e, required, statusOnly, rep)
	}

	var before, after []gates.Gate
	commands := []string(nil)
	if f.phase == phaseUnity {
		if p, perr := readPreReport(f.preReport, headCommit(root)); perr == nil {
			before, after, commands = p.BeforeUnity, p.AfterUnity, p.Commands
		} else {
			// A missing or unusable pre-unity file never weakens the
			// verdict: the phase falls back to running every gate.
			fmt.Fprintf(os.Stderr, "verify: pre-unity report unusable (%v); running the full gate set\n", perr)
		}
	}
	if before == nil {
		before = gatesBeforeUnity(root, e, required, statusOnly, rep)
		after = gatesAfterUnity(root, e, required, statusOnly)
		commands = rep.Commands
	}
	gs := spliceGates(before, unityGates(root, e, required, statusOnly), after)

	failed := false
	for _, g := range gs {
		if g.Status() == gates.StatusFail {
			failed = true
		}
	}
	result := "PASSED"
	if failed {
		result = "FAILED"
	}

	report := gates.VerifyReport{
		Schema:   "verify-report-v1",
		Result:   result,
		Go:       os.Getenv("THINHTHAN_GO_VERSION"),
		RanAt:    time.Now().UTC().Format(time.RFC3339),
		OS:       runnerOS(e),
		Commands: commands,
		Gates:    gs,
	}
	out, _ := json.MarshalIndent(report, "", "  ")
	dest := f.reportOut
	if dest == "" {
		dest = filepath.Join(root, "verify-report.json")
	}
	if err := os.WriteFile(dest, append(out, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "verify: write report: "+err.Error())
		return 2
	}

	printGates(gs)
	fmt.Printf("verify: %s (report %s)\n", result, dest)
	if failed {
		return 1
	}
	return 0
}

// writePreUnity runs the Unity-independent gates and writes the intermediate
// file. It exits 0 whenever the file was written: the verdict belongs to the
// final `-phase unity` run, which folds these results in.
func writePreUnity(root string, f cliFlags, e *gates.Env, required func(string) bool, statusOnly bool, rep *gates.RunReport) int {
	if f.reportOut == "" {
		fmt.Fprintln(os.Stderr, "verify: -phase pre-unity requires -report-out")
		return 2
	}
	p := preUnityReport{
		Schema:      preUnitySchema,
		Head:        headCommit(root),
		BeforeUnity: gatesBeforeUnity(root, e, required, statusOnly, rep),
		AfterUnity:  gatesAfterUnity(root, e, required, statusOnly),
	}
	p.Commands = rep.Commands
	out, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(f.reportOut, append(out, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "verify: write pre-unity report: "+err.Error())
		return 2
	}
	printGates(spliceGates(p.BeforeUnity, nil, p.AfterUnity))
	fmt.Printf("verify: pre-unity phase done (%s); the final verdict comes from -phase unity\n", f.reportOut)
	return 0
}

func printGates(gs []gates.Gate) {
	for _, g := range gs {
		fmt.Printf("== %s %s: %s\n", g.ID, g.Title, g.Status())
		for _, c := range g.Checks {
			detail := ""
			if c.Detail != "" {
				detail = " — " + c.Detail
			}
			fmt.Printf("   %-6s %s%s\n", c.Status, c.ID, detail)
		}
	}
}

func runnerOS(e *gates.Env) string {
	if e.RunnerOS == "Windows" {
		return "windows"
	}
	return "linux"
}
