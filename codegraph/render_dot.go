package codegraph

import (
	"fmt"
	"strings"
)

// dotPalette is an 8-color palette used to fill nodes by community ID.
var dotPalette = []string{
	"#a6cee3",
	"#b2df8a",
	"#fb9a99",
	"#fdbf6f",
	"#cab2d6",
	"#ffff99",
	"#1f78b4",
	"#33a02c",
}

// dashedRelations are edge relations rendered with style=dashed.
// Other relations (calls/contains/method/inherits/...) render style=solid.
var dashedRelations = map[string]bool{
	"imports": true,
}

// DOT renders the result as a Graphviz DOT digraph. Kinds without a graph
// payload (stats, gods) return a comment line directing the caller to the
// text format.
func (r *Result) DOT() string {
	if r == nil {
		return ""
	}
	switch r.Kind {
	case "stats":
		return "// stats has no graph payload — use --format=text\n"
	case "gods":
		return "// gods has no graph payload — use --format=text\n"
	case "subgraph", "path", "node":
		return renderDOT(r.Nodes, r.Edges)
	case "neighbors":
		if r.Neighbors == nil {
			return renderDOT(nil, nil)
		}
		nodes := append([]*Node{r.Neighbors.Node}, r.Neighbors.Neighbors...)
		return renderDOT(nodes, r.Neighbors.Edges)
	case "callers":
		if r.Callers == nil {
			return renderDOT(nil, nil)
		}
		nodes := append([]*Node{r.Callers.Node}, r.Callers.Callers...)
		return renderDOT(nodes, r.Callers.Edges)
	case "community":
		if r.Community == nil {
			return renderDOT(nil, nil)
		}
		return renderDOT(r.Community.Nodes, nil)
	default:
		return ""
	}
}

func renderDOT(nodes []*Node, edges []*Edge) string {
	var b strings.Builder
	b.WriteString("digraph plexus {\n")
	b.WriteString("  rankdir=LR;\n")
	for _, n := range nodes {
		if n == nil {
			continue
		}
		shape := "ellipse"
		if n.Kind == "file" {
			shape = "box"
		}
		palette := len(dotPalette)
		idx := ((n.Community % palette) + palette) % palette
		fillcolor := dotPalette[idx]
		fmt.Fprintf(&b, "  %s [label=%s, shape=%s, style=filled, fillcolor=%s];\n",
			dotQuote(n.ID), dotQuote(n.Label), shape, dotQuote(fillcolor))
	}
	for _, e := range edges {
		if e == nil {
			continue
		}
		style := "solid"
		if dashedRelations[e.Relation] {
			style = "dashed"
		}
		fmt.Fprintf(&b, "  %s -> %s [label=%s, style=%s];\n",
			dotQuote(e.Source), dotQuote(e.Target), dotQuote(e.Relation), style)
	}
	b.WriteString("}\n")
	return b.String()
}

// dotQuote wraps s in double quotes, escaping inner backslashes and quotes.
func dotQuote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
