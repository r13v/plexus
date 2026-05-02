package codegraph

import (
	"cmp"
	"slices"
	"strings"
)

// BFS returns nodes reachable from startIDs within the given depth via forward edges.
// If relationFilter is non-empty, only edges with that relation are followed.
// Returns visited nodes in BFS order (excluding start nodes that don't exist).
func (g *Graph) BFS(startIDs []string, depth int, relationFilter string) []*Node {
	if depth <= 0 {
		depth = 3
	}
	visited := make(map[string]bool, len(startIDs))
	var result []*Node

	type item struct {
		id string
		d  int
	}
	var queue []item

	for _, id := range startIDs {
		if n, ok := g.Nodes[id]; ok && !visited[id] {
			visited[id] = true
			result = append(result, n)
			queue = append(queue, item{id, 0})
		}
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.d >= depth {
			continue
		}
		for _, e := range g.Adj[cur.id] {
			if relationFilter != "" && e.Relation != relationFilter {
				continue
			}
			if !visited[e.Target] {
				visited[e.Target] = true
				if n, ok := g.Nodes[e.Target]; ok {
					result = append(result, n)
					queue = append(queue, item{e.Target, cur.d + 1})
				}
			}
		}
	}

	return result
}

// DFS returns nodes reachable from startIDs within the given depth via forward edges.
// If relationFilter is non-empty, only edges with that relation are followed.
// Returns visited nodes in DFS pre-order.
func (g *Graph) DFS(startIDs []string, depth int, relationFilter string) []*Node {
	if depth <= 0 {
		depth = 3
	}
	visited := make(map[string]bool, len(startIDs))
	var result []*Node

	var walk func(id string, d int)
	walk = func(id string, d int) {
		n, ok := g.Nodes[id]
		if !ok || visited[id] {
			return
		}
		visited[id] = true
		result = append(result, n)
		if d >= depth {
			return
		}
		for _, e := range g.Adj[id] {
			if relationFilter != "" && e.Relation != relationFilter {
				continue
			}
			walk(e.Target, d+1)
		}
	}

	for _, id := range startIDs {
		walk(id, 0)
	}
	return result
}

// ShortestPath finds the shortest path from src to tgt using BFS on forward edges.
// maxHops limits the search depth; 0 defaults to 10. Returns nil, nil if no path exists.
func (g *Graph) ShortestPath(src, tgt string, maxHops int) ([]*Node, []*Edge) {
	if maxHops <= 0 {
		maxHops = 10
	}
	if _, ok := g.Nodes[src]; !ok {
		return nil, nil
	}
	if _, ok := g.Nodes[tgt]; !ok {
		return nil, nil
	}
	if src == tgt {
		return []*Node{g.Nodes[src]}, nil
	}

	type state struct {
		id   string
		dist int
	}

	prev := make(map[string]string)
	prevEdge := make(map[string]*Edge)
	visited := map[string]bool{src: true}
	queue := []state{{src, 0}}

	found := false
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.dist >= maxHops {
			continue
		}
		for _, e := range g.Adj[cur.id] {
			if visited[e.Target] {
				continue
			}
			visited[e.Target] = true
			prev[e.Target] = cur.id
			prevEdge[e.Target] = e
			if e.Target == tgt {
				found = true
				break
			}
			queue = append(queue, state{e.Target, cur.dist + 1})
		}
		if found {
			break
		}
	}

	if !found {
		return nil, nil
	}

	var nodeIDs []string
	var edges []*Edge
	for cur := tgt; cur != src; cur = prev[cur] {
		nodeIDs = append(nodeIDs, cur)
		edges = append(edges, prevEdge[cur])
	}
	nodeIDs = append(nodeIDs, src)

	// Reverse to get src→tgt order.
	slices.Reverse(nodeIDs)
	slices.Reverse(edges)

	nodes := make([]*Node, len(nodeIDs))
	for i, id := range nodeIDs {
		nodes[i] = g.Nodes[id]
	}
	return nodes, edges
}

type scoredNode struct {
	node  *Node
	score int
}

