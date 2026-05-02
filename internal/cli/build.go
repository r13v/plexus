package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build (or rebuild) the code graph cache",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			top, err := resolveRepoToplevel(cmd)
			if err != nil {
				return err
			}
			sha, err := codegraph.GitHead(top)
			if err != nil {
				return fmt.Errorf("reading HEAD: %w", err)
			}
			start := time.Now()
			g, cached, err := codegraph.BuildFromRepo(top, sha, cacheDirFromFlag(cmd))
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"nodes=%d edges=%d communities=%d cached=%v elapsed=%s\n",
				len(g.Nodes), len(g.Edges), len(g.Communities), cached,
				time.Since(start).Round(time.Millisecond),
			)
			return nil
		},
	}
}
