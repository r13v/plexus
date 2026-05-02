package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newCallersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "callers <label>",
		Short: "Show reverse calls edges into a node",
		Args:  cobra.ExactArgs(1),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			return &codegraph.Input{Action: "callers", Label: args[0]}, nil
		}),
	}
}
