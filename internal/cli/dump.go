package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dump",
		Short: "Dump the cached graph as pretty-printed JSON (--format ignored)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			g, err := loadGraph(cmd)
			if err != nil {
				return err
			}
			b, err := json.MarshalIndent(g, "", "  ")
			if err != nil {
				return fmt.Errorf("encoding graph: %w", err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return err
		},
	}
}
