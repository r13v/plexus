package codegraph

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitInitFixture turns dir into a self-contained git repo and commits its
// current contents. WalkRepo discovers files via `git ls-files`, so every
// fixture used by tests must be a real git working tree.
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

// copyDir recursively copies src into dst, preserving file modes. Used to
// stage read-only testdata fixtures into a writable temp dir before
// gitInitFixture mutates them.
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

// prepareGoproj copies testdata/goproj into a fresh temp dir and turns it
// into a git repo. Returns the absolute path of the prepared fixture.
func prepareGoproj(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	copyDir(t, filepath.Join("testdata", "goproj"), dst)
	gitInitFixture(t, dst)
	return dst
}
