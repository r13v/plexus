package codegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func expectedKey(toplevel string) string {
	sum := sha256.Sum256([]byte(toplevel))
	hash8 := hex.EncodeToString(sum[:])[:8]
	return filepath.Base(toplevel) + "-" + hash8
}

func TestCachePath_ExplicitOverrideTakesPrecedence(t *testing.T) {
	t.Setenv("PLEXUS_CACHE_DIR", "/should/not/be/used")

	override := "/custom/cache/dir"
	toplevel := "/repos/myproj"

	got, err := CachePath(toplevel, override)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}

	want := filepath.Join(override, expectedKey(toplevel), cacheFilename)
	if got != want {
		t.Errorf("CachePath = %q, want %q", got, want)
	}
}

func TestCachePath_EnvOverride(t *testing.T) {
	envDir := t.TempDir()
	t.Setenv("PLEXUS_CACHE_DIR", envDir)

	toplevel := "/repos/myproj"
	got, err := CachePath(toplevel, "")
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}

	want := filepath.Join(envDir, expectedKey(toplevel), cacheFilename)
	if got != want {
		t.Errorf("CachePath = %q, want %q", got, want)
	}
}

func TestCachePath_DefaultUsesUserCacheDir(t *testing.T) {
	t.Setenv("PLEXUS_CACHE_DIR", "")

	toplevel := "/repos/myproj"
	got, err := CachePath(toplevel, "")
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}

	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir: %v", err)
	}
	want := filepath.Join(ucd, "plexus", expectedKey(toplevel), cacheFilename)
	if got != want {
		t.Errorf("CachePath = %q, want %q", got, want)
	}
}

func TestCachePath_PrecedenceOrdering(t *testing.T) {
	t.Setenv("PLEXUS_CACHE_DIR", "/from/env")

	override := "/from/override"
	toplevel := "/repos/myproj"

	got, err := CachePath(toplevel, override)
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if !strings.HasPrefix(got, override+string(filepath.Separator)) {
		t.Errorf("expected explicit override to win, got %q", got)
	}
	if strings.Contains(got, "from/env") {
		t.Errorf("env should not be used when override is set, got %q", got)
	}
}

func TestCachePath_WeirdToplevelPath(t *testing.T) {
	override := t.TempDir()

	cases := []string{
		"/repos/dir with spaces",
		"/repos/проект-юникод",
		"/repos/with.dots/sub",
	}
	seen := make(map[string]string, len(cases))
	for _, top := range cases {
		got, err := CachePath(top, override)
		if err != nil {
			t.Fatalf("CachePath(%q): %v", top, err)
		}

		key := expectedKey(top)
		want := filepath.Join(override, key, cacheFilename)
		if got != want {
			t.Errorf("CachePath(%q) = %q, want %q", top, got, want)
		}

		if other, dup := seen[key]; dup {
			t.Errorf("hash collision: %q and %q both produced key %q", top, other, key)
		}
		seen[key] = top
	}
}

func TestCachePath_EmptyToplevelErrors(t *testing.T) {
	if _, err := CachePath("", ""); err == nil {
		t.Fatal("expected error for empty repoToplevel, got nil")
	}
}

func TestCachePath_FilenameIsGob(t *testing.T) {
	got, err := CachePath("/repos/x", t.TempDir())
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	if filepath.Base(got) != "code_graph.gob" {
		t.Errorf("cache filename = %q, want code_graph.gob", filepath.Base(got))
	}
}
