package codegraph

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// 60s covers `ls-files -co --exclude-standard` on large monorepos with many
// untracked files (where excludestandard pays per-file). Short ops like
// `rev-parse` complete well under this.
const gitTimeout = 60 * time.Second

func runGit(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// GitHead returns the HEAD commit SHA of the repo at repoPath.
func GitHead(repoPath string) (string, error) {
	out, err := runGit(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GitToplevel returns the canonical repo root for cwd.
func GitToplevel(cwd string) (string, error) {
	out, err := runGit(cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GitLsFiles returns tracked + untracked (excluding ignored) files in repoPath,
// as repo-root-relative paths.
func GitLsFiles(repoPath string) ([]string, error) {
	out, err := runGit(repoPath, "ls-files", "-co", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	// trim trailing NUL if present
	if out[len(out)-1] == 0 {
		out = out[:len(out)-1]
	}
	parts := bytes.Split(out, []byte{0})
	files := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		files = append(files, string(p))
	}
	return files, nil
}
