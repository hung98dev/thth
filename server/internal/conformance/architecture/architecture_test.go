package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"thinhthan/internal/conformance/gates"
)

// ---------- fixtures ------------------------------------------------------

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := gates.RepoRoot(".")
	if err != nil {
		t.Skipf("not inside a git worktree: %v", err)
	}
	return root
}

func realGoFiles(t *testing.T) []string {
	t.Helper()
	files, err := repoFiles(repoRoot(t))
	if err != nil {
		t.Fatalf("repo files: %v", err)
	}
	var out []string
	for _, f := range files {
		if strings.HasPrefix(f, "server/") && strings.HasSuffix(f, ".go") {
			out = append(out, f)
		}
	}
	return out
}

// goFile renders a minimal compilable Go source importing the given paths.
func goFile(imports ...string) string {
	var b strings.Builder
	b.WriteString("package x\n\nimport (\n")
	for _, i := range imports {
		b.WriteString("\t\"" + i + "\"\n")
	}
	b.WriteString(")\n\nvar _ = 0\n")
	return b.String()
}

// ---------- tests ----------------------------------------------------------

func TestImportDirection(t *testing.T) {
	root := repoRoot(t)
	if got := ImportDirectionProblems(root, realGoFiles(t)); len(got) != 0 {
		t.Fatalf("real tree: %v", got)
	}
	tmp := t.TempDir()
	writeFile(t, tmp, "server/internal/sim/x.go", goFile("thinhthan/internal/edge"))
	writeFile(t, tmp, "server/internal/edge/y.go", goFile())
	files := []string{"server/internal/sim/x.go", "server/internal/edge/y.go"}
	if got := ImportDirectionProblems(tmp, files); len(got) == 0 {
		t.Fatal("sim -> edge import not reported")
	}
	// durable may own SQL imports: adding it must leave only the sim->edge hit
	writeFile(t, tmp, "server/internal/durable/d.go", goFile("database/sql"))
	files = append(files, "server/internal/durable/d.go")
	if got := ImportDirectionProblems(tmp, files); len(got) != 1 {
		t.Fatalf("durable sql import wrongly flagged: %v", got)
	}
	writeFile(t, tmp, "server/internal/sim/db.go", goFile("github.com/jackc/pgx/v5"))
	files = append(files, "server/internal/sim/db.go")
	if got := ImportDirectionProblems(tmp, files); len(got) != 2 {
		t.Fatalf("sim -> sql import not reported: %v", got)
	}
}

