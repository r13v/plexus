package codegraph

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func gitInitFixtureWithFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	gitInitFixture(t, dir)
}

func TestGitHead(t *testing.T) {
	dir := t.TempDir()
	gitInitFixtureWithFiles(t, dir, map[string]string{"a.txt": "hi"})
	sha, err := GitHead(dir)
	if err != nil {
		t.Fatalf("GitHead: %v", err)
	}
	if len(sha) != 40 {
		t.Fatalf("expected 40-char SHA, got %q", sha)
	}
}

func TestGitHead_NotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := GitHead(dir); err == nil {
		t.Fatal("expected error in non-repo dir")
	}
}

func TestGitToplevel(t *testing.T) {
	dir := t.TempDir()
	gitInitFixtureWithFiles(t, dir, map[string]string{"a.txt": "hi"})
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	top, err := GitToplevel(sub)
	if err != nil {
		t.Fatalf("GitToplevel: %v", err)
	}
	// macOS /var symlinks to /private/var; resolve both for comparison.
	wantResolved, _ := filepath.EvalSymlinks(dir)
	gotResolved, _ := filepath.EvalSymlinks(top)
	if gotResolved != wantResolved {
		t.Fatalf("toplevel %q != %q", gotResolved, wantResolved)
	}
}

func TestGitToplevel_NotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := GitToplevel(dir); err == nil {
		t.Fatal("expected error outside repo")
	}
}

func TestGitLsFiles(t *testing.T) {
	dir := t.TempDir()
	gitInitFixtureWithFiles(t, dir, map[string]string{
		"a.txt":       "a",
		"sub/b.txt":   "b",
		".gitignore":  "ignored.txt\n",
		"ignored.txt": "skip me",
	})
	// untracked file (should still appear via -co)
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := GitLsFiles(dir)
	if err != nil {
		t.Fatalf("GitLsFiles: %v", err)
	}
	slices.Sort(files)
	want := []string{".gitignore", "a.txt", "c.txt", "sub/b.txt"}
	if !slices.Equal(files, want) {
		t.Fatalf("got %v, want %v", files, want)
	}
}

func TestGitLsFiles_NotARepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := GitLsFiles(dir); err == nil {
		t.Fatal("expected error in non-repo dir")
	}
}
