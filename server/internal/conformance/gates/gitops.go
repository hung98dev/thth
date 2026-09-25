package gates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// gitDir runs git in dir and returns trimmed stdout.
func gitDir(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// RepoRoot walks up from dir looking for the repo root (.git + docs/).
func RepoRoot(dir string) (string, error) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			_ = st
			return d, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("no .git found above %s", dir)
		}
		d = parent
	}
}

// DiffFiles returns the changed file list of refBase...refHead (relative
// slash paths) — merge-base semantics so main-side drift on a stale PR does
// not appear as phantom PR changes.
func DiffFiles(root, base, head string) ([]string, error) {
	out, err := gitDir(root, "diff", "--name-only", "--diff-filter=ACDMRT", base+"..."+head)
	if err != nil {
		return nil, fmt.Errorf("git diff %s...%s: %v", base, head, err)
	}
	return splitLines(out), nil
}

// RefFile reads a file's content at ref ("ref:path"); empty when absent.
func RefFile(root, ref, path string) (string, error) {
	out, err := gitDir(root, "show", ref+":"+path)
	if err != nil {
		return "", err
	}
	return out, nil
}

// removedLines returns lines deleted from path in base...head (merge-base
// semantics), for append-only control files.
func removedLines(root string, e *Env, path string) []string {
	out, err := gitDir(root, "diff", "-U0", e.BaseSHA+"..."+e.HeadSHA, "--", path)
	if err != nil {
		return nil
	}
	var removed []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed = append(removed, strings.TrimSpace(strings.TrimPrefix(line, "-")))
		}
	}
	return removed
}

// splitLines splits output on \n dropping empties.
func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
