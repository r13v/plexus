package codegraph

import (
	"errors"
	"testing"
)

// buildTestGraph creates a graph:
//
//	A --calls--> B --calls--> C --calls--> D
//	A --imports--> E
//	B --calls--> E
//	F (isolated)
func buildTestGraph() *Graph {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "FuncA()", Kind: "function", SourceFile: "pkg/a.go", SourceLocation: "L1", Community: 0})
	g.AddNode(&Node{ID: "b", Label: "FuncB()", Kind: "function", SourceFile: "pkg/b.go", SourceLocation: "L10", Community: 0})
	g.AddNode(&Node{ID: "c", Label: "FuncC()", Kind: "function", SourceFile: "pkg/c.go", SourceLocation: "L20", Community: 1})
	g.AddNode(&Node{ID: "d", Label: "FuncD()", Kind: "function", SourceFile: "pkg/d.go", SourceLocation: "L30", Community: 1})
	g.AddNode(&Node{ID: "e", Label: "TypeE", Kind: "type", SourceFile: "pkg/e.go", SourceLocation: "L5", Community: 2})
	g.AddNode(&Node{ID: "f", Label: "Isolated()", Kind: "function", SourceFile: "pkg/f.go", SourceLocation: "L1", Community: 3})

	g.AddEdge(&Edge{Source: "a", Target: "b", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b", Target: "c", Relation: "calls"})
	g.AddEdge(&Edge{Source: "c", Target: "d", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a", Target: "e", Relation: "imports"})
	g.AddEdge(&Edge{Source: "b", Target: "e", Relation: "calls"})

	g.Communities[0] = []string{"a", "b"}
	g.Communities[1] = []string{"c", "d"}
	g.Communities[2] = []string{"e"}
	g.Communities[3] = []string{"f"}

	return g
}

func nodeIDs(nodes []*Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}

func containsID(nodes []*Node, id string) bool {
	for _, n := range nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func TestBFS_BasicOrder(t *testing.T) {
	g := buildTestGraph()
	nodes := g.BFS([]string{"a"}, 3, "")
	ids := nodeIDs(nodes)

	if ids[0] != "a" {
		t.Errorf("BFS should start with seed, got %v", ids)
	}
	if len(ids) < 4 {
		t.Errorf("BFS depth=3 from A should reach at least A,B,C,E,D; got %v", ids)
	}

	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate node %q in BFS result", id)
		}
		seen[id] = true
	}

	// B and E should appear before C and D (depth 1 before depth 2).
	bIdx, cIdx := -1, -1
	for i, id := range ids {
		if id == "b" {
			bIdx = i
		}
		if id == "c" {
			cIdx = i
		}
	}
	if bIdx >= cIdx {
		t.Errorf("BFS: B (depth 1) should appear before C (depth 2), got B@%d C@%d", bIdx, cIdx)
	}
}

func TestBFS_DepthLimit(t *testing.T) {
	g := buildTestGraph()
	nodes := g.BFS([]string{"a"}, 1, "")
	ids := nodeIDs(nodes)

	for _, id := range ids {
		if id == "c" || id == "d" {
			t.Errorf("BFS depth=1 should not reach %q", id)
		}
	}
	if len(ids) != 3 { // a, b, e
		t.Errorf("expected 3 nodes at depth 1, got %d: %v", len(ids), ids)
	}
}

func TestBFS_RelationFilter(t *testing.T) {
	g := buildTestGraph()
	nodes := g.BFS([]string{"a"}, 3, "imports")
	ids := nodeIDs(nodes)

	// Only imports relation: A -> E
	if len(ids) != 2 {
		t.Errorf("expected [a, e] with imports filter, got %v", ids)
	}
}

func TestBFS_NonexistentStart(t *testing.T) {
	g := buildTestGraph()
	nodes := g.BFS([]string{"nonexistent"}, 3, "")
	if len(nodes) != 0 {
		t.Errorf("BFS with nonexistent start should return empty, got %d nodes", len(nodes))
	}
}

func TestDFS_DepthFirst(t *testing.T) {
	g := buildTestGraph()
	nodes := g.DFS([]string{"a"}, 3, "")
	ids := nodeIDs(nodes)

	if ids[0] != "a" {
		t.Errorf("DFS should start with seed, got %v", ids)
	}
	if len(ids) < 4 {
		t.Errorf("DFS depth=3 from A should reach at least 4 nodes; got %v", ids)
	}

	aIdx, bIdx, cIdx := -1, -1, -1
	for i, id := range ids {
		switch id {
		case "a":
			aIdx = i
		case "b":
			bIdx = i
		case "c":
			cIdx = i
		}
	}
	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("DFS should traverse A→B→C in order, got A@%d B@%d C@%d", aIdx, bIdx, cIdx)
	}
}

func TestDFS_DepthLimit(t *testing.T) {
	g := buildTestGraph()
	nodes := g.DFS([]string{"a"}, 1, "")
	ids := nodeIDs(nodes)

	for _, id := range ids {
		if id == "c" || id == "d" {
			t.Errorf("DFS depth=1 should not reach %q", id)
		}
	}
}

func TestDFS_RelationFilter(t *testing.T) {
	g := buildTestGraph()
	nodes := g.DFS([]string{"a"}, 5, "calls")
	expected := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": true}
	got := map[string]bool{}
	for _, n := range nodes {
		got[n.ID] = true
	}
	for k := range expected {
		if !got[k] {
			t.Errorf("DFS calls filter: expected %q in result", k)
		}
	}
}

func TestShortestPath_Connected(t *testing.T) {
	g := buildTestGraph()
	nodes, edges := g.ShortestPath("a", "d", 10)

	if nodes == nil {
		t.Fatal("expected path from A to D, got nil")
	}
	ids := nodeIDs(nodes)
	expected := []string{"a", "b", "c", "d"}
	if len(ids) != len(expected) {
		t.Fatalf("expected path %v, got %v", expected, ids)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("path[%d] = %q, want %q", i, id, expected[i])
		}
	}
	if len(edges) != 3 {
		t.Errorf("expected 3 edges in path, got %d", len(edges))
	}
}

