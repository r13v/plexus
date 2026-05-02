package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPathCmd_FindsPath(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "path", "main()", "cleanup()")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if stdout == "" {
		t.Fatal("expected non-empty path output")
	}
}

func TestPathCmd_NoPath(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "path", "cleanup()", "main()")
	if err != nil {
		t.Fatalf("path (no-path case): %v", err)
	}
	if !strings.Contains(stdout, "no path found") {
		t.Fatalf("expected 'no path found' message, got: %q", stdout)
	}
}

func TestPathCmd_NoPathJSON(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "--format", "json", "path", "cleanup()", "main()")
	if err != nil {
		t.Fatalf("path json (no-path case): %v", err)
	}
	var payload struct {
		Kind string `json:"kind"`
		Path struct {
			Source struct {
				Label string `json:"label"`
			} `json:"source"`
			Target struct {
				Label string `json:"label"`
			} `json:"target"`
			Edges []any `json:"edges"`
		} `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("parse json: %v (raw=%q)", err, stdout)
	}
	if payload.Kind != "path" {
		t.Errorf("kind = %q, want path", payload.Kind)
	}
	if payload.Path.Source.Label != "cleanup()" || payload.Path.Target.Label != "main()" {
		t.Errorf("endpoints wrong: %+v", payload.Path)
	}
	if len(payload.Path.Edges) != 0 {
		t.Errorf("expected no edges, got %d", len(payload.Path.Edges))
	}
}

func TestPathCmd_NoPathDOT(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "--format", "dot", "path", "cleanup()", "main()")
	if err != nil {
		t.Fatalf("path dot (no-path case): %v", err)
	}
	if !strings.HasPrefix(stdout, "digraph plexus {") {
		t.Errorf("expected digraph header, got: %q", stdout)
	}
	if !strings.Contains(stdout, "}") {
		t.Errorf("expected closing brace, got: %q", stdout)
	}
}
