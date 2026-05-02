package codegraph

import "testing"

func TestCluster_TwoCliques(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a1", Label: "A1", Kind: "function"})
	g.AddNode(&Node{ID: "a2", Label: "A2", Kind: "function"})
	g.AddNode(&Node{ID: "a3", Label: "A3", Kind: "function"})

	g.AddNode(&Node{ID: "b1", Label: "B1", Kind: "function"})
	g.AddNode(&Node{ID: "b2", Label: "B2", Kind: "function"})
	g.AddNode(&Node{ID: "b3", Label: "B3", Kind: "function"})

	// Clique A: triangle a1-a2-a3
	g.AddEdge(&Edge{Source: "a1", Target: "a2", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a2", Target: "a1", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a2", Target: "a3", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a3", Target: "a2", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a3", Target: "a1", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a1", Target: "a3", Relation: "calls"})

	// Clique B: triangle b1-b2-b3
	g.AddEdge(&Edge{Source: "b1", Target: "b2", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b2", Target: "b1", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b2", Target: "b3", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b3", Target: "b2", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b3", Target: "b1", Relation: "calls"})
	g.AddEdge(&Edge{Source: "b1", Target: "b3", Relation: "calls"})

	// Single bridge edge connecting the two cliques
	g.AddEdge(&Edge{Source: "a1", Target: "b1", Relation: "calls"})

	Cluster(g)

	commA := g.Nodes["a1"].Community
	if g.Nodes["a2"].Community != commA || g.Nodes["a3"].Community != commA {
		t.Errorf("clique A not in same community: a1=%d a2=%d a3=%d",
			g.Nodes["a1"].Community, g.Nodes["a2"].Community, g.Nodes["a3"].Community)
	}

	commB := g.Nodes["b1"].Community
	if g.Nodes["b2"].Community != commB || g.Nodes["b3"].Community != commB {
		t.Errorf("clique B not in same community: b1=%d b2=%d b3=%d",
			g.Nodes["b1"].Community, g.Nodes["b2"].Community, g.Nodes["b3"].Community)
	}

	if commA == commB {
		t.Errorf("expected two distinct communities, both got %d", commA)
	}

	if len(g.Communities) < 2 {
		t.Errorf("expected at least 2 communities, got %d", len(g.Communities))
	}

	if len(g.Communities[commA]) != 3 {
		t.Errorf("community %d should have 3 members, got %d", commA, len(g.Communities[commA]))
	}
	if len(g.Communities[commB]) != 3 {
		t.Errorf("community %d should have 3 members, got %d", commB, len(g.Communities[commB]))
	}
}

func TestCluster_Isolates(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "y1", Label: "Y1", Kind: "function"})
	g.AddNode(&Node{ID: "y2", Label: "Y2", Kind: "function"})
	g.AddEdge(&Edge{Source: "y1", Target: "y2", Relation: "calls"})

	g.AddNode(&Node{ID: "x1", Label: "X1", Kind: "function"})
	g.AddNode(&Node{ID: "x2", Label: "X2", Kind: "function"})

	Cluster(g)

	if g.Nodes["y1"].Community != g.Nodes["y2"].Community {
		t.Errorf("connected pair should share community: y1=%d y2=%d",
			g.Nodes["y1"].Community, g.Nodes["y2"].Community)
	}

	commX1 := g.Nodes["x1"].Community
	commX2 := g.Nodes["x2"].Community
	commY := g.Nodes["y1"].Community

	if commX1 == commX2 {
		t.Errorf("isolates should have different communities: both got %d", commX1)
	}
	if commX1 == commY {
		t.Errorf("isolate x1 should not share community with connected pair: both %d", commX1)
	}
	if commX2 == commY {
		t.Errorf("isolate x2 should not share community with connected pair: both %d", commX2)
	}

	if len(g.Communities[commX1]) != 1 {
		t.Errorf("isolate community %d should have 1 member, got %d", commX1, len(g.Communities[commX1]))
	}
	if len(g.Communities[commX2]) != 1 {
		t.Errorf("isolate community %d should have 1 member, got %d", commX2, len(g.Communities[commX2]))
	}
}

func TestCluster_EmptyGraph(t *testing.T) {
	g := NewGraph()
	Cluster(g)
	if len(g.Communities) != 0 {
		t.Errorf("expected 0 communities for empty graph, got %d", len(g.Communities))
	}
}