func TestShortestPath_Disconnected(t *testing.T) {
	g := buildTestGraph()
	nodes, edges := g.ShortestPath("a", "f", 10)
	if nodes != nil || edges != nil {
		t.Errorf("expected nil for disconnected nodes, got %v, %v", nodes, edges)
	}
}

func TestShortestPath_SameNode(t *testing.T) {
	g := buildTestGraph()
	nodes, edges := g.ShortestPath("a", "a", 10)
	if len(nodes) != 1 || nodes[0].ID != "a" {
		t.Errorf("same-node path should return [a], got %v", nodeIDs(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("same-node path should have 0 edges, got %d", len(edges))
	}
}

func TestShortestPath_NonexistentNodes(t *testing.T) {
	g := buildTestGraph()
	nodes, _ := g.ShortestPath("missing", "a", 10)
	if nodes != nil {
		t.Error("expected nil for missing source")
	}
	nodes, _ = g.ShortestPath("a", "missing", 10)
	if nodes != nil {
		t.Error("expected nil for missing target")
	}
}

func TestShortestPath_MaxHops(t *testing.T) {
	g := buildTestGraph()
	nodes, _ := g.ShortestPath("a", "d", 2)
	if nodes != nil {
		t.Error("expected nil when maxHops insufficient")
	}
}

func TestScoreNodesByKeyword(t *testing.T) {
	g := buildTestGraph()

	tests := []struct {
		name    string
		terms   []string
		wantTop string
		minLen  int
	}{
		{"exact label match", []string{"FuncA"}, "a", 1},
		{"file match", []string{"pkg/b.go"}, "b", 1},
		{"case insensitive", []string{"funcc"}, "c", 1},
		{"multi-term boost", []string{"Func", "a.go"}, "a", 1},
		{"no match", []string{"zzzzz"}, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scored := g.scoreNodesByKeyword(tt.terms)
			if len(scored) < tt.minLen {
				t.Fatalf("expected at least %d results, got %d", tt.minLen, len(scored))
			}
			if tt.wantTop != "" && scored[0].node.ID != tt.wantTop {
				t.Errorf("top result = %q (score=%d), want %q", scored[0].node.ID, scored[0].score, tt.wantTop)
			}
		})
	}
}

func TestScoreNodesByKeyword_MaxThree(t *testing.T) {
	g := buildTestGraph()
	scored := g.scoreNodesByKeyword([]string{"Func"})
	if len(scored) > 3 {
		t.Errorf("expected max 3 results, got %d", len(scored))
	}
}

// --- High-level query method tests ---

func TestQuery_BFS(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("FuncA", "bfs", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(res.Nodes, "a") {
		t.Error("Query should include seed node a")
	}
	if !containsID(res.Nodes, "b") {
		t.Error("Query BFS depth=2 from FuncA should reach b")
	}
}

func TestQuery_DFS(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("FuncA", "dfs", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(res.Nodes, "a") {
		t.Error("Query DFS should include seed node a")
	}
	if !containsID(res.Nodes, "d") {
		t.Error("Query DFS depth=3 from FuncA should reach d via a->b->c->d")
	}
}

func TestQuery_NoTerms(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("", "bfs", 3)
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if !errors.Is(err, ErrNoQueryTerms) {
		t.Errorf("expected ErrNoQueryTerms, got %v", err)
	}
}

func TestQuery_NoMatch(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("zzzznonexistent", "bfs", 3)
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if !errors.Is(err, ErrNoMatch) {
		t.Errorf("expected ErrNoMatch, got %v", err)
	}
}

func TestQuery_IncludesEdges(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("FuncA", "bfs", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Edges) == 0 {
		t.Error("Query should include edges between traversed nodes")
	}
}

func TestQuery_PopulatesSeeds(t *testing.T) {
	g := buildTestGraph()
	res, err := g.Query("FuncA", "bfs", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Seeds) == 0 {
		t.Error("Query should populate Seeds with scored matches")
	}
	if res.Seeds[0].ID != "a" {
		t.Errorf("top seed should be a (FuncA), got %q", res.Seeds[0].ID)
	}
}

func TestGetNode_Found(t *testing.T) {
	g := buildTestGraph()
	n, err := g.GetNode("FuncA()")
	if err != nil {
		t.Fatal(err)
	}
	if n.ID != "a" {
		t.Errorf("GetNode should return node a, got %q", n.ID)
	}
}

func TestGetNode_CaseInsensitive(t *testing.T) {
	g := buildTestGraph()
	n, err := g.GetNode("funca()")
	if err != nil {
		t.Fatal(err)
	}
	if n.ID != "a" {
		t.Errorf("GetNode should be case-insensitive (got %q)", n.ID)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	g := buildTestGraph()
	n, err := g.GetNode("NonExistent")
	if n != nil {
		t.Errorf("expected nil node, got %+v", n)
	}
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestGetNode_Ambiguous(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "p1", Label: "Props", Kind: "interface", SourceFile: "a.tsx"})
	g.AddNode(&Node{ID: "p2", Label: "Props", Kind: "interface", SourceFile: "b.tsx"})
	n, err := g.GetNode("Props")
	if n != nil {
		t.Errorf("expected nil for ambiguous label, got %+v", n)
	}
	if disambig, derr := g.GetNode("p1"); derr != nil || disambig == nil || disambig.ID != "p1" {
		t.Errorf("expected ID-based disambiguation to return p1, got node=%+v err=%v", disambig, derr)
	}
	var amb *AmbiguousLabelError
	if !errors.As(err, &amb) {
		t.Fatalf("expected *AmbiguousLabelError, got %v", err)
	}
	if amb.Count != 2 {
		t.Errorf("expected count=2, got %d", amb.Count)
	}
}

func TestGetNeighbors_Basic(t *testing.T) {
	g := buildTestGraph()
	nb, err := g.GetNeighbors("FuncA()", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(nb.Neighbors, "b") {
		t.Error("FuncA neighbors should include b (calls)")
	}
	if !containsID(nb.Neighbors, "e") {
		t.Error("FuncA neighbors should include e (imports)")
	}
}

func TestGetNeighbors_RelationFilter(t *testing.T) {
	g := buildTestGraph()
	nb, err := g.GetNeighbors("FuncA()", "calls")
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(nb.Neighbors, "b") {
		t.Error("calls neighbors should include b")
	}
	if containsID(nb.Neighbors, "e") {
		t.Error("calls neighbors should not include e (imports relation)")
	}
}

func TestGetNeighbors_NotFound(t *testing.T) {
	g := buildTestGraph()
	nb, err := g.GetNeighbors("NonExistent", "")
	if nb != nil {
		t.Errorf("expected nil, got %+v", nb)
	}
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestGetNeighbors_NoEdges(t *testing.T) {
	g := buildTestGraph()
	nb, err := g.GetNeighbors("Isolated()", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(nb.Edges) != 0 {
		t.Errorf("isolated node should have no outgoing edges, got %d", len(nb.Edges))
	}
}

func TestGetCallers_Basic(t *testing.T) {
	g := buildTestGraph()
	c, err := g.GetCallers("FuncB()")
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(c.Callers, "a") {
		t.Error("Callers of FuncB should include a")
	}
	if len(c.Edges) == 0 {
		t.Error("Callers should include edges")
	}
}

func TestGetCallers_OnlyCalls(t *testing.T) {
	g := buildTestGraph()
	// E is called by B (calls) and imported by A (imports — not a caller).
	c, err := g.GetCallers("TypeE")
	if err != nil {
		t.Fatal(err)
	}
	if !containsID(c.Callers, "b") {
		t.Error("Callers of TypeE should include b (calls relation)")
	}
	if containsID(c.Callers, "a") {
		t.Error("Callers should only include calls relation, not imports")
	}
}

func TestGetCallers_NoCaller(t *testing.T) {
	g := buildTestGraph()
	c, err := g.GetCallers("FuncA()")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Edges) != 0 {
		t.Errorf("FuncA has no callers, got %d edges", len(c.Edges))
	}
}

func TestGetCallers_NotFound(t *testing.T) {
	g := buildTestGraph()
	_, err := g.GetCallers("NoSuchNode")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestStats(t *testing.T) {
	g := buildTestGraph()
	s := g.Stats()

	if s.NodeCount != 6 {
		t.Errorf("expected 6 nodes, got %d", s.NodeCount)
	}
	if s.EdgeCount != 5 {
		t.Errorf("expected 5 edges, got %d", s.EdgeCount)
	}
	if s.CommunityCount != 4 {
		t.Errorf("expected 4 communities, got %d", s.CommunityCount)
	}
	if len(s.TopByDegree) == 0 {
		t.Fatal("expected non-empty top-by-degree")
	}
	// B has highest degree: 2 outgoing (c, e) + 1 incoming (a) = 3.
	if s.TopByDegree[0].Node.ID != "b" {
		t.Errorf("expected top-degree node to be b, got %q", s.TopByDegree[0].Node.ID)
	}
	if s.TopByDegree[0].Degree != 3 {
		t.Errorf("expected top degree=3, got %d", s.TopByDegree[0].Degree)
	}
}

func TestStats_TopByDegreeCapped(t *testing.T) {
	g := NewGraph()
	for i := range 20 {
		id := string(rune('a' + i))
		g.AddNode(&Node{ID: id, Label: id})
	}
	// Hub edge from a to every other node.
	for i := 1; i < 20; i++ {
		g.AddEdge(&Edge{Source: "a", Target: string(rune('a' + i)), Relation: "calls"})
	}
	s := g.Stats()
	if len(s.TopByDegree) != 10 {
		t.Errorf("expected top-by-degree capped at 10, got %d", len(s.TopByDegree))
	}
}

func TestGodNodes_Basic(t *testing.T) {
	g := buildTestGraph()
	gods := g.GodNodes(3)
	if len(gods) == 0 {
		t.Fatal("expected non-empty god nodes")
	}
	if gods[0].Node.ID != "b" {
		t.Errorf("expected top god to be b, got %q", gods[0].Node.ID)
	}
}

func TestGodNodes_ExcludesFileNodes(t *testing.T) {
	g := buildTestGraph()
	g.AddNode(&Node{ID: "file_main", Label: "main.go", Kind: "file", SourceFile: "main.go", SourceLocation: "L1"})
	g.AddEdge(&Edge{Source: "file_main", Target: "a", Relation: "contains"})
	g.AddEdge(&Edge{Source: "file_main", Target: "b", Relation: "contains"})
	g.AddEdge(&Edge{Source: "file_main", Target: "c", Relation: "contains"})
	g.AddEdge(&Edge{Source: "file_main", Target: "d", Relation: "contains"})
	g.AddEdge(&Edge{Source: "file_main", Target: "e", Relation: "contains"})

	gods := g.GodNodes(10)
	for _, e := range gods {
		if e.Node.Kind == "file" {
			t.Errorf("GodNodes should exclude file-level nodes, got %+v", e.Node)
		}
	}
}

func TestGodNodes_ExcludesZeroDegree(t *testing.T) {
	g := buildTestGraph()
	gods := g.GodNodes(10)
	for _, e := range gods {
		if e.Node.ID == "f" {
			t.Error("GodNodes should exclude zero-degree node f")
		}
		if e.Degree == 0 {
			t.Errorf("GodNodes should exclude zero-degree, got %+v", e)
		}
	}
}

func TestGodNodes_DefaultTopN(t *testing.T) {
	g := buildTestGraph()
	gods := g.GodNodes(0) // default 10
	if len(gods) == 0 {
		t.Error("GodNodes(0) should still return entries")
	}
}

func TestGodNodes_EmptyGraph(t *testing.T) {
	g := NewGraph()
	gods := g.GodNodes(5)
	if len(gods) != 0 {
		t.Errorf("expected empty result for empty graph, got %d", len(gods))
	}
}

func TestCommunity_Found(t *testing.T) {
	g := buildTestGraph()
	cr, err := g.Community(0)
	if err != nil {
		t.Fatal(err)
	}
	if cr.ID != 0 {
		t.Errorf("expected ID=0, got %d", cr.ID)
	}
	if !containsID(cr.Nodes, "a") {
		t.Error("Community 0 should include a")
	}
	if !containsID(cr.Nodes, "b") {
		t.Error("Community 0 should include b")
	}
}

func TestCommunity_NotFound(t *testing.T) {
	g := buildTestGraph()
	cr, err := g.Community(99)
	if cr != nil {
		t.Errorf("expected nil, got %+v", cr)
	}
	if !errors.Is(err, ErrCommunityNotFound) {
		t.Errorf("expected ErrCommunityNotFound, got %v", err)
	}
}

func TestCommunity_Sorted(t *testing.T) {
	g := buildTestGraph()
	cr, err := g.Community(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cr.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes in community 0, got %d", len(cr.Nodes))
	}
	for i := 1; i < len(cr.Nodes); i++ {
		if cr.Nodes[i-1].Label > cr.Nodes[i].Label {
			t.Errorf("Community labels should be sorted; got %q before %q",
				cr.Nodes[i-1].Label, cr.Nodes[i].Label)
		}
	}
}

func TestCommunity_DisambiguatesSameLabelAcrossFiles(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "p1", Label: "Props", Kind: "interface", SourceFile: "src/card/index.tsx", SourceLocation: "L3", Community: 0})
	g.AddNode(&Node{ID: "p2", Label: "Props", Kind: "interface", SourceFile: "src/list/index.tsx", SourceLocation: "L8", Community: 0})
	g.Communities[0] = []string{"p1", "p2"}

	cr, err := g.Community(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cr.Nodes) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(cr.Nodes))
	}
	// Same-label entries ordered by source file (card before list).
	if cr.Nodes[0].SourceFile != "src/card/index.tsx" || cr.Nodes[1].SourceFile != "src/list/index.tsx" {
		t.Errorf("same-label entries should be ordered by source file; got %q then %q",
			cr.Nodes[0].SourceFile, cr.Nodes[1].SourceFile)
	}
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		file, loc, want string
	}{
		{"pkg/a.go", "L42", "pkg/a.go:42"},
		{"pkg/a.go", "", "pkg/a.go"},
		{"", "L1", ""},
		{"pkg/a.go", "42", "pkg/a.go:42"},
	}
	for _, tt := range tests {
		got := formatLocation(tt.file, tt.loc)
		if got != tt.want {
			t.Errorf("formatLocation(%q, %q) = %q, want %q", tt.file, tt.loc, got, tt.want)
		}
	}
}
