package codegraph

import (
	"errors"
	"strings"
	"testing"
)

func buildDispatchTestGraph() *Graph {
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

func TestDispatch_Stats(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "stats" {
		t.Errorf("Kind = %q, want %q", res.Kind, "stats")
	}
	if res.Stats == nil {
		t.Fatal("Stats is nil")
	}
	if res.Stats.NodeCount != 3 {
		t.Errorf("NodeCount = %d, want 3", res.Stats.NodeCount)
	}
	if res.Stats.EdgeCount != 2 {
		t.Errorf("EdgeCount = %d, want 2", res.Stats.EdgeCount)
	}
	if res.Stats.CommunityCount != 2 {
		t.Errorf("CommunityCount = %d, want 2", res.Stats.CommunityCount)
	}
}

func TestDispatch_Gods(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "gods"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "gods" {
		t.Errorf("Kind = %q, want %q", res.Kind, "gods")
	}
	if len(res.Gods) == 0 {
		t.Fatal("Gods is empty")
	}
}

func TestDispatch_Query(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "query", Query: "FuncA"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "subgraph" {
		t.Errorf("Kind = %q, want %q", res.Kind, "subgraph")
	}
	if res.Subgraph == nil {
		t.Fatal("Subgraph is nil")
	}
	foundA := false
	for _, n := range res.Nodes {
		if n.ID == "func_a" {
			foundA = true
			break
		}
	}
	if !foundA {
		t.Errorf("expected func_a in result Nodes, got %d nodes", len(res.Nodes))
	}
}

func TestDispatch_QueryRequiresQueryField(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "query"})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestDispatch_QueryNoMatch(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "query", Query: "nonexistent_xyz"})
	if !errors.Is(err, ErrNoMatch) {
		t.Errorf("err = %v, want ErrNoMatch", err)
	}
}

func TestDispatch_Neighbors(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "neighbors" {
		t.Errorf("Kind = %q, want %q", res.Kind, "neighbors")
	}
	if res.Neighbors == nil || res.Neighbors.Node.ID != "func_a" {
		t.Errorf("expected node func_a, got %+v", res.Neighbors)
	}
	if len(res.Neighbors.Neighbors) != 1 || res.Neighbors.Neighbors[0].ID != "func_b" {
		t.Errorf("expected neighbor func_b, got %+v", res.Neighbors.Neighbors)
	}
}

func TestDispatch_Callers(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "callers", Label: "FuncB()"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "callers" {
		t.Errorf("Kind = %q, want %q", res.Kind, "callers")
	}
	if res.Callers == nil || res.Callers.Node.ID != "func_b" {
		t.Errorf("expected node func_b, got %+v", res.Callers)
	}
	if len(res.Callers.Callers) != 1 || res.Callers.Callers[0].ID != "func_a" {
		t.Errorf("expected caller func_a, got %+v", res.Callers.Callers)
	}
}

func TestDispatch_Path(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "path", Source: "FuncA()", Target: "FuncC()"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "path" {
		t.Errorf("Kind = %q, want %q", res.Kind, "path")
	}
	if res.Path == nil {
		t.Fatal("Path is nil")
	}
	if len(res.Path.Nodes) != 3 {
		t.Errorf("Path.Nodes len = %d, want 3", len(res.Path.Nodes))
	}
	if res.Path.Source.ID != "func_a" || res.Path.Target.ID != "func_c" {
		t.Errorf("Path endpoints wrong: src=%s tgt=%s", res.Path.Source.ID, res.Path.Target.ID)
	}
}

