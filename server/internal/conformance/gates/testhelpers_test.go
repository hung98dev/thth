package gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	gitT(t, dir, "init", "-q", "-b", "main")
	gitT(t, dir, "config", "user.email", "t@t")
	gitT(t, dir, "config", "user.name", "t")
	// No detached background gc/maintenance: it can still be writing
	// .git/objects when t.TempDir's RemoveAll cleanup runs.
	gitT(t, dir, "config", "gc.auto", "0")
	gitT(t, dir, "config", "maintenance.auto", "false")
	writeRepoFile(t, dir, "README.md", "init\n")
	commitAll(t, dir)
}

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, dir string) {
	t.Helper()
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "x")
}
