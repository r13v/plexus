package codegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const cacheFilename = "code_graph.gob"

// CachePath returns the full cache file path for a repository.
//
// Precedence for the parent directory:
//  1. override (if non-empty)
//  2. PLEXUS_CACHE_DIR env var (if set)
//  3. os.UserCacheDir()/plexus
//
// Inside the parent, the file lives at <basename>-<hash8>/code_graph.gob
// where basename is filepath.Base(repoToplevel) and hash8 is the first
// 8 hex chars of sha256(repoToplevel).
func CachePath(repoToplevel, override string) (string, error) {
	if repoToplevel == "" {
		return "", fmt.Errorf("codegraph: empty repoToplevel")
	}

	parent, err := cacheParent(override)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256([]byte(repoToplevel))
	hash8 := hex.EncodeToString(sum[:])[:8]
	basename := filepath.Base(repoToplevel)
	key := fmt.Sprintf("%s-%s", basename, hash8)
	return filepath.Join(parent, key, cacheFilename), nil
}

func cacheParent(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if env := os.Getenv("PLEXUS_CACHE_DIR"); env != "" {
		return env, nil
	}
	ucd, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("codegraph: user cache dir: %w", err)
	}
	return filepath.Join(ucd, "plexus"), nil
}
