package codegraph

import (
	"maps"
	"math/rand/v2"
	"slices"

	"gonum.org/v1/gonum/graph/community"
	"gonum.org/v1/gonum/graph/simple"
)

// Cluster assigns community IDs to nodes using Louvain modularity optimization.
// Directed edges are projected to undirected for clustering. Isolates (nodes
// with no edges) each receive their own single-node community.
func Cluster(g *Graph) {
	if len(g.Nodes) == 0 {
		return
	}

	sortedIDs := slices.Sorted(maps.Keys(g.Nodes))

	idToInt := make(map[string]int64, len(g.Nodes))
	intToID := make(map[int64]string, len(g.Nodes))
	for idx, id := range sortedIDs {
		idToInt[id] = int64(idx)
		intToID[int64(idx)] = id
	}

	ug := simple.NewUndirectedGraph()
	for _, id := range sortedIDs {
		ug.AddNode(simple.Node(idToInt[id]))
	}

	for _, e := range g.Edges {
		src := idToInt[e.Source]
		tgt := idToInt[e.Target]
		if src != tgt && !ug.HasEdgeBetween(src, tgt) {
			ug.SetEdge(simple.Edge{F: simple.Node(src), T: simple.Node(tgt)})
		}
	}

	// Seed the Louvain RNG deterministically so a build against the same HEAD
	// always produces the same community IDs.
	reduced := community.Modularize(ug, 1, rand.NewPCG(0, 0))
	comms := reduced.Communities()

	g.Communities = make(map[int][]string)
	for i, comm := range comms {
		for _, n := range comm {
			nodeID := intToID[n.ID()]
			g.Nodes[nodeID].Community = i
			g.Communities[i] = append(g.Communities[i], nodeID)
		}
	}
}
