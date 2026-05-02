package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

// search maps to Input.Action = "query"
func newSearchCmd() *cobra.Command {
	var (
		mode        string
		depth       int
		tokenBudget int
	)
	cmd := &cobra.Command{
		Use:   "search <query...>",
		Short: "Keyword-search the graph and render a relevant subgraph",
		Args:  cobra.MinimumNArgs(1),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			return &codegraph.Input{
				Action:      "query",
				Query:       strings.Join(args, " "),
				Mode:        mode,
				Depth:       depth,
				TokenBudget: tokenBudget,
			}, nil
		}),
	}
	cmd.Flags().StringVar(&mode, "mode", "bfs", "Traversal mode: bfs or dfs")
	cmd.Flags().IntVar(&depth, "depth", 0, "Traversal depth (default 3, max 6)")
	cmd.Flags().IntVar(&tokenBudget, "token-budget", 0, "Max output size in approximate tokens (default 4000, max 50000)")
	return cmd
}
