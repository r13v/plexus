package codegraph

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBuildFromRepo_Fresh(t *testing.T) {
	repo := prepareGoproj(t)
	cacheDir := t.TempDir()

	g, cached, err := BuildFromRepo(repo, "abc123", cacheDir)
	if err != nil {
		t.Fatalf("BuildFromRepo failed: %v", err)
	}
	if cached {
		t.Fatal("expected fresh build, got cached")
	}

	if len(g.Nodes) == 0 {
		t.Fatal("expected nodes, got 0")
	}
	if len(g.Edges) == 0 {
		t.Fatal("expected edges, got 0")
	}
	if len(g.Communities) == 0 {
		t.Fatal("expected communities, got 0")
	}

	cachePath, err := CachePath(repo, cacheDir)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file not written at %s: %v", cachePath, err)
	}
}

func TestBuildFromRepo_CacheHit(t *testing.T) {
	repo := prepareGoproj(t)
	cacheDir := t.TempDir()

	g1, _, err := BuildFromRepo(repo, "sha_first", cacheDir)
	if err != nil {
		t.Fatalf("first build failed: %v", err)
	}
	nodeCount := len(g1.Nodes)

	g2, cached, err := BuildFromRepo(repo, "sha_first", cacheDir)
	if err != nil {
		t.Fatalf("cache-hit build failed: %v", err)
	}
	if !cached {
		t.Fatal("expected cache hit on same SHA")
	}
	if len(g2.Nodes) != nodeCount {
		t.Errorf("cache hit returned %d nodes, want %d", len(g2.Nodes), nodeCount)
	}

	cachePath, err := CachePath(repo, cacheDir)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	_, headSHA, err := LoadGraph(cachePath)
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if headSHA != "sha_first" {
		t.Errorf("cache HEAD = %q, want %q", headSHA, "sha_first")
	}
}

func TestBuildFromRepo_CacheMissOnNewSHA(t *testing.T) {
	repo := prepareGoproj(t)
	cacheDir := t.TempDir()

	if _, _, err := BuildFromRepo(repo, "sha_old", cacheDir); err != nil {
		t.Fatalf("first build failed: %v", err)
	}

	g, cached, err := BuildFromRepo(repo, "sha_new", cacheDir)
	if err != nil {
		t.Fatalf("rebuild failed: %v", err)
	}
	if cached {
		t.Fatal("expected fresh build on new SHA")
	}
	if len(g.Nodes) == 0 {
		t.Fatal("rebuild produced 0 nodes")
	}

	cachePath, err := CachePath(repo, cacheDir)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	_, headSHA, err := LoadGraph(cachePath)
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if headSHA != "sha_new" {
		t.Errorf("cache HEAD = %q, want %q", headSHA, "sha_new")
	}
}

func TestBuildFromRepo_CacheWriteFailure(t *testing.T) {
	tmp := t.TempDir()
	srcMain := filepath.Join(tmp, "main.go")
	if err := os.WriteFile(srcMain, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	gitInitFixture(t, tmp)

	// Force the cache parent to be a regular file so MkdirAll fails.
	cacheDir := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(cacheDir, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	g, _, err := BuildFromRepo(tmp, "sha_readonly", cacheDir)
	if err != nil {
		t.Fatalf("BuildFromRepo should succeed even if cache write fails: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatal("expected nodes even with cache write failure")
	}
}

func TestBuildFromRepo_BrokenGoFile(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "broken.go"), []byte("package main\nfunc broken( {\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, tmp)

	g, _, err := BuildFromRepo(tmp, "sha_broken", t.TempDir())
	if err != nil {
		t.Fatalf("BuildFromRepo should succeed with broken syntax files: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatal("expected nodes from the valid file")
	}
}

func TestBuildFromRepo_GraphIntegrity(t *testing.T) {
	repo := prepareGoproj(t)

	g, _, err := BuildFromRepo(repo, "integrity_check", t.TempDir())
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	for _, e := range g.Edges {
		if _, ok := g.Nodes[e.Source]; !ok {
			t.Errorf("edge source %q not in nodes", e.Source)
		}
		if _, ok := g.Nodes[e.Target]; !ok {
			t.Errorf("edge target %q not in nodes", e.Target)
		}
	}

	for id := range g.Nodes {
		found := false
		for _, members := range g.Communities {
			if slices.Contains(members, id) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("node %q not assigned to any community", id)
		}
	}

	if len(g.Adj) == 0 && len(g.Edges) > 0 {
		t.Error("Adj map not populated")
	}
	if len(g.RevAdj) == 0 && len(g.Edges) > 0 {
		t.Error("RevAdj map not populated")
	}
}
