package cli

import (
	"strings"
	"testing"
)

func TestNodeCmd_FindsLabel(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "node", "Service")
	if err != nil {
		t.Fatalf("node: %v", err)
	}
	if !strings.Contains(stdout, "Service") {
		t.Fatalf("expected 'Service' in node output, got: %q", stdout)
	}
}

func TestNodeCmd_NotFound(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	_, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "node", "DoesNotExistFoo")
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}
