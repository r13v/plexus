package codegraph

import (
	"errors"
	"fmt"
)

// Result is the structured payload returned by Dispatch (Task 4.1b) and
// already used by callers of the per-action methods on Graph. Renderers
// (Result.Text/JSON/DOT, added in Tasks 4.2-4.4) format it for output.
type Result struct {
	Kind      string           `json:"kind"`
	Nodes     []*Node          `json:"nodes,omitzero"`
	Edges     []*Edge          `json:"edges,omitzero"`
	Stats     *StatsResult     `json:"stats,omitzero"`
	Gods      []DegEntry       `json:"gods,omitzero"`
	Path      *PathResult      `json:"path,omitzero"`
	Community *CommunityResult `json:"community,omitzero"`
	Neighbors *NeighborsResult `json:"neighbors,omitzero"`
	Callers   *CallersResult   `json:"callers,omitzero"`
	Subgraph  *SubgraphResult  `json:"subgraph,omitzero"`
}

// StatsResult is the structured form of Graph.Stats().
type StatsResult struct {
	NodeCount      int        `json:"node_count"`
	EdgeCount      int        `json:"edge_count"`
	CommunityCount int        `json:"community_count"`
	TopByDegree    []DegEntry `json:"top_by_degree,omitzero"`
}

// DegEntry pairs a node with its total degree (in + out).
type DegEntry struct {
	Node   *Node `json:"node"`
	Degree int   `json:"degree"`
}

// PathResult is the structured form of a shortest-path query.
type PathResult struct {
	Source *Node   `json:"source"`
	Target *Node   `json:"target"`
	Nodes  []*Node `json:"nodes,omitzero"`
	Edges  []*Edge `json:"edges,omitzero"`
}

// CommunityResult is the structured form of Graph.Community(id).
type CommunityResult struct {
	ID    int     `json:"id"`
	Nodes []*Node `json:"nodes,omitzero"`
}

// NeighborsResult is the structured form of Graph.GetNeighbors().
type NeighborsResult struct {
	Node      *Node   `json:"node"`
	Neighbors []*Node `json:"neighbors,omitzero"`
	Edges     []*Edge `json:"edges,omitzero"`
}

// CallersResult is the structured form of Graph.GetCallers().
type CallersResult struct {
	Node    *Node   `json:"node"`
	Callers []*Node `json:"callers,omitzero"`
	Edges   []*Edge `json:"edges,omitzero"`
}

// SubgraphResult is the structured form of Graph.Query().
type SubgraphResult struct {
	Seeds []*Node `json:"seeds,omitzero"`
	Nodes []*Node `json:"nodes,omitzero"`
	Edges []*Edge `json:"edges,omitzero"`
}

// Sentinel errors returned by the query methods.
var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrCommunityNotFound = errors.New("community not found or empty")
	ErrNoQueryTerms      = errors.New("no query terms provided")
	ErrNoMatch           = errors.New("no matching nodes")
)

// AmbiguousLabelError is returned when a label resolves to multiple nodes.
// The CLI/dispatch layer can format it; callers can also branch on it via
// errors.As to surface candidates.
type AmbiguousLabelError struct {
	Label string
	Count int
}

func (e *AmbiguousLabelError) Error() string {
	return fmt.Sprintf("ambiguous label %q matches %d nodes — use full node ID", e.Label, e.Count)
}
