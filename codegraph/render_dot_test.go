package codegraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadDotGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "render_dot", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func assertDotGolden(t *testing.T, name, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		path := filepath.Join("testdata", "render_dot", name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want := loadDotGolden(t, name)
	if got != want {
		t.Errorf("DOT mismatch for %s\n--- want ---\n%s--- got ---\n%s", name, want, got)
	}
}

func TestRenderDOT_Subgraph(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "query", Query: "FuncA"})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "subgraph.dot", res.DOT())
}

func TestRenderDOT_Path(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "path", Source: "FuncA()", Target: "FuncC()"})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "path.dot", res.DOT())
}

func TestRenderDOT_Neighbors(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "neighbors.dot", res.DOT())
}

func TestRenderDOT_Callers(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "callers", Label: "FuncB()"})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "callers.dot", res.DOT())
}

func TestRenderDOT_Community(t *testing.T) {
	g := buildRenderTextGraph()
	id := 0
	res, err := Dispatch(g, &Input{Action: "community", CommunityID: &id})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "community.dot", res.DOT())
}

func TestRenderDOT_Node(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "node", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	assertDotGolden(t, "node.dot", res.DOT())
}

func TestRenderDOT_Stats(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	want := "// stats has no graph payload — use --format=text\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderDOT_Gods(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "gods"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	want := "// gods has no graph payload — use --format=text\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderDOT_NilResult(t *testing.T) {
	var r *Result
	if got := r.DOT(); got != "" {
		t.Errorf("nil DOT = %q, want empty", got)
	}
}

func TestRenderDOT_UnknownKind(t *testing.T) {
	r := &Result{Kind: "bogus"}
	if got := r.DOT(); got != "" {
		t.Errorf("unknown kind DOT = %q, want empty", got)
	}
}

func TestRenderDOT_FileNodeShape(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "f1", Label: "main.go", Kind: "file"})
	g.AddNode(&Node{ID: "fn1", Label: "Main()", Kind: "function"})
	g.AddEdge(&Edge{Source: "f1", Target: "fn1", Relation: "contains"})
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "main.go"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	if !strings.Contains(got, `"f1" [label="main.go", shape=box`) {
		t.Errorf("expected file shape=box, got: %s", got)
	}
	if !strings.Contains(got, `"fn1" [label="Main()", shape=ellipse`) {
		t.Errorf("expected symbol shape=ellipse, got: %s", got)
	}
	if !strings.Contains(got, `style=solid`) {
		t.Errorf("expected contains edge style=solid, got: %s", got)
	}
}

func TestRenderDOT_ImportsDashedEdge(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "a.go", Kind: "file"})
	g.AddNode(&Node{ID: "b", Label: "b.go", Kind: "file"})
	g.AddEdge(&Edge{Source: "a", Target: "b", Relation: "imports"})
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "a.go"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	if !strings.Contains(got, `style=dashed`) {
		t.Errorf("expected dashed style for imports edge, got: %s", got)
	}
}

func TestRenderDOT_LabelEscaping(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: `weird"id\with`, Label: `Func"X\Y`, Kind: "function"})
	g.AddNode(&Node{ID: "plain", Label: "Plain", Kind: "function"})
	g.AddEdge(&Edge{Source: `weird"id\with`, Target: "plain", Relation: `rel"r\n`})
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: `Func"X\Y`})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	if !strings.Contains(got, `"weird\"id\\with"`) {
		t.Errorf("expected escaped node ID, got: %s", got)
	}
	if !strings.Contains(got, `label="Func\"X\\Y"`) {
		t.Errorf("expected escaped node label, got: %s", got)
	}
	if !strings.Contains(got, `label="rel\"r\\n"`) {
		t.Errorf("expected escaped edge label, got: %s", got)
	}
}

func TestRenderDOT_CommunityColor(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "n0", Label: "N0", Kind: "function", Community: 0})
	g.AddNode(&Node{ID: "n1", Label: "N1", Kind: "function", Community: 1})
	g.AddEdge(&Edge{Source: "n0", Target: "n1", Relation: "calls"})
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "N0"})
	if err != nil {
		t.Fatal(err)
	}
	got := res.DOT()
	if !strings.Contains(got, `fillcolor="`+dotPalette[0]+`"`) {
		t.Errorf("expected palette[0] for community 0, got: %s", got)
	}
	if !strings.Contains(got, `fillcolor="`+dotPalette[1]+`"`) {
		t.Errorf("expected palette[1] for community 1, got: %s", got)
	}
}
