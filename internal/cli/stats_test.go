package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStatsCmd_Text(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "stats")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(strings.ToLower(stdout), "node") {
		t.Fatalf("expected 'node' in stats text, got: %q", stdout)
	}
}

func TestStatsCmd_JSON(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "--format", "json", "stats")
	if err != nil {
		t.Fatalf("stats json: %v", err)
	}
	var got struct {
		Kind  string `json:"kind"`
		Stats struct {
			NodeCount int `json:"node_count"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse json: %v (raw=%q)", err, stdout)
	}
	if got.Kind != "stats" || got.Stats.NodeCount == 0 {
		t.Fatalf("unexpected stats payload: %+v", got)
	}
}
