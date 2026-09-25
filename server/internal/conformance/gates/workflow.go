package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// WorkflowJob is a parsed job of verify.yml: its runs-on label, step list
// (in order) and whether a job-level if exists.
type WorkflowJob struct {
	Name     string
	RunsOn   string
	Steps    []WorkflowStep
	HasJobIf bool
	If       string
}

// WorkflowStep is one step of a job: name, uses, run body, shell, if.
type WorkflowStep struct {
	Name  string
	Uses  string
	Run   string
	Shell string
	If    string
	Index int
}

// WorkflowFile is a minimal structural parse of a GitHub workflow YAML —
// enough for the conformance tests (jobs/steps ordering, uses/run/shell/if
// fields, runs-on). It is NOT a general YAML parser.
type WorkflowFile struct {
	Jobs     []WorkflowJob
	On       string
	Raw      string
	Triggers map[string]bool
}

var (
	jobHeaderRe = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	runsOnRe    = regexp.MustCompile(`^    runs-on:\s*(.+)$`)
	jobIfRe     = regexp.MustCompile(`^    if:\s*(.+)$`)
	usesRe      = regexp.MustCompile(`^        uses:\s*(.+)$`)
	runRe       = regexp.MustCompile(`^        run:\s*(.+)$`)
	shellRe     = regexp.MustCompile(`^        shell:\s*(.+)$`)
	stepIfRe    = regexp.MustCompile(`^        if:\s*(.+)$`)
	stepNameRe  = regexp.MustCompile(`^      -\s+name:\s*(.+)$`)
)

// ParseWorkflow reads a workflow file into a structural model. Returns an
// error when it cannot find the jobs section.
func ParseWorkflow(path string) (*WorkflowFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := string(data)
	wf := &WorkflowFile{Raw: raw, Triggers: map[string]bool{}}
	lines := strings.Split(raw, "\n")

	inJobs := false
	var cur *WorkflowJob
	var curStep *WorkflowStep
	var runIndent = -1
	for _, line := range lines {
		// triggers: "on:" block, lines "  pull_request:", "  push:", etc.
		trim := strings.TrimSpace(line)
		if trim == "on:" {
			wf.On += "on:"
			continue
		}
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trim, ":") &&
			(trim == "pull_request:" || trim == "pull_request_target:" || trim == "push:" ||
				trim == "workflow_dispatch:" || trim == "merge_group:" || trim == "schedule:") {
			wf.Triggers[strings.TrimSuffix(trim, ":")] = true
			continue
		}
		if trim == "jobs:" {
			inJobs = true
			continue
		}
		if !inJobs {
			continue
		}
		if m := jobHeaderRe.FindStringSubmatch(line); m != nil {
			wf.Jobs = append(wf.Jobs, WorkflowJob{Name: m[1]})
			cur = &wf.Jobs[len(wf.Jobs)-1]
			curStep = nil
			runIndent = -1
			continue
		}
		if cur == nil {
			continue
		}
		if m := runsOnRe.FindStringSubmatch(line); m != nil {
			cur.RunsOn = strings.Trim(m[1], `"'`)
			continue
		}
		if m := jobIfRe.FindStringSubmatch(line); m != nil {
			cur.HasJobIf = true
			cur.If = m[1]
			continue
		}
		if m := stepNameRe.FindStringSubmatch(line); m != nil {
			cur.Steps = append(cur.Steps, WorkflowStep{Name: strings.Trim(m[1], `"'`), Index: len(cur.Steps)})
			curStep = &cur.Steps[len(cur.Steps)-1]
			runIndent = -1
			continue
		}
		if curStep == nil {
			continue
		}
		if m := usesRe.FindStringSubmatch(line); m != nil {
			curStep.Uses = strings.Trim(m[1], `"'`)
			continue
		}
		if m := runRe.FindStringSubmatch(line); m != nil {
			curStep.Run = m[1]
			// Block scalar indicators: |, > and their chomping variants.
			if strings.HasPrefix(curStep.Run, "|") || strings.HasPrefix(curStep.Run, ">") {
				runIndent = -2 // capture following indented lines
			}
			continue
		}
		if runIndent == -2 && (strings.HasPrefix(line, "          ") || trim == "") {
			curStep.Run += "\n" + line
			continue
		}
		runIndent = -1
		if m := shellRe.FindStringSubmatch(line); m != nil {
			curStep.Shell = strings.Trim(m[1], `"'`)
			continue
		}
		if m := stepIfRe.FindStringSubmatch(line); m != nil {
			curStep.If = m[1]
			continue
		}
	}
	return wf, nil
}

// JobByName returns the job named name, or nil.
func (w *WorkflowFile) JobByName(name string) *WorkflowJob {
	for i := range w.Jobs {
		if w.Jobs[i].Name == name {
			return &w.Jobs[i]
		}
	}
	return nil
}

// VerifyWorkflow loads .github/workflows/verify.yml.
func VerifyWorkflow(root string) (*WorkflowFile, error) {
	p := filepath.Join(root, ".github", "workflows", "verify.yml")
	wf, err := ParseWorkflow(p)
	if err != nil {
		return nil, fmt.Errorf("verify.yml: %w", err)
	}
	if len(wf.Jobs) == 0 {
		return nil, fmt.Errorf("verify.yml: no jobs parsed")
	}
	return wf, nil
}
