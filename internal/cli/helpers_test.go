package cli

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitInitFixture turns dir into a self-contained git repo with one commit.
func gitInitFixture(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{
			"-c", "user.email=fixture@example.com",
			"-c", "user.name=fixture",
			"-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "fixture", "--allow-empty",
		},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
		}
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copyDir %s -> %s: %v", src, dst, err)
	}
}

// prepareGoFixture copies the codegraph/testdata/goproj fixture into a fresh
// git-initialized temp directory and returns its absolute path. CLI tests use
// this together with a separate temp cache dir via --cache-dir.
func prepareGoFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("..", "..", "codegraph", "testdata", "goproj")
	copyDir(t, src, dst)
	gitInitFixture(t, dst)
	return dst
}

// runCLI executes the root command with the given args, capturing stdout.
// Stderr is discarded; tests assert on stdout / err.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return outBuf.String(), err
}

// buildFixture runs `plexus build` against the fixture so subsequent commands
// hit the cache. Returns the cache dir.
func buildFixture(t *testing.T, repo string) string {
	t.Helper()
	cacheDir := t.TempDir()
	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "build")
	if err != nil {
		t.Fatalf("build failed: %v (stdout=%s)", err, stdout)
	}
	return cacheDir
}
