package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newGodsCmd() *cobra.Command {
	var topN int
	cmd := &cobra.Command{
		Use:   "gods",
		Short: "List top architectural hubs (non-file nodes by degree)",
		Args:  cobra.NoArgs,
		RunE: runAction(func([]string) (*codegraph.Input, error) {
			return &codegraph.Input{Action: "gods", TopN: topN}, nil
		}),
	}
	cmd.Flags().IntVarP(&topN, "top-n", "n", 0, "Number of top results (default 10, max 100)")
	return cmd
}
