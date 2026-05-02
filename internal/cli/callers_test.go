package cli

import (
	"strings"
	"testing"
)

func TestCallersCmd_FindsCallers(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "callers", "cleanup()")
	if err != nil {
		t.Fatalf("callers: %v", err)
	}
	if !strings.Contains(stdout, "cleanup") {
		t.Fatalf("expected 'cleanup' in callers output, got: %q", stdout)
	}
}
