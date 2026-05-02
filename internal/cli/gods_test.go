package cli

import (
	"encoding/json"
	"testing"
)

func TestGodsCmd_JSON(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "--format", "json", "gods", "--top-n", "5")
	if err != nil {
		t.Fatalf("gods: %v", err)
	}
	var got struct {
		Kind string           `json:"kind"`
		Gods []map[string]any `json:"gods"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("parse json: %v (raw=%q)", err, stdout)
	}
	if got.Kind != "gods" {
		t.Fatalf("expected kind=gods, got %q", got.Kind)
	}
	if len(got.Gods) == 0 {
		t.Fatal("expected at least one god node from goproj fixture")
	}
}
