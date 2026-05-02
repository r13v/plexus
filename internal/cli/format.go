package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

// Output formats accepted by the --format flag.
const (
	FormatText = "text"
	FormatJSON = "json"
	FormatDOT  = "dot"
)

// formatFromFlag reads and validates the --format persistent flag.
func formatFromFlag(cmd *cobra.Command) (string, error) {
	raw, _ := cmd.Flags().GetString("format")
	switch raw {
	case FormatText, FormatJSON, FormatDOT:
		return raw, nil
	case "":
		return FormatText, nil
	default:
		return "", fmt.Errorf("invalid --format %q (want text|json|dot)", raw)
	}
}

// WriteResult writes a *Result to w using the given format. opts apply only
// to FormatText (TokenBudget bounds output for subgraph/path).
func WriteResult(w io.Writer, r *codegraph.Result, format string, opts codegraph.TextOpts) error {
	switch format {
	case FormatText, "":
		_, err := io.WriteString(w, r.Text(opts))
		return err
	case FormatJSON:
		b, err := r.JSON()
		if err != nil {
			return err
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		_, err = io.WriteString(w, "\n")
		return err
	case FormatDOT:
		_, err := io.WriteString(w, r.DOT())
		return err
	default:
		return fmt.Errorf("invalid format %q (want text|json|dot)", format)
	}
}

// emitResult is a convenience wrapper that picks the format from the cobra
// command's flags and writes to its stdout. Reads the local --token-budget
// flag (if defined on cmd) for text rendering.
func emitResult(cmd *cobra.Command, r *codegraph.Result) error {
	format, err := formatFromFlag(cmd)
	if err != nil {
		return err
	}
	opts := codegraph.TextOpts{}
	if f := cmd.Flags().Lookup("token-budget"); f != nil {
		if v, gerr := cmd.Flags().GetInt("token-budget"); gerr == nil {
			opts.TokenBudget = v
		}
	}
	return WriteResult(cmd.OutOrStdout(), r, format, opts)
}
