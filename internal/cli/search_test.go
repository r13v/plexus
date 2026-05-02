package cli

import (
	"strings"
	"testing"
)

func TestSearchCmd_FindsSymbol(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "search", "Service")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !strings.Contains(stdout, "Service") {
		t.Fatalf("expected 'Service' in search output, got: %q", stdout)
	}
}

func TestSearchCmd_NoMatch(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	_, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "search", "definitelynotpresent_xyzzy")
	if err == nil {
		t.Fatal("expected error for no match")
	}
}
