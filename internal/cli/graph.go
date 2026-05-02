package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

// repoPathFromFlag resolves --repo to an absolute path.
func repoPathFromFlag(cmd *cobra.Command) (string, error) {
	raw, _ := cmd.Flags().GetString("repo")
	if raw == "" {
		raw = "."
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("resolving repo path: %w", err)
	}
	return abs, nil
}

func cacheDirFromFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("cache-dir")
	return v
}

// resolveRepoToplevel returns the canonical git toplevel for the given --repo.
func resolveRepoToplevel(cmd *cobra.Command) (string, error) {
	repoPath, err := repoPathFromFlag(cmd)
	if err != nil {
		return "", err
	}
	top, err := codegraph.GitToplevel(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolving git toplevel for %s: %w", repoPath, err)
	}
	return top, nil
}

// loadGraph loads the cache for the resolved repo, auto-building when missing
// and warning on stderr when the cached HEAD differs from current HEAD.
func loadGraph(cmd *cobra.Command) (*codegraph.Graph, error) {
	top, err := resolveRepoToplevel(cmd)
	if err != nil {
		return nil, err
	}
	cacheDir := cacheDirFromFlag(cmd)
	cachePath, err := codegraph.CachePath(top, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("resolving cache path: %w", err)
	}

	g, cachedSHA, loadErr := codegraph.LoadGraph(cachePath)
	if loadErr == nil {
		stale, headSHA := isStaleCache(top, cachedSHA)
		if !stale {
			return g, nil
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"graph cache is stale (cached %s, HEAD %s); rebuilding...\n",
			shortSHA(cachedSHA), shortSHA(headSHA))
		rebuilt, _, berr := codegraph.BuildFromRepo(top, headSHA, cacheDir)
		if berr != nil {
			return nil, berr
		}
		return rebuilt, nil
	}

	headSHA, herr := codegraph.GitHead(top)
	if herr != nil {
		return nil, fmt.Errorf("no cache at %s and cannot read HEAD to build one: %w", cachePath, herr)
	}
	if errors.Is(loadErr, os.ErrNotExist) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "no graph cache at %s; building...\n", cachePath)
	} else {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: graph cache at %s unreadable (%v); rebuilding...\n", cachePath, loadErr)
	}
	g, _, berr := codegraph.BuildFromRepo(top, headSHA, cacheDir)
	if berr != nil {
		return nil, berr
	}
	return g, nil
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

// isStaleCache reports whether the cached SHA is known and differs from HEAD.
// When git HEAD cannot be read (or no SHA was cached), the cache is treated as
// fresh — we serve what we have rather than failing the query.
func isStaleCache(top, cachedSHA string) (stale bool, headSHA string) {
	if cachedSHA == "" {
		return false, ""
	}
	head, err := codegraph.GitHead(top)
	if err != nil {
		return false, ""
	}
	return head != cachedSHA, head
}

// runAction is a helper that loads the graph, builds an Input, dispatches and
// emits the result using the --format flag.
func runAction(build func(args []string) (*codegraph.Input, error)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		g, err := loadGraph(cmd)
		if err != nil {
			return err
		}
		in, err := build(args)
		if err != nil {
			return err
		}
		result, err := codegraph.Dispatch(g, in)
		if err != nil {
			return err
		}
		return emitResult(cmd, result)
	}
}
