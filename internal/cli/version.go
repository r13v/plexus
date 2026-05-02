package cli

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is overridden at build time via -ldflags. When unset, fall back to
// the VCS revision recorded in build info.
var version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print plexus version and build info",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, rev, modified := buildInfo()
			out := cmd.OutOrStdout()
			if rev == "" {
				_, err := fmt.Fprintf(out, "plexus %s %s/%s\n", v, runtime.GOOS, runtime.GOARCH)
				return err
			}
			suffix := ""
			if modified {
				suffix = "-dirty"
			}
			_, err := fmt.Fprintf(out, "plexus %s %s%s %s/%s\n", v, rev, suffix, runtime.GOOS, runtime.GOARCH)
			return err
		},
	}
}

func buildInfo() (ver, rev string, modified bool) {
	ver = version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver, "", false
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return ver, rev, modified
}
