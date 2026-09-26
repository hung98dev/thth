package gates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CheckQ3Go — the Go portion of Q3: gofmt, vet, test, build, staticcheck.
// `-race` runs on the Linux job only (ADR-0072): the caller passes
// racePackages for sim|edge|durable|global on Linux, nil on Windows.
func CheckQ3Go(root string, e *Env, goPath string, race bool, rep *RunReport) []Check {
	const gate = "Q3"
	id := func(s string) string { return gate + "." + s }
	var checks []Check
	run := func(label string, argv ...string) (string, error) {
		rep.Commands = append(rep.Commands, label)
		return goTool(root, goPath, argv...)
	}

	out, err := run("go -C server build ./...", "build", "./...")
	if err != nil {
		checks = append(checks, Fail(id("go.build"), out))
	} else {
		checks = append(checks, Pass(id("go.build"), ""))
	}

	out, err = run("go -C server test ./...", "test", "./...", "-count=1")
	if err != nil {
		checks = append(checks, Fail(id("go.test"), out))
	} else {
		checks = append(checks, Pass(id("go.test"), ""))
	}

	out, err = run("go -C server vet ./...", "vet", "./...")
	if err != nil {
		checks = append(checks, Fail(id("go.vet"), out))
	} else {
		checks = append(checks, Pass(id("go.vet"), ""))
	}

	// gofmt drift check (list mode only — never rewrite in CI).
	out, err = runCmd(root, 2*time.Minute, "gofmt", "-l", filepath.Join(root, "server"))
	if err != nil {
		checks = append(checks, Fail(id("go.fmt"), err.Error()))
	} else if out != "" {
		checks = append(checks, Fail(id("go.fmt"), "gofmt drift: "+out))
	} else {
		checks = append(checks, Pass(id("go.fmt"), ""))
	}

	if _, lerr := exec.LookPath("staticcheck"); lerr != nil {
		checks = append(checks, e.missingCheck(id("go.staticcheck"), "staticcheck not on PATH"))
	} else {
		out, err = runCmd(filepath.Join(root, "server"), 5*time.Minute, "staticcheck", "./...")
		if err != nil || out != "" {
			checks = append(checks, Fail(id("go.staticcheck"), out))
		} else {
			checks = append(checks, Pass(id("go.staticcheck"), ""))
		}
	}

	// -race on Linux only, for the package prefixes the ratchet names.
	if race {
		racePkgs := raceTargets(root)
		if len(racePkgs) > 0 {
			out, err = run("go -C server test -race <sim|edge|durable|global>",
				append([]string{"test", "-race", "-count=1"}, racePkgs...)...)
			if err != nil {
				checks = append(checks, Fail(id("go.race"), out))
			} else {
				checks = append(checks, Pass(id("go.race"), strings.Join(racePkgs, " ")))
			}
		} else {
			checks = append(checks, Pass(id("go.race"), "no sim|edge|durable|global packages yet"))
		}
	} else {
		checks = append(checks, Pass(id("go.race"), "not the Linux job"))
	}

	// Allocation budgets: non-race pass + report-only 200x benchmarks once
	// capacity.md § Hot-Path packages exist (HOT-* owners wire them later).
	benchPkgs := benchTargets(root)
	if len(benchPkgs) > 0 {
		out, err = run("go -C server test -benchmem -bench=. -benchtime=200x <hot-path>",
			append([]string{"test", "-run", "^$", "-benchmem", "-bench=.", "-benchtime=200x"}, benchPkgs...)...)
		if err != nil {
			checks = append(checks, Fail(id("go.alloc_budget"), out))
		} else {
			checks = append(checks, Pass(id("go.alloc_budget"), "report-only benchmarks ran"))
		}
	} else {
		checks = append(checks, Pass(id("go.alloc_budget"), "no hot-path benchmark packages yet"))
	}

	return checks
}

// raceTargets lists ./internal/{sim,edge,durable,global}/... package patterns
// that exist on disk.
func raceTargets(root string) []string {
	var out []string
	for _, d := range []string{"sim", "edge", "durable", "global"} {
		full := filepath.Join(root, "server", "internal", d)
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			out = append(out, "./internal/"+d+"/...")
		}
	}
	return out
}

// benchTargets returns package patterns containing *_test.go files with
// Benchmark functions under hot-path candidate dirs (capacity.md § Hot-Path).
func benchTargets(root string) []string {
	var out []string
	for _, d := range []string{"sim", "edge", "durable", "global", "core"} {
		dir := filepath.Join(root, "server", "internal", d)
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		if dirHasBenchmark(dir) {
			out = append(out, "./internal/"+d+"/...")
		}
	}
	return out
}

var benchRe = regexp.MustCompile(`func Benchmark[A-Za-z0-9_]*\s*\(`)

func dirHasBenchmark(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, "_test.go") || found {
			return nil
		}
		data, err := os.ReadFile(p)
		if err == nil && benchRe.Match(data) {
			found = true
		}
		return nil
	})
	return found
}

// unityResultsRe finds totals in Unity Test-Results XML
// (<test-run ... total="N" passed="P" failed="F" skipped="S" ...>).
var (
	testRunRe = regexp.MustCompile(`<test-run[^>]*`)
	attrInt   = regexp.MustCompile(`(total|passed|failed|skipped|inconclusive)="(\d+)"`)
)

type unityTotals struct {
	total, passed, failed, skipped int
	file                           string
}

// CheckUnityResults parses Unity EditMode/PlayMode results XML in dir. Empty
// suites (0/0) are a pass with an explicit note; missing dir defers/fails per
// context.
func CheckUnityResults(dir, gate, suite string, e *Env) []Check {
	id := gate + ".unity." + strings.ToLower(suite)
	if dir == "" {
		return []Check{e.missingCheck(id, "no Unity results dir provided")}
	}
	files, err := filepath.Glob(filepath.Join(dir, "*"+suite+"*.xml"))
	if err != nil || len(files) == 0 {
		// Fall back to any results XML when suite naming differs.
		files, _ = filepath.Glob(filepath.Join(dir, "*.xml"))
	}
	if len(files) == 0 {
		return []Check{e.missingCheck(id, "no Unity results XML in "+dir)}
	}
	var checks []Check
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			checks = append(checks, Fail(id, "unreadable "+f))
			continue
		}
		m := testRunRe.Find(data)
		if m == nil {
			checks = append(checks, Fail(id, "no <test-run> in "+filepath.Base(f)))
			continue
		}
		var t unityTotals
		t.file = filepath.Base(f)
		for _, a := range attrInt.FindAllStringSubmatch(string(m), -1) {
			v, _ := strconv.Atoi(a[2])
			switch a[1] {
			case "total":
				t.total += v
			case "passed":
				t.passed += v
			case "failed":
				t.failed += v
			case "skipped":
				t.skipped += v
			}
		}
		if t.failed > 0 {
			checks = append(checks, Fail(id, fmt.Sprintf("%s: %d failed of %d", t.file, t.failed, t.total)))
		} else if t.total == 0 {
			checks = append(checks, Pass(id, t.file+": empty suite (0 tests)"))
		} else {
			checks = append(checks, Pass(id, fmt.Sprintf("%s: %d/%d passed (%d skipped)", t.file, t.passed, t.total, t.skipped)))
		}
	}
	return checks
}
