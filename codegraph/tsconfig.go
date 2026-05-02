package codegraph

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// tsConfig captures the subset of tsconfig.json we use for import resolution:
// baseUrl + path aliases. Other fields are ignored.
type tsConfig struct {
	BaseURL string
	// Paths maps alias patterns ("@/*", "~/utils") to candidate targets
	// ("src/*", "./src/utils"). Wildcards use "*" in both key and value.
	Paths map[string][]string
	// root is the absolute directory containing tsconfig.json, used to
	// anchor relative baseUrl/paths entries to the repo root.
	root string
	// repoRoot is the repository root; resolved alias candidates are
	// emitted as repo-root-relative paths.
	repoRoot string
}

type tsConfigRaw struct {
	CompilerOptions struct {
		BaseURL string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// loadTSConfig reads and parses repoRoot/tsconfig.json. Returns (nil, nil) when
// no tsconfig is present. Parse errors are logged and return (nil, nil) so
// callers can continue without aliases.
func loadTSConfig(repoRoot string) (*tsConfig, error) {
	tsconfigPath := filepath.Join(repoRoot, "tsconfig.json")
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tsconfig.json: %w", err)
	}

	stripped := stripJSONC(data)
	var raw tsConfigRaw
	if err := json.Unmarshal(stripped, &raw); err != nil {
		slog.Warn("codegraph: tsconfig.json parse failed, ignoring path aliases", "err", err)
		return nil, nil
	}

	cfg := &tsConfig{
		BaseURL:  raw.CompilerOptions.BaseURL,
		Paths:    raw.CompilerOptions.Paths,
		root:     repoRoot,
		repoRoot: repoRoot,
	}
	return cfg, nil
}

// resolveAlias returns repo-root-relative candidate paths for a bare specifier
// that matches a tsconfig path alias. Returns nil when the spec matches no
// alias. Extensions and index files are NOT appended here — the caller handles
// that via the same extension-guess logic used for relative imports.
func (c *tsConfig) resolveAlias(spec string) []string {
	if c == nil || len(c.Paths) == 0 {
		return nil
	}

	bestKey := ""
	bestPrefix := ""
	for key := range c.Paths {
		if !matchAlias(key, spec) {
			continue
		}
		// Longest literal prefix wins, matching TypeScript resolution rules.
		prefix, _, _ := strings.Cut(key, "*")
		if len(prefix) > len(bestPrefix) || bestKey == "" {
			bestKey = key
			bestPrefix = prefix
		}
	}
	if bestKey == "" {
		return nil
	}

	var star string
	if prefix, suffix, found := strings.Cut(bestKey, "*"); found {
		star = spec[len(prefix) : len(spec)-len(suffix)]
	}

	var candidates []string
	for _, target := range c.Paths[bestKey] {
		replaced := target
		if strings.Contains(target, "*") {
			replaced = strings.Replace(target, "*", star, 1)
		}
		candidates = append(candidates, c.anchor(replaced))
	}
	return candidates
}

// anchor turns a tsconfig-local path into a repo-root-relative path,
// applying baseUrl when the target isn't already explicitly relative.
func (c *tsConfig) anchor(p string) string {
	p = filepath.ToSlash(p)
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") {
		abs := path.Join(c.root, p)
		rel, err := filepath.Rel(c.repoRoot, abs)
		if err != nil {
			return path.Clean(p)
		}
		return filepath.ToSlash(rel)
	}
	base := c.BaseURL
	if base == "" {
		base = "."
	}
	abs := path.Join(c.root, base, p)
	rel, err := filepath.Rel(c.repoRoot, abs)
	if err != nil {
		return path.Clean(path.Join(base, p))
	}
	return filepath.ToSlash(rel)
}

// matchAlias returns true when spec matches a tsconfig paths key.
// Keys may contain a single "*" wildcard.
func matchAlias(key, spec string) bool {
	prefix, suffix, found := strings.Cut(key, "*")
	if !found {
		return key == spec
	}
	if !strings.HasPrefix(spec, prefix) {
		return false
	}
	if !strings.HasSuffix(spec, suffix) {
		return false
	}
	return len(spec) >= len(prefix)+len(suffix)
}

var (
	reLineComment  = regexp.MustCompile(`(?m)//[^\n]*`)
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reTrailComma   = regexp.MustCompile(`,(\s*[}\]])`)
)

// stripJSONC removes JSONC-only syntax (line and block comments, trailing
// commas) so encoding/json can parse typical tsconfig.json files. The stripper
// is regex-based and doesn't respect string boundaries, so a "//" inside a
// string literal would be clobbered; tsconfigs rarely contain such strings.
func stripJSONC(data []byte) []byte {
	data = reBlockComment.ReplaceAll(data, nil)
	data = reLineComment.ReplaceAll(data, nil)
	data = reTrailComma.ReplaceAll(data, []byte("$1"))
	return data
}
