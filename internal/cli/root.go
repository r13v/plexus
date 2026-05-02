package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the root cobra command with persistent flags shared by all
// subcommands. Subcommands are added by their respective files.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plexus",
		Short:         "Code-graph queries for a repository",
		Long:          "Plexus builds and queries a per-repo code graph (callers, neighbors, paths, god nodes, communities, keyword search).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringP("repo", "r", ".", "path to the repository root")
	cmd.PersistentFlags().String("cache-dir", "", "override cache parent directory (else PLEXUS_CACHE_DIR or XDG cache)")
	cmd.PersistentFlags().String("format", "text", "output format: text|json|dot")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newStatsCmd())
	cmd.AddCommand(newGodsCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newNodeCmd())
	cmd.AddCommand(newNeighborsCmd())
	cmd.AddCommand(newCallersCmd())
	cmd.AddCommand(newPathCmd())
	cmd.AddCommand(newCommunityCmd())
	cmd.AddCommand(newDumpCmd())

	return cmd
}

// Execute runs the root command and returns a process exit code.
func Execute() int {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
