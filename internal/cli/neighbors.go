package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newNeighborsCmd() *cobra.Command {
	var relation string
	cmd := &cobra.Command{
		Use:   "neighbors <label>",
		Short: "Show outgoing edges from a node",
		Args:  cobra.ExactArgs(1),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			return &codegraph.Input{Action: "neighbors", Label: args[0], RelationFilter: relation}, nil
		}),
	}
	cmd.Flags().StringVar(&relation, "relation", "", "Filter edges by relation type (contains|method|inherits|imports|calls)")
	return cmd
}
