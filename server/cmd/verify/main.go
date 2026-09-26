// Command verify runs the Q0-Q6 conformance gates. verify.ps1 is the only
// caller; CI invokes it on both GitHub-hosted OSes and the evidence job merges
// the two per-job reports into the schema-v2 manifest.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"thinhthan/internal/conformance/gates"
)

type cliFlags struct {
	localDefer      bool
	unityResultsDir string
	reportOut       string
	mergeDir        string
	printHash       bool
	taskID          string
}

func main() {
	var f cliFlags
	flag.BoolVar(&f.localDefer, "local-defer", false, "defer checks needing CI/Unity/PG context")
	flag.StringVar(&f.unityResultsDir, "unity-results-dir", "", "directory of Unity test-results XML")
	flag.StringVar(&f.reportOut, "report-out", "", "verify-report.json path")
	flag.StringVar(&f.mergeDir, "merge-reports", "", "merge linux+windows verify reports into a manifest")
	flag.BoolVar(&f.printHash, "print-source-tree-hash", false, "print the ADR-0057 source tree hash and exit")
	flag.StringVar(&f.taskID, "task", "", "task id for merge mode (e.g. IMP-000)")
	flag.Parse()

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

// run executes the gate set and writes verify-report.json.
func run(root string, f cliFlags) int {
	e := gates.LoadEnv(f.localDefer, f.unityResultsDir)
	rep := &gates.RunReport{}

	packets, _, err := gates.ParseTaskQueue(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify: parse task queue: "+err.Error())
		return 2
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

	var gs []gates.Gate
	addGate := func(id, title string, owners []string, checks []gates.Check) {
		if owner := gates.FirstUnmetOwner(owners, required); owner != "" && id != "Q0" {
			checks = []gates.Check{gates.SkipOwnerNotDone(id, owner)}
		}
		if statusOnly && id != "Q0" {
			checks = []gates.Check{gates.SkipStatusOnly(id)}
		}
		gs = append(gs, gates.Gate{ID: id, Title: title, Checks: checks})
	}

	addGate("Q0", "Task/spec integrity", []string{"IMP-000"}, gates.CheckQ0(root, e))

	var q1 []gates.Check
	if required("IMP-000") && !statusOnly {
		q1 = gates.CheckQ1(root, e)
	}
	addGate("Q1", "Version reproducibility", []string{"IMP-000"}, q1)

	var q2 []gates.Check
	if required("IMP-061") && !statusOnly {
		q2 = gates.CheckQ2(root, e, "")
	}
	addGate("Q2", "Code generation drift", []string{"IMP-061"}, q2)

	var q3Go []gates.Check
	if required("IMP-000") && !statusOnly {
		q3Go = gates.CheckQ3Go(root, e, "", e.RunnerOS == "Linux", rep)
	}
	addGate("Q3", "Test suites (Go)", []string{"IMP-000"}, q3Go)

	var q3Edit, q3Play []gates.Check
	if required("IMP-000") && !statusOnly {
		q3Edit = gates.CheckUnityResults(e.UnityResultsDir, "Q3", "EditMode", e)
	}
	if required("IMP-065") && !statusOnly {
		q3Play = gates.CheckUnityResults(e.UnityResultsDir, "Q3", "PlayMode", e)
	}
	addGate("Q3", "Test suites (Unity EditMode)", []string{"IMP-000"}, q3Edit)
	addGate("Q3", "Test suites (Unity PlayMode)", []string{"IMP-065"}, q3Play)

	var q4Base, q4Client []gates.Check
	if required("IMP-000") && !statusOnly {
		q4Base = gates.CheckQ4(root, e)
	}
	if required("IMP-083") && !statusOnly {
		q4Client = gates.CheckQ4Client(root, e)
	}
	addGate("Q4", "Architecture conformance (base)", []string{"IMP-000"}, q4Base)
	addGate("Q4", "Architecture conformance (client API fence)", []string{"IMP-083"}, q4Client)

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
	addGate("Q5", "Data/content integrity", q5Owners, q5)

	var q6 []gates.Check
	if required("IMP-000") && !statusOnly {
		q6 = gates.CheckQ6(root, e)
	}
	addGate("Q6", "Evidence/cleanliness", []string{"IMP-000"}, q6)

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
		Commands: rep.Commands,
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
	fmt.Printf("verify: %s (report %s)\n", result, dest)
	if failed {
		return 1
	}
	return 0
}

func runnerOS(e *gates.Env) string {
	if e.RunnerOS == "Windows" {
		return "windows"
	}
	return "linux"
}