func TestDispatch_PathNoPath(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "x", Label: "X()", Kind: "function"})
	g.AddNode(&Node{ID: "y", Label: "Y()", Kind: "function"})
	res, err := Dispatch(g, &Input{Action: "path", Source: "X()", Target: "Y()"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Kind != "path" {
		t.Errorf("Kind = %q, want %q", res.Kind, "path")
	}
	if res.Path == nil {
		t.Fatal("Path is nil")
	}
	if res.Path.Source.ID != "x" || res.Path.Target.ID != "y" {
		t.Errorf("Path endpoints wrong: src=%s tgt=%s", res.Path.Source.ID, res.Path.Target.ID)
	}
	if len(res.Path.Nodes) != 0 || len(res.Path.Edges) != 0 {
		t.Errorf("expected empty path nodes/edges, got nodes=%d edges=%d", len(res.Path.Nodes), len(res.Path.Edges))
	}
}

func TestDispatch_PathBadSource(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "path", Source: "Missing()", Target: "FuncC()"})
	if err == nil {
		t.Fatal("expected error for missing source")
	}
	if !strings.HasPrefix(err.Error(), "source:") {
		t.Errorf("expected source-prefixed error, got: %v", err)
	}
}

func TestDispatch_Community(t *testing.T) {
	g := buildDispatchTestGraph()
	cid := 0
	res, err := Dispatch(g, &Input{Action: "community", CommunityID: &cid})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "community" {
		t.Errorf("Kind = %q, want %q", res.Kind, "community")
	}
	if res.Community == nil || res.Community.ID != 0 {
		t.Fatalf("Community wrong: %+v", res.Community)
	}
	if len(res.Community.Nodes) != 2 {
		t.Errorf("Community.Nodes len = %d, want 2", len(res.Community.Nodes))
	}
}

func TestDispatch_CommunityRequiresID(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "community"})
	if err == nil {
		t.Fatal("expected error for missing community_id")
	}
}

func TestDispatch_CommunityNotFound(t *testing.T) {
	g := buildDispatchTestGraph()
	cid := 99
	_, err := Dispatch(g, &Input{Action: "community", CommunityID: &cid})
	if !errors.Is(err, ErrCommunityNotFound) {
		t.Errorf("err = %v, want ErrCommunityNotFound", err)
	}
}

func TestDispatch_Node(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "node", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "node" {
		t.Errorf("Kind = %q, want %q", res.Kind, "node")
	}
	if len(res.Nodes) != 1 || res.Nodes[0].ID != "func_a" {
		t.Errorf("expected single node func_a, got %+v", res.Nodes)
	}
}

func TestDispatch_NodeNotFound(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "node", Label: "Missing()"})
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("err = %v, want ErrNodeNotFound", err)
	}
}

func TestDispatch_NeighborsAmbiguous(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a1", Label: "Foo()", Kind: "function", SourceFile: "a.go"})
	g.AddNode(&Node{ID: "a2", Label: "Foo()", Kind: "function", SourceFile: "b.go"})
	_, err := Dispatch(g, &Input{Action: "neighbors", Label: "Foo()"})
	if err == nil {
		t.Fatal("expected ambiguous label error")
	}
	var amb *AmbiguousLabelError
	if !errors.As(err, &amb) {
		t.Errorf("expected *AmbiguousLabelError, got %T: %v", err, err)
	}
}

func TestDispatch_NilGraph(t *testing.T) {
	_, err := Dispatch(nil, &Input{Action: "stats"})
	if err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestDispatch_UnknownAction(t *testing.T) {
	g := buildDispatchTestGraph()
	_, err := Dispatch(g, &Input{Action: "bogus"})
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Errorf("expected 'unknown action' in error, got: %v", err)
	}
}

func TestDispatch_DepthClamped(t *testing.T) {
	g := buildDispatchTestGraph()
	res, err := Dispatch(g, &Input{Action: "query", Query: "FuncA", Depth: 100})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != "subgraph" {
		t.Errorf("Kind = %q, want %q", res.Kind, "subgraph")
	}
}

func TestClampVal(t *testing.T) {
	tests := []struct {
		val, def, max, want int
	}{
		{0, 3, 6, 3},
		{-1, 3, 6, 3},
		{2, 3, 6, 2},
		{6, 3, 6, 6},
		{7, 3, 6, 6},
		{100, 3, 6, 6},
	}
	for _, tt := range tests {
		got := clampVal(tt.val, tt.def, tt.max)
		if got != tt.want {
			t.Errorf("clampVal(%d, %d, %d) = %d, want %d", tt.val, tt.def, tt.max, got, tt.want)
		}
	}
}
