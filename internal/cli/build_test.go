package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/r13v/plexus/codegraph"
)

func TestBuildCmd_FreshAndCached(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := t.TempDir()

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "build")
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	if !strings.Contains(stdout, "nodes=") || !strings.Contains(stdout, "cached=false") {
		t.Fatalf("first build output unexpected: %q", stdout)
	}

	stdout2, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "build")
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	if !strings.Contains(stdout2, "cached=true") {
		t.Fatalf("expected cached=true on second build, got: %q", stdout2)
	}
}

// commitFile adds a file to the fixture repo and commits it, advancing HEAD.
func commitFile(t *testing.T, repo, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", name},
		{
			"-c", "user.email=fixture@example.com",
			"-c", "user.name=fixture",
			"-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "add " + name,
		},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

// TestStatsCmd_RebuildsOnStaleCache covers the README guarantee that non-build
// commands rebuild on demand when HEAD moves past the cached SHA.
func TestStatsCmd_RebuildsOnStaleCache(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	commitFile(t, repo, "extra.go", "package goproj\n\nfunc Extra() {}\n")

	cmd := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs([]string{"--repo", repo, "--cache-dir", cacheDir, "--format", "json", "stats"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats: %v (stderr=%s)", err, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "rebuilding") {
		t.Errorf("expected rebuild notice on stderr, got: %q", errBuf.String())
	}

	top, err := codegraph.GitToplevel(repo)
	if err != nil {
		t.Fatal(err)
	}
	cachePath, err := codegraph.CachePath(top, cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	_, cachedSHA, err := codegraph.LoadGraph(cachePath)
	if err != nil {
		t.Fatalf("load rebuilt cache: %v", err)
	}
	headSHA, err := codegraph.GitHead(top)
	if err != nil {
		t.Fatal(err)
	}
	if cachedSHA != headSHA {
		t.Errorf("cache SHA %s != HEAD %s after rebuild", cachedSHA, headSHA)
	}

	var payload struct {
		Stats struct {
			NodeCount int `json:"node_count"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(outBuf.Bytes(), &payload); err != nil {
		t.Fatalf("parse stats: %v", err)
	}
	if payload.Stats.NodeCount == 0 {
		t.Fatal("expected non-zero node count in rebuilt graph")
	}
}
