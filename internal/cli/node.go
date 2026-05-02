package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newNodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "node <label>",
		Short: "Render a single node by exact label (case-insensitive)",
		Args:  cobra.ExactArgs(1),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			return &codegraph.Input{Action: "node", Label: args[0]}, nil
		}),
	}
}
