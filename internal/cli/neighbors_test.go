package cli

import (
	"strings"
	"testing"
)

func TestNeighborsCmd_DefaultRelation(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "neighbors", "Service")
	if err != nil {
		t.Fatalf("neighbors: %v", err)
	}
	if !strings.Contains(stdout, "Service") {
		t.Fatalf("expected 'Service' in neighbors output, got: %q", stdout)
	}
}

func TestNeighborsCmd_RelationFilter(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "neighbors", "Service", "--relation", "method")
	if err != nil {
		t.Fatalf("neighbors --relation method: %v", err)
	}
	if stdout == "" {
		t.Fatal("expected non-empty neighbors output")
	}
}
