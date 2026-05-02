package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDumpCmd_OutputsGraphJSON(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "dump")
	if err != nil {
		t.Fatalf("dump: %v", err)
	}

	var got struct {
		Nodes map[string]struct {
			ID    string `json:"id"`
			Label string `json:"label"`
			Kind  string `json:"kind"`
		} `json:"nodes"`
		Edges []struct {
			Source   string `json:"source"`
			Target   string `json:"target"`
			Relation string `json:"relation"`
		} `json:"edges"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse json: %v (raw=%q)", err, stdout)
	}
	if len(got.Nodes) == 0 {
		t.Fatalf("expected non-empty nodes, got: %s", stdout)
	}
	if !strings.Contains(stdout, "\n  \"") {
		t.Fatalf("expected pretty-printed (indented) JSON, got: %q", stdout)
	}
}

func TestDumpCmd_IgnoresFormatFlag(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	// Even with --format=text, dump must emit JSON.
	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "--format", "text", "dump")
	if err != nil {
		t.Fatalf("dump --format=text: %v", err)
	}
	var probe map[string]any
	if err := json.Unmarshal([]byte(stdout), &probe); err != nil {
		t.Fatalf("expected JSON regardless of --format, parse failed: %v (raw=%q)", err, stdout)
	}
	if _, ok := probe["nodes"]; !ok {
		t.Fatalf("expected 'nodes' key in dump output: %s", stdout)
	}
}
