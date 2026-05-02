package codegraph

import (
	"fmt"
	"log/slog"
	"time"
)

// BuildFromRepo builds or loads a cached code graph for the given repository.
// If a cache file exists and its HEAD matches headSHA, the cached graph is returned.
// Otherwise, files are walked, extracted, merged, clustered, and the result is cached.
//
// cacheDir is an explicit override for the cache parent directory. Pass "" to
// fall back to PLEXUS_CACHE_DIR or os.UserCacheDir()/plexus (see CachePath).
//
// Cache write failures are logged but do not prevent returning the in-memory graph.
// The returned bool indicates whether the result came from cache.
func BuildFromRepo(repoPath, headSHA, cacheDir string) (*Graph, bool, error) {
	cachePath, err := CachePath(repoPath, cacheDir)
	if err != nil {
		return nil, false, fmt.Errorf("codegraph: resolve cache path: %w", err)
	}

	g, cachedSHA, err := LoadGraph(cachePath)
	if err == nil && headSHA != "" && cachedSHA == headSHA {
		slog.Info("codegraph: cache hit", "repo", repoPath, "sha", headSHA[:min(len(headSHA), 8)])
		return g, true, nil
	}

	start := time.Now()

	nodes, edges, result, err := WalkRepo(repoPath)
	if err != nil {
		return nil, false, fmt.Errorf("codegraph: build %s: %w", repoPath, err)
	}

	g = NewGraph()
	for _, n := range nodes {
		g.AddNode(n)
	}
	for i := range edges {
		g.AddEdge(&edges[i])
	}

	Cluster(g)

	if len(g.Nodes) == 0 {
		slog.Info("codegraph: empty graph, skipping cache write", "repo", repoPath)
		return g, false, nil
	}

	if saveErr := g.Save(cachePath, headSHA); saveErr != nil {
		slog.Warn("codegraph: cache write failed, using in-memory graph", "repo", repoPath, "err", saveErr)
	}

	elapsed := time.Since(start)
	slog.Info("codegraph: built",
		"repo", repoPath,
		"nodes", len(g.Nodes),
		"edges", len(g.Edges),
		"communities", len(g.Communities),
		"skipped", result.Skipped,
		"errors", result.Errors,
		"elapsed", elapsed.Round(time.Millisecond),
	)

	return g, false, nil
}
