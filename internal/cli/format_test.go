package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/r13v/plexus/codegraph"
)

func sampleStatsResult() *codegraph.Result {
	return &codegraph.Result{
		Kind: "stats",
		Stats: &codegraph.StatsResult{
			NodeCount:      3,
			EdgeCount:      2,
			CommunityCount: 1,
		},
	}
}

func sampleSubgraphResult() *codegraph.Result {
	n1 := &codegraph.Node{ID: "a:f", Label: "f", Kind: "function", SourceFile: "a.go"}
	n2 := &codegraph.Node{ID: "a:g", Label: "g", Kind: "function", SourceFile: "a.go"}
	return &codegraph.Result{
		Kind:  "subgraph",
		Nodes: []*codegraph.Node{n1, n2},
		Edges: []*codegraph.Edge{{Source: n1.ID, Target: n2.ID, Relation: "calls"}},
	}
}

func TestWriteResult_Text(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleStatsResult(), FormatText, codegraph.TextOpts{}); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "3") || !strings.Contains(out, "2") {
		t.Fatalf("expected counts in text output, got: %q", out)
	}
}

func TestWriteResult_JSON(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleStatsResult(), FormatJSON, codegraph.TextOpts{}); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("JSON output should end with newline: %q", buf.String())
	}
	var got struct {
		Kind  string `json:"kind"`
		Stats *struct {
			NodeCount int `json:"node_count"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json parse: %v (raw: %s)", err, buf.String())
	}
	if got.Kind != "stats" || got.Stats == nil || got.Stats.NodeCount != 3 {
		t.Fatalf("unexpected JSON payload: %+v", got)
	}
}

func TestWriteResult_DOT(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteResult(&buf, sampleSubgraphResult(), FormatDOT, codegraph.TextOpts{}); err != nil {
		t.Fatalf("WriteResult: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "digraph") {
		t.Fatalf("expected digraph in DOT output, got: %q", out)
	}
}

func TestWriteResult_InvalidFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteResult(&buf, sampleStatsResult(), "yaml", codegraph.TextOpts{})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestFormatFromFlag(t *testing.T) {
	tests := []struct {
		flag    string
		want    string
		wantErr bool
	}{
		{"", FormatText, false},
		{"text", FormatText, false},
		{"json", FormatJSON, false},
		{"dot", FormatDOT, false},
		{"yaml", "", true},
	}
	for _, tt := range tests {
		name := tt.flag
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs([]string{"--format=" + tt.flag, "version"})
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			if tt.flag == "" {
				cmd.SetArgs([]string{"version"})
			}

			var captured *cobra.Command
			vc, _, _ := cmd.Find([]string{"version"})
			vc.RunE = func(c *cobra.Command, _ []string) error {
				captured = c
				return nil
			}

			err := cmd.Execute()
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("execute: %v", err)
				}
				return
			}
			got, ferr := formatFromFlag(captured)
			if tt.wantErr {
				if ferr == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if ferr != nil {
				t.Fatalf("formatFromFlag: %v", ferr)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestEmitResult_PicksFormatFromFlag(t *testing.T) {
	cmd := NewRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	vc, _, _ := cmd.Find([]string{"version"})
	vc.RunE = func(c *cobra.Command, _ []string) error {
		return emitResult(c, sampleStatsResult())
	}
	cmd.SetArgs([]string{"--format=json", "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(buf.String(), `"kind"`) {
		t.Fatalf("expected JSON output, got: %q", buf.String())
	}
}

func TestEmitResult_TokenBudgetFlowsToTextRenderer(t *testing.T) {
	repo := prepareGoFixture(t)
	cacheDir := buildFixture(t, repo)

	wide, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "search", "--token-budget=50000", "Service")
	if err != nil {
		t.Fatalf("search wide budget: %v", err)
	}
	tight, err := runCLI(t, "--repo", repo, "--cache-dir", cacheDir, "search", "--token-budget=1", "Service")
	if err != nil {
		t.Fatalf("search tight budget: %v", err)
	}
	if !strings.Contains(tight, "(truncated, token budget exceeded)") {
		t.Fatalf("expected truncation marker with budget=1, got: %q", tight)
	}
	if strings.Contains(wide, "(truncated, token budget exceeded)") {
		t.Fatalf("did not expect truncation with budget=50000, got: %q", wide)
	}
}