func TestOneProductionMain(t *testing.T) {
	root := repoRoot(t)
	if got := ProductionMainProblems(root, realGoFiles(t)); len(got) != 0 {
		t.Fatalf("real tree: %v", got)
	}
	tmp := t.TempDir()
	writeFile(t, tmp, "server/cmd/server/main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, tmp, "server/cmd/verify/main.go", "package main\n\nfunc main() {}\n")
	files := []string{"server/cmd/server/main.go", "server/cmd/verify/main.go"}
	if got := ProductionMainProblems(tmp, files); len(got) != 0 {
		t.Fatalf("allowed mains flagged: %v", got)
	}
	// a main.go under an internal package's own cmd/ is in scope ONLY for
	// server/cmd/ — package-internal helpers are allowed.
	writeFile(t, tmp, "server/internal/conformance/caching/cmd/cachemerge/main.go",
		"package main\n\nfunc main() {}\n")
	files = append(files, "server/internal/conformance/caching/cmd/cachemerge/main.go")
	if got := ProductionMainProblems(tmp, files); len(got) != 0 {
		t.Fatalf("internal helper main flagged: %v", got)
	}
	writeFile(t, tmp, "server/cmd/rogue/main.go", "package main\n\nfunc main() {}\n")
	files = append(files, "server/cmd/rogue/main.go")
	if got := ProductionMainProblems(tmp, files); len(got) != 1 {
		t.Fatalf("extra server/cmd main not reported: %v", got)
	}
}

func TestForbiddenDependencies(t *testing.T) {
	root := repoRoot(t)
	if got := ForbiddenDependencyProblems(root, realGoFiles(t)); len(got) != 0 {
		t.Fatalf("real tree: %v", got)
	}
	tmp := t.TempDir()
	writeFile(t, tmp, "server/internal/edge/h.go", goFile("github.com/gin-gonic/gin"))
	writeFile(t, tmp, "server/internal/sim/r.go", goFile("math/rand"))
	files := []string{"server/internal/edge/h.go", "server/internal/sim/r.go"}
	if got := ForbiddenDependencyProblems(tmp, files); len(got) < 2 {
		t.Fatalf("forbidden imports not reported: %v", got)
	}
	// go.mod require also fails
	writeFile(t, tmp, "server/go.mod", "module thinhthan\n\ngo 1.27.1\n\nrequire go.uber.org/zap v1.2.3\n")
	if got := ForbiddenDependencyProblems(tmp, []string{"server/go.mod"}); len(got) == 0 {
		t.Fatal("forbidden go.mod require not reported")
	}
	// math/rand/v2 is NOT forbidden by this rule
	tmp2 := t.TempDir()
	writeFile(t, tmp2, "server/internal/core/rng/r.go", goFile("math/rand/v2"))
	if got := ForbiddenDependencyProblems(tmp2, []string{"server/internal/core/rng/r.go"}); len(got) != 0 {
		t.Fatalf("math/rand/v2 wrongly forbidden: %v", got)
	}
}

func TestGeneratedBoundary(t *testing.T) {
	root := repoRoot(t)
	files := allFilesUnder(t, root, "server/internal/protocol/")
	cfiles := allFilesUnder(t, root, "client/Assets/Scripts/Protocol/")
	var inScope []string
	inScope = append(inScope, files...)
	inScope = append(inScope, cfiles...)
	if got := GeneratedBoundaryProblems(root, inScope); len(got) != 0 {
		t.Fatalf("real tree: %v", got)
	}
	tmp := t.TempDir()
	gen := "// Code generated by protoc-gen-go. DO NOT EDIT.\n// source: x.proto\n\npackage v1\n"
	writeFile(t, tmp, "server/internal/protocol/v1/x.pb.go", gen)
	writeFile(t, tmp, "server/internal/protocol/v1/manual.go", "package v1\n")
	if got := GeneratedBoundaryProblems(tmp, []string{
		"server/internal/protocol/v1/x.pb.go",
		"server/internal/protocol/v1/manual.go",
	}); len(got) == 0 {
		t.Fatal("hand-written file inside generated dir not reported")
	}
	// bad .pb.go without the generated header also fails
	tmp2 := t.TempDir()
	writeFile(t, tmp2, "server/internal/protocol/v1/y.pb.go", "package v1\n")
	if got := GeneratedBoundaryProblems(tmp2, []string{"server/internal/protocol/v1/y.pb.go"}); len(got) == 0 {
		t.Fatal(".pb.go without generated header not reported")
	}
	// client Protocol/ .cs without the generated header fails
	tmp3 := t.TempDir()
	writeFile(t, tmp3, "client/Assets/Scripts/Protocol/Msg.cs", "public sealed class Msg {}\n")
	if got := GeneratedBoundaryProblems(tmp3, []string{"client/Assets/Scripts/Protocol/Msg.cs"}); len(got) == 0 {
		t.Fatal("unmarked generated C# not reported")
	}
	hdr := "// <auto-generated> DO NOT EDIT </auto-generated>\n#nullable disable\n#pragma warning disable\npublic sealed class Msg {}\n"
	writeFile(t, tmp3, "client/Assets/Scripts/Protocol/Ok.cs", hdr)
	got := GeneratedBoundaryProblems(tmp3, []string{
		"client/Assets/Scripts/Protocol/Ok.cs",
		"client/Assets/Scripts/Protocol/Msg.cs",
	})
	if len(got) != 1 || !strings.Contains(got[0], "Msg.cs") {
		t.Fatalf("expected only Msg.cs flagged: %v", got)
	}
}

func allFilesUnder(t *testing.T, root, prefix string) []string {
	t.Helper()
	var out []string
	base := filepath.Join(root, filepath.FromSlash(prefix))
	_ = filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}
