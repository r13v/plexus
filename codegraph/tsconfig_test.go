package codegraph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	in := []byte(`{
  // line comment
  "compilerOptions": {
    /* block
       comment */
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"], },
  },
}`)
	out := stripJSONC(in)
	// Must round-trip through json.Unmarshal without error.
	var parsed tsConfigRaw
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("stripJSONC output did not parse: %v\n---\n%s", err, out)
	}
	if parsed.CompilerOptions.BaseURL != "." {
		t.Errorf("baseUrl=%q, want .", parsed.CompilerOptions.BaseURL)
	}
	if got := parsed.CompilerOptions.Paths["@/*"]; len(got) != 1 || got[0] != "src/*" {
		t.Errorf("paths[@/*]=%v, want [src/*]", got)
	}
}

func TestLoadTSConfig(t *testing.T) {
	dir := t.TempDir()
	body := `{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"],
      "~utils": ["src/shared/utils.ts"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadTSConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.BaseURL != "." {
		t.Errorf("baseUrl=%q", cfg.BaseURL)
	}
	if len(cfg.Paths) != 2 {
		t.Errorf("want 2 aliases, got %d", len(cfg.Paths))
	}
}

func TestLoadTSConfigMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := loadTSConfig(dir)
	if err != nil || cfg != nil {
		t.Errorf("missing tsconfig should return (nil, nil); got (%+v, %v)", cfg, err)
	}
}

func TestResolveAliasWildcard(t *testing.T) {
	cfg := &tsConfig{
		BaseURL:  ".",
		Paths:    map[string][]string{"@/*": {"src/*"}},
		root:     "/repo",
		repoRoot: "/repo",
	}
	got := cfg.resolveAlias("@/utils/format")
	if len(got) != 1 || got[0] != "src/utils/format" {
		t.Errorf("got %v, want [src/utils/format]", got)
	}
}

func TestResolveAliasExact(t *testing.T) {
	cfg := &tsConfig{
		BaseURL:  ".",
		Paths:    map[string][]string{"~utils": {"src/shared/utils.ts"}},
		root:     "/repo",
		repoRoot: "/repo",
	}
	got := cfg.resolveAlias("~utils")
	if len(got) != 1 || got[0] != "src/shared/utils.ts" {
		t.Errorf("got %v, want [src/shared/utils.ts]", got)
	}
}

func TestResolveAliasLongestMatch(t *testing.T) {
	cfg := &tsConfig{
		BaseURL:  ".",
		Paths:    map[string][]string{"@/*": {"generic/*"}, "@/utils/*": {"specific/*"}},
		root:     "/repo",
		repoRoot: "/repo",
	}
	got := cfg.resolveAlias("@/utils/format")
	if len(got) != 1 || got[0] != "specific/format" {
		t.Errorf("got %v, want [specific/format] (longer-prefix alias should win)", got)
	}
}