// scoreNodesByKeyword scores nodes by case-insensitive substring match against terms.
// Each term matching label or source_file increments the score by 1.
// Returns up to 3 top-scoring nodes, sorted descending by score.
func (g *Graph) scoreNodesByKeyword(terms []string) []scoredNode {
	lower := make([]string, len(terms))
	for i, t := range terms {
		lower[i] = strings.ToLower(t)
	}

	var scored []scoredNode
	for _, n := range g.Nodes {
		s := 0
		labelLow := strings.ToLower(n.Label)
		fileLow := strings.ToLower(n.SourceFile)
		for _, t := range lower {
			if strings.Contains(labelLow, t) {
				s++
			}
			if strings.Contains(fileLow, t) {
				s++
			}
		}
		if s > 0 {
			scored = append(scored, scoredNode{node: n, score: s})
		}
	}

	slices.SortFunc(scored, func(a, b scoredNode) int {
		if a.score != b.score {
			return cmp.Compare(b.score, a.score)
		}
		return cmp.Compare(a.node.ID, b.node.ID)
	})

	if len(scored) > 3 {
		scored = scored[:3]
	}
	return scored
}

// formatLocation returns "path:line" using editor-style position notation.
// Strips a leading "L" from SourceLocation ("L42" → "42"). Returns "" when
// no file is set, or just the file when no line is set.
func formatLocation(file, loc string) string {
	if file == "" {
		return ""
	}
	line := strings.TrimPrefix(loc, "L")
	if line == "" {
		return file
	}
	return file + ":" + line
}

// Query splits a question into terms, finds seed nodes by keyword scoring,
// then expands via BFS or DFS into a SubgraphResult. mode is "bfs" or "dfs"
// (default "bfs"). depth defaults to 3. Token-budget truncation now lives in
// the renderer; callers pick a budget at format time.
func (g *Graph) Query(question, mode string, depth int) (*SubgraphResult, error) {
	if depth <= 0 {
		depth = 3
	}
	terms := strings.Fields(question)
	if len(terms) == 0 {
		return nil, ErrNoQueryTerms
	}

	scored := g.scoreNodesByKeyword(terms)
	if len(scored) == 0 {
		return nil, ErrNoMatch
	}

	seedIDs := make([]string, len(scored))
	seeds := make([]*Node, len(scored))
	for i, s := range scored {
		seedIDs[i] = s.node.ID
		seeds[i] = s.node
	}

	var nodes []*Node
	if strings.EqualFold(mode, "dfs") {
		nodes = g.DFS(seedIDs, depth, "")
	} else {
		nodes = g.BFS(seedIDs, depth, "")
	}

	edges := g.collectEdgesBetween(nodes)
	return &SubgraphResult{Seeds: seeds, Nodes: nodes, Edges: edges}, nil
}

// GetNode returns the node whose label matches (case-insensitive). Returns
// ErrNodeNotFound or *AmbiguousLabelError on the unhappy paths.
func (g *Graph) GetNode(label string) (*Node, error) {
	return g.findByLabel(label)
}

// GetNeighbors returns the outgoing edges from the node matching label and
// the unique neighbor nodes. If relationFilter is non-empty, only edges with
// that relation are included. Empty Edges means the node is a sink (or has
// no edges of the requested relation) — not an error.
func (g *Graph) GetNeighbors(label, relationFilter string) (*NeighborsResult, error) {
	n, err := g.findByLabel(label)
	if err != nil {
		return nil, err
	}

	res := &NeighborsResult{Node: n}
	seen := map[string]bool{n.ID: true}

	for _, e := range g.Adj[n.ID] {
		if relationFilter != "" && e.Relation != relationFilter {
			continue
		}
		res.Edges = append(res.Edges, e)
		if !seen[e.Target] {
			seen[e.Target] = true
			if tgt, ok := g.Nodes[e.Target]; ok {
				res.Neighbors = append(res.Neighbors, tgt)
			}
		}
	}
	return res, nil
}

