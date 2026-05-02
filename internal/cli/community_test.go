package cli

import (
	"testing"
)

func TestCommunityCmd_Zero(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	stdout, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "community", "0")
	if err != nil {
		t.Fatalf("community 0: %v", err)
	}
	if stdout == "" {
		t.Fatal("expected non-empty community output")
	}
}

func TestCommunityCmd_NegativeRejected(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	_, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "community", "-1")
	if err == nil {
		t.Fatal("expected error for negative community id")
	}
}

func TestCommunityCmd_NonInteger(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	_, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "community", "abc")
	if err == nil {
		t.Fatal("expected error for non-integer community id")
	}
}
