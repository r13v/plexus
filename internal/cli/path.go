package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newPathCmd() *cobra.Command {
	var tokenBudget int
	cmd := &cobra.Command{
		Use:   "path <source> <target>",
		Short: "Shortest forward-edge path between two nodes",
		Args:  cobra.ExactArgs(2),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			return &codegraph.Input{
				Action:      "path",
				Source:      args[0],
				Target:      args[1],
				TokenBudget: tokenBudget,
			}, nil
		}),
	}
	cmd.Flags().IntVar(&tokenBudget, "token-budget", 0, "Max output size in approximate tokens (default 4000, max 50000)")
	return cmd
}