// GetCallers returns nodes that call the node matching label (reverse "calls"
// edges only). Empty Edges means the node has no callers — not an error.
func (g *Graph) GetCallers(label string) (*CallersResult, error) {
	n, err := g.findByLabel(label)
	if err != nil {
		return nil, err
	}

	res := &CallersResult{Node: n}
	seen := map[string]bool{n.ID: true}

	for _, e := range g.RevAdj[n.ID] {
		if e.Relation != "calls" {
			continue
		}
		res.Edges = append(res.Edges, e)
		if !seen[e.Source] {
			seen[e.Source] = true
			if src, ok := g.Nodes[e.Source]; ok {
				res.Callers = append(res.Callers, src)
			}
		}
	}
	return res, nil
}

// Stats returns summary statistics: node/edge/community counts and the top-10
// nodes by total degree (in + out), with stable ordering by ID on ties.
func (g *Graph) Stats() *StatsResult {
	res := &StatsResult{
		NodeCount:      len(g.Nodes),
		EdgeCount:      len(g.Edges),
		CommunityCount: len(g.Communities),
	}

	degs := make([]DegEntry, 0, len(g.Nodes))
	for id, n := range g.Nodes {
		d := len(g.Adj[id]) + len(g.RevAdj[id])
		degs = append(degs, DegEntry{Node: n, Degree: d})
	}
	sortDegEntries(degs)

	top := min(10, len(degs))
	res.TopByDegree = degs[:top]
	return res
}

// GodNodes returns the top-N nodes by degree, excluding file-level nodes
// (kind == "file") and nodes with zero degree. topN defaults to 10.
func (g *Graph) GodNodes(topN int) []DegEntry {
	if topN <= 0 {
		topN = 10
	}

	var degs []DegEntry
	for id, n := range g.Nodes {
		if n.Kind == "file" {
			continue
		}
		d := len(g.Adj[id]) + len(g.RevAdj[id])
		if d == 0 {
			continue
		}
		degs = append(degs, DegEntry{Node: n, Degree: d})
	}
	sortDegEntries(degs)

	top := min(topN, len(degs))
	return degs[:top]
}

func sortDegEntries(degs []DegEntry) {
	slices.SortFunc(degs, func(a, b DegEntry) int {
		if a.Degree != b.Degree {
			return cmp.Compare(b.Degree, a.Degree)
		}
		return cmp.Compare(a.Node.ID, b.Node.ID)
	})
}

// Community lists nodes belonging to the given community ID, sorted by label
// then source file (stable for same-label nodes across files). Returns
// ErrCommunityNotFound when the community is missing or empty.
func (g *Graph) Community(id int) (*CommunityResult, error) {
	members, ok := g.Communities[id]
	if !ok || len(members) == 0 {
		return nil, ErrCommunityNotFound
	}

	var nodes []*Node
	for _, nodeID := range members {
		if n, ok := g.Nodes[nodeID]; ok {
			nodes = append(nodes, n)
		}
	}

	slices.SortFunc(nodes, func(a, b *Node) int {
		if a.Label != b.Label {
			return cmp.Compare(a.Label, b.Label)
		}
		return cmp.Compare(a.SourceFile, b.SourceFile)
	})

	if len(nodes) == 0 {
		return nil, ErrCommunityNotFound
	}
	return &CommunityResult{ID: id, Nodes: nodes}, nil
}

// findByLabel returns the unique node whose label matches (case-insensitive),
// or — as an escape hatch when labels collide — whose ID matches exactly.
// Returns ErrNodeNotFound when no node matches, or *AmbiguousLabelError when
// multiple nodes match by label and no node ID matches the input verbatim.
func (g *Graph) findByLabel(label string) (*Node, error) {
	if n, ok := g.Nodes[label]; ok {
		return n, nil
	}
	var found *Node
	var count int
	for _, n := range g.Nodes {
		if strings.EqualFold(n.Label, label) {
			count++
			if found == nil {
				found = n
			}
		}
	}
	if count > 1 {
		return nil, &AmbiguousLabelError{Label: label, Count: count}
	}
	if found == nil {
		return nil, ErrNodeNotFound
	}
	return found, nil
}

// collectEdgesBetween returns all edges where both source and target are in the node set.
func (g *Graph) collectEdgesBetween(nodes []*Node) []*Edge {
	set := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		set[n.ID] = true
	}
	var result []*Edge
	for i := range g.Edges {
		e := &g.Edges[i]
		if set[e.Source] && set[e.Target] {
			result = append(result, e)
		}
	}
	return result
}
