package codegraph

import (
	"fmt"
)

// Dispatch defaults and limits.
const (
	DefaultDepth       = 3
	MaxDepth           = 6
	DefaultTokenBudget = 4000
	MaxTokenBudget     = 50000
	DefaultTopN        = 10
	MaxTopN            = 100
)

// Input describes which action to perform on the graph, mirroring the ADK
// tool's CodeGraphInput but living in the codegraph package so both the CLI
// and the ADK tool can share the dispatch logic.
type Input struct {
	Action         string
	Query          string
	Label          string
	Source         string
	Target         string
	Mode           string
	Depth          int
	TopN           int
	RelationFilter string
	TokenBudget    int
	CommunityID    *int
}

// Dispatch executes a graph action described by in and returns a structured
// *Result. Renderers (Result.Text/JSON/DOT) format it for output.
//
// The path action treats "no path between source and target" as a successful
// query with an empty PathResult.Edges, so machine-readable formats stay
// contractual; the text renderer prints "no path found" for that case.
func Dispatch(g *Graph, in *Input) (*Result, error) {
	if g == nil {
		return nil, fmt.Errorf("code_graph: no graph loaded")
	}
	depth := clampVal(in.Depth, DefaultDepth, MaxDepth)
	topN := clampVal(in.TopN, DefaultTopN, MaxTopN)

	switch in.Action {
	case "stats":
		return &Result{Kind: "stats", Stats: g.Stats()}, nil

	case "gods":
		return &Result{Kind: "gods", Gods: g.GodNodes(topN)}, nil

	case "query":
		if in.Query == "" {
			return nil, fmt.Errorf("code_graph: query action requires 'query' field")
		}
		sg, err := g.Query(in.Query, in.Mode, depth)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: "subgraph", Subgraph: sg, Nodes: sg.Nodes, Edges: sg.Edges}, nil

	case "neighbors":
		if in.Label == "" {
			return nil, fmt.Errorf("code_graph: neighbors action requires 'label' field")
		}
		nb, err := g.GetNeighbors(in.Label, in.RelationFilter)
		if err != nil {
			return nil, err
		}
		nodes := append([]*Node{nb.Node}, nb.Neighbors...)
		return &Result{Kind: "neighbors", Neighbors: nb, Nodes: nodes, Edges: nb.Edges}, nil

	case "callers":
		if in.Label == "" {
			return nil, fmt.Errorf("code_graph: callers action requires 'label' field")
		}
		c, err := g.GetCallers(in.Label)
		if err != nil {
			return nil, err
		}
		nodes := append([]*Node{c.Node}, c.Callers...)
		return &Result{Kind: "callers", Callers: c, Nodes: nodes, Edges: c.Edges}, nil

	case "path":
		if in.Source == "" || in.Target == "" {
			return nil, fmt.Errorf("code_graph: path action requires 'source' and 'target' fields")
		}
		srcNode, err := g.findByLabel(in.Source)
		if err != nil {
			return nil, fmt.Errorf("source: %w", err)
		}
		tgtNode, err := g.findByLabel(in.Target)
		if err != nil {
			return nil, fmt.Errorf("target: %w", err)
		}
		nodes, edges := g.ShortestPath(srcNode.ID, tgtNode.ID, 0)
		pr := &PathResult{Source: srcNode, Target: tgtNode, Nodes: nodes, Edges: edges}
		return &Result{Kind: "path", Path: pr, Nodes: nodes, Edges: edges}, nil

	case "community":
		if in.CommunityID == nil {
			return nil, fmt.Errorf("code_graph: community action requires 'community_id' field")
		}
		if *in.CommunityID < 0 {
			return nil, fmt.Errorf("code_graph: community_id must be >= 0")
		}
		cr, err := g.Community(*in.CommunityID)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: "community", Community: cr, Nodes: cr.Nodes}, nil

	case "node":
		if in.Label == "" {
			return nil, fmt.Errorf("code_graph: node action requires 'label' field")
		}
		n, err := g.GetNode(in.Label)
		if err != nil {
			return nil, err
		}
		return &Result{Kind: "node", Nodes: []*Node{n}}, nil

	default:
		return nil, fmt.Errorf("code_graph: unknown action %q (valid: stats, gods, query, neighbors, callers, path, community, node)", in.Action)
	}
}

func clampVal(val, defaultVal, maxVal int) int {
	if val <= 0 {
		return defaultVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}
