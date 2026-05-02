package codegraph

import (
	"fmt"
	"strings"
)

// TextOpts controls Result.Text rendering. TokenBudget bounds output for the
// "subgraph" and "path" kinds — large traversal results that can blow up an
// agent's context window. The other kinds (stats, gods, community, node,
// neighbors, callers) are tabular and intrinsically bounded by topN, single
// label lookups, or community size, so TokenBudget is ignored for them.
type TextOpts struct {
	TokenBudget int
}

// Text renders the result as a human-readable string, dispatching on r.Kind.
func (r *Result) Text(opts TextOpts) string {
	if r == nil {
		return ""
	}
	switch r.Kind {
	case "stats":
		return renderStats(r.Stats)
	case "gods":
		return renderGods(r.Gods)
	case "subgraph":
		return renderSubgraph(r.Nodes, r.Edges, resolveBudget(opts.TokenBudget))
	case "path":
		if r.Path == nil || len(r.Path.Nodes) == 0 {
			return "no path found\n"
		}
		return renderSubgraph(r.Nodes, r.Edges, resolveBudget(opts.TokenBudget))
	case "community":
		return renderCommunity(r.Community)
	case "node":
		return renderSubgraph(r.Nodes, nil, 0)
	case "neighbors":
		return renderNeighbors(r.Neighbors)
	case "callers":
		return renderCallers(r.Callers)
	default:
		return ""
	}
}

func resolveBudget(b int) int {
	if b <= 0 {
		return DefaultTokenBudget
	}
	if b > MaxTokenBudget {
		return MaxTokenBudget
	}
	return b
}

// renderSubgraph produces graphify-style text output for a set of nodes and
// edges. tokenBudget <= 0 disables truncation. When >0, output is truncated
// at tokenBudget*3 characters with a notice appended.
func renderSubgraph(nodes []*Node, edges []*Edge, tokenBudget int) string {
	var b strings.Builder
	for _, n := range nodes {
		if loc := formatLocation(n.SourceFile, n.SourceLocation); loc != "" {
			fmt.Fprintf(&b, "NODE %s  %s  [community=%d]\n", n.Label, loc, n.Community)
		} else {
			fmt.Fprintf(&b, "NODE %s  [community=%d]\n", n.Label, n.Community)
		}
	}
	labelOf := make(map[string]string, len(nodes))
	for _, n := range nodes {
		labelOf[n.ID] = n.Label
	}
	for _, e := range edges {
		src, tgt := e.Source, e.Target
		if l, ok := labelOf[src]; ok {
			src = l
		}
		if l, ok := labelOf[tgt]; ok {
			tgt = l
		}
		if loc := formatLocation(e.SourceFile, e.SourceLocation); loc != "" {
			fmt.Fprintf(&b, "EDGE %s --%s--> %s  [at %s]\n", src, e.Relation, tgt, loc)
		} else {
			fmt.Fprintf(&b, "EDGE %s --%s--> %s\n", src, e.Relation, tgt)
		}
	}

	out := b.String()
	if tokenBudget <= 0 {
		return out
	}
	limit := tokenBudget * 3
	if len(out) > limit {
		if idx := strings.LastIndex(out[:limit], "\n"); idx > 0 {
			out = out[:idx]
		} else {
			out = out[:limit]
		}
		out += "\n... (truncated, token budget exceeded)\n"
	}
	return out
}

func renderStats(s *StatsResult) string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Nodes: %d\n", s.NodeCount)
	fmt.Fprintf(&b, "Edges: %d\n", s.EdgeCount)
	fmt.Fprintf(&b, "Communities: %d\n", s.CommunityCount)
	if len(s.TopByDegree) > 0 {
		b.WriteString("\nTop by degree:\n")
		for _, e := range s.TopByDegree {
			writeDegEntry(&b, e)
		}
	}
	return b.String()
}

func renderGods(gods []DegEntry) string {
	if len(gods) == 0 {
		return "no god nodes found"
	}
	var b strings.Builder
	b.WriteString("God nodes (highest degree):\n")
	for _, e := range gods {
		writeDegEntry(&b, e)
	}
	return b.String()
}

func writeDegEntry(b *strings.Builder, e DegEntry) {
	if loc := formatLocation(e.Node.SourceFile, e.Node.SourceLocation); loc != "" {
		fmt.Fprintf(b, "  %s  %s  (degree=%d)\n", e.Node.Label, loc, e.Degree)
	} else {
		fmt.Fprintf(b, "  %s  (degree=%d)\n", e.Node.Label, e.Degree)
	}
}

func renderCommunity(c *CommunityResult) string {
	if c == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Community %d (%d members):\n", c.ID, len(c.Nodes))
	for _, n := range c.Nodes {
		if loc := formatLocation(n.SourceFile, n.SourceLocation); loc != "" {
			fmt.Fprintf(&b, "  %s  %s\n", n.Label, loc)
		} else {
			fmt.Fprintf(&b, "  %s\n", n.Label)
		}
	}
	return b.String()
}

func renderNeighbors(nb *NeighborsResult) string {
	if nb == nil {
		return ""
	}
	if len(nb.Edges) == 0 {
		return "no outgoing edges from: " + nb.Node.Label
	}
	nodes := append([]*Node{nb.Node}, nb.Neighbors...)
	return renderSubgraph(nodes, nb.Edges, 0)
}

func renderCallers(c *CallersResult) string {
	if c == nil {
		return ""
	}
	if len(c.Edges) == 0 {
		return "no callers found for: " + c.Node.Label
	}
	nodes := append([]*Node{c.Node}, c.Callers...)
	return renderSubgraph(nodes, c.Edges, 0)
}
