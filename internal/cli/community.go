package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func newCommunityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "community <id>",
		Short: "List all node labels in a Louvain community cluster",
		Args:  cobra.ExactArgs(1),
		RunE: runAction(func(args []string) (*codegraph.Input, error) {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return nil, fmt.Errorf("community id must be an integer: %w", err)
			}
			if id < 0 {
				return nil, fmt.Errorf("community id must be >= 0")
			}
			return &codegraph.Input{Action: "community", CommunityID: &id}, nil
		}),
	}
}
