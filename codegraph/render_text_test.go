package codegraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildRenderTextGraph() *Graph {
	g := NewGraph()
	g.AddNode(&Node{ID: "func_a", Label: "FuncA()", Kind: "function", SourceFile: "a.go", SourceLocation: "L10", Community: 0})
	g.AddNode(&Node{ID: "func_b", Label: "FuncB()", Kind: "function", SourceFile: "b.go", SourceLocation: "L20", Community: 0})
	g.AddNode(&Node{ID: "func_c", Label: "FuncC()", Kind: "function", SourceFile: "c.go", SourceLocation: "L30", Community: 1})
	g.AddEdge(&Edge{Source: "func_a", Target: "func_b", Relation: "calls"})
	g.AddEdge(&Edge{Source: "func_b", Target: "func_c", Relation: "calls"})
	g.Communities[0] = []string{"func_a", "func_b"}
	g.Communities[1] = []string{"func_c"}
	return g
}

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "render_text", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	want := loadGolden(t, name)
	if got != want {
		t.Errorf("render mismatch for %s\n--- want ---\n%s--- got ---\n%s", name, want, got)
	}
}

func TestRenderText_Stats(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "stats.txt", res.Text(TextOpts{}))
}

func TestRenderText_Gods(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "gods"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "gods.txt", res.Text(TextOpts{}))
}

func TestRenderText_Subgraph(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "query", Query: "FuncA"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "subgraph.txt", res.Text(TextOpts{}))
}

func TestRenderText_Path(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "path", Source: "FuncA()", Target: "FuncC()"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "path.txt", res.Text(TextOpts{}))
}

func TestRenderText_Community(t *testing.T) {
	g := buildRenderTextGraph()
	id := 0
	res, err := Dispatch(g, &Input{Action: "community", CommunityID: &id})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "community.txt", res.Text(TextOpts{}))
}

func TestRenderText_Node(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "node", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "node.txt", res.Text(TextOpts{}))
}

func TestRenderText_Neighbors(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "neighbors.txt", res.Text(TextOpts{}))
}

func TestRenderText_Callers(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "callers", Label: "FuncB()"})
	if err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "callers.txt", res.Text(TextOpts{}))
}

func TestRenderText_NeighborsNoEdges(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "FuncC()"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Text(TextOpts{})
	want := "no outgoing edges from: FuncC()"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderText_CallersNoCallers(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "callers", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Text(TextOpts{})
	want := "no callers found for: FuncA()"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderText_GodsEmpty(t *testing.T) {
	g := NewGraph()
	res, err := Dispatch(g, &Input{Action: "gods"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Text(TextOpts{})
	want := "no god nodes found"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderText_SubgraphTokenBudgetTruncates(t *testing.T) {
	g := NewGraph()
	for i := range 200 {
		id := fmt.Sprintf("node_n%03d", i)
		g.AddNode(&Node{ID: id, Label: id, Kind: "function", SourceFile: "f.go", SourceLocation: "L1"})
	}
	res, err := Dispatch(g, &Input{Action: "query", Query: "n"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.Text(TextOpts{TokenBudget: 10})
	if !strings.Contains(got, "(truncated, token budget exceeded)") {
		t.Errorf("expected truncation notice, got: %s", got)
	}
	if len(got) > 10*3+100 {
		t.Errorf("output not truncated: len=%d", len(got))
	}
}

func TestRenderText_NilResult(t *testing.T) {
	var r *Result
	if got := r.Text(TextOpts{}); got != "" {
		t.Errorf("nil Text = %q, want empty", got)
	}
}

func TestRenderText_UnknownKind(t *testing.T) {
	r := &Result{Kind: "bogus"}
	if got := r.Text(TextOpts{}); got != "" {
		t.Errorf("unknown kind Text = %q, want empty", got)
	}
}
