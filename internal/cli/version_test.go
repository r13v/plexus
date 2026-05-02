package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	cmd := NewRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "plexus ") {
		t.Fatalf("expected plexus prefix, got: %q", out)
	}
}

func TestRootCmd_PersistentFlags(t *testing.T) {
	cmd := NewRootCmd()
	for _, name := range []string{"repo", "cache-dir", "format"} {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing persistent flag --%s", name)
		}
	}
}
