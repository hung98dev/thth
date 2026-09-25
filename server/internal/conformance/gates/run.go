package gates

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunReport records what cmd/verify executed for the verify-report.
type RunReport struct {
	Commands []string
}

// runCmd executes argv in dir with a timeout and returns output.
func runCmd(dir string, timeout time.Duration, argv ...string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if timeout > 0 {
		wait := make(chan error, 1)
		go func() { wait <- cmd.Run() }()
		select {
		case err := <-wait:
			return strings.TrimSpace(buf.String()), err
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
			return strings.TrimSpace(buf.String()), fmt.Errorf("timeout after %s", timeout)
		}
	}
	return strings.TrimSpace(buf.String()), cmd.Run()
}

// goTool runs `go -C server ...` inside the repo; goPath is the pinned
// toolchain binary or "go" when deferred locally.
func goTool(root, goPath string, args ...string) (string, error) {
	if goPath == "" {
		goPath = "go"
	}
	full := append([]string{goPath, "-C", filepath.Join(root, "server")}, args...)
	return runCmd(root, 15*time.Minute, full...)
}

// relSlash normalizes a path to repo-relative slash form for reports.
func relSlash(p string) string {
	return filepath.ToSlash(p)
}
