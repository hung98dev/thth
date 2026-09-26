package gates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"thinhthan/internal/stackpin"
)

// CheckQ1 — Version reproducibility: exact native pins; no floating or
// unlisted dependency (delegates file checks to stackpin, adds the active
// toolchain's own version probe).
func CheckQ1(root string, e *Env) []Check {
	const gate = "Q1"
	id := func(s string) string { return gate + "." + s }
	var checks []Check

	pinFindings := stackpin.Run(root)
	var pinProblems []string
	for _, f := range pinFindings {
		for _, p := range f.Problems {
			pinProblems = append(pinProblems, f.ID+": "+p)
		}
	}
	if len(pinProblems) > 0 {
		checks = append(checks, Fail(id("pins"), strings.Join(pinProblems, "; ")))
	} else {
		checks = append(checks, Pass(id("pins"), "all pins equal technology_versions.md"))
	}

	// Active toolchain probes — a missing binary is a failure in CI and a
	// DEFERRED(local-missing) locally.
	goVer, goErr := runCmd(root, 60*time.Second, "go", "version")
	if goErr != nil {
		checks = append(checks, e.missingCheck(id("go.version"), "go toolchain not on PATH"))
	} else {
		want := "go" + stackpin.GoVersion + " "
		if !strings.Contains(goVer, want) {
			checks = append(checks, Fail(id("go.version"), fmt.Sprintf("%q is not the pinned %s", goVer, stackpin.GoVersion)))
		} else {
			checks = append(checks, Pass(id("go.version"), goVer))
		}
	}

	protocVer, protocErr := runCmd(root, 60*time.Second, "protoc", "--version")
	if protocErr != nil {
		checks = append(checks, e.missingCheck(id("protoc.version"), "protoc not on PATH"))
	} else {
		if !strings.Contains(protocVer, stackpin.Protoc) {
			checks = append(checks, Fail(id("protoc.version"), fmt.Sprintf("%q != protoc %s", protocVer, stackpin.Protoc)))
		} else {
			checks = append(checks, Pass(id("protoc.version"), protocVer))
		}
	}

	genVer, genErr := runCmd(root, 60*time.Second, "protoc-gen-go", "--version")
	if genErr != nil {
		checks = append(checks, e.missingCheck(id("protoc_gen_go.version"), "protoc-gen-go not on PATH"))
	} else {
		if !strings.Contains(genVer, stackpin.ProtocGenGo) {
			checks = append(checks, Fail(id("protoc_gen_go.version"), fmt.Sprintf("%q != %s", genVer, stackpin.ProtocGenGo)))
		} else {
			checks = append(checks, Pass(id("protoc_gen_go.version"), genVer))
		}
	}

	// staticcheck presence — version is asserted by the workflow install step;
	// Q1 records whether it can run (style gate runs it).
	if _, scErr := exec.LookPath("staticcheck"); scErr != nil {
		checks = append(checks, e.missingCheck(id("staticcheck.version"), "staticcheck not on PATH"))
	} else {
		out, _ := runCmd(root, 60*time.Second, "staticcheck", "-version")
		checks = append(checks, Pass(id("staticcheck.version"), out))
	}

	// Unity editor presence: in CI the pinned docker image supplies it; a
	// local run without a Unity install defers.
	if unityPath := os.Getenv("UNITY_EDITOR_PATH"); unityPath != "" {
		checks = append(checks, Pass(id("unity.editor"), unityPath))
	} else {
		checks = append(checks, e.missingCheck(id("unity.editor"), "no local Unity editor (CI uses the pinned unityci image)"))
	}

	return checks
}

// checkGeneratedDrift fails when gofmt/build/test left the tree dirty in
// ways the diff did not authorize — used by Q6 for generated outputs.
func untrackedUnder(root string, dirs ...string) []string {
	var out []string
	for _, d := range dirs {
		full := filepath.Join(root, filepath.FromSlash(d))
		if _, err := os.Stat(full); err != nil {
			continue
		}
		untracked, err := gitDir(root, "ls-files", "--others", "--exclude-standard", "--", d)
		if err != nil {
			continue
		}
		for _, l := range splitLines(untracked) {
			out = append(out, l)
		}
	}
	return out
}
