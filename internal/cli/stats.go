package cli

import (
	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show graph size and top-degree nodes",
		Args:  cobra.NoArgs,
		RunE: runAction(func([]string) (*codegraph.Input, error) {
			return &codegraph.Input{Action: "stats"}, nil
		}),
	}
}
