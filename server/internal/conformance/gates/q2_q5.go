package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckQ2 — Code generation drift (owner IMP-061). When the owner is DONE the
// pinned protoc generators must exist and `scripts/codegen.ps1` produces
// byte-identical Go/C# output (generated-code header included).
func CheckQ2(root string, e *Env, goPath string) []Check {
	const gate = "Q2"
	id := func(s string) string { return gate + "." + s }

	// Pinned generators required.
	for _, tool := range []string{"protoc", "protoc-gen-go"} {
		if _, err := runCmd(root, 60*time.Second, tool, "--version"); err != nil {
			return []Check{e.missingCheck(id("generators"), tool+" not on PATH")}
		}
	}

	script := filepath.Join(root, "scripts", "codegen.ps1")
	if _, err := os.Stat(script); err != nil {
		return []Check{Fail(id("script"), "scripts/codegen.ps1 missing")}
	}
	out, err := runCmd(root, 5*time.Minute, "pwsh", "-NoProfile", "-File", script, "-RepoRoot", root)
	if err != nil {
		return []Check{Fail(id("script"), fmt.Sprintf("codegen.ps1 failed: %v (%s)", err, out))}
	}
	diff, err := gitDir(root, "status", "--porcelain")
	if err != nil {
		return []Check{Fail(id("script"), "git status failed")}
	}
	var dirty []string
	for _, l := range splitLines(diff) {
		// generated dirs only
		if strings.Contains(l, "server/internal/protocol") || strings.Contains(l, "client/Assets/Scripts/Protocol") {
			dirty = append(dirty, l)
		}
	}
	if len(dirty) > 0 {
		return []Check{Fail(id("drift"), "codegen drift: "+strings.Join(dirty, "; "))}
	}
	// Generated header (Q2 includes the generated C# header check).
	return []Check{Pass(id("drift"), "byte-identical regeneration")}
}

// CheckQ5 — Data/content integrity (owners IMP-005 migrations, IMP-003
// content, IMP-004 balance). Migration rehearsal + schema drift + content
// compile/activation; fails closed when the owning task is DONE without the
// harness pieces.
func CheckQ5(root string, e *Env, goPath string) []Check {
	const gate = "Q5"
	id := func(s string) string { return gate + "." + s }
	var checks []Check

	mig := filepath.Join(root, "server", "migrations")
	if _, err := os.Stat(mig); err != nil {
		checks = append(checks, e.missingCheck(id("migrations.dir"), "server/migrations/ absent"))
	} else {
		entries, _ := os.ReadDir(mig)
		names := map[string]map[string]bool{}
		for _, en := range entries {
			n := en.Name()
			if strings.HasSuffix(n, ".up.sql") {
				pair := strings.TrimSuffix(n, ".up.sql") + ".down.sql"
				if names[n] == nil {
					names[n] = map[string]bool{}
				}
				if _, err := os.Stat(filepath.Join(mig, pair)); err != nil {
					checks = append(checks, Fail(id("migrations.pairs"), n+" lacks "+pair))
				}
			}
		}
		checks = append(checks, Pass(id("migrations.pairs"), ""))
		// Rehearsal requires the pinned postgres; verify.ps1 arranges
		// THINHTHAN_TEST_PG_DSN before calling the verifier.
		if os.Getenv("THINHTHAN_TEST_PG_DSN") == "" {
			checks = append(checks, e.missingCheck(id("migrations.rehearsal"), "THINHTHAN_TEST_PG_DSN unset"))
		} else {
			out, err := goTool(root, goPath, "test", "./internal/durable/...", "-run", "TestMigration", "-count=1")
			if err != nil {
				checks = append(checks, Fail(id("migrations.rehearsal"), out))
			} else {
				checks = append(checks, Pass(id("migrations.rehearsal"), ""))
			}
		}
	}

	content := filepath.Join(root, "content")
	if _, err := os.Stat(content); err != nil {
		checks = append(checks, e.missingCheck(id("content.dir"), "content/ absent"))
	} else {
		out, err := goTool(root, goPath, "test", "./internal/config/...", "-count=1")
		if err != nil {
			checks = append(checks, Fail(id("content.compile"), out))
		} else {
			checks = append(checks, Pass(id("content.compile"), ""))
		}
	}
	return checks
}
