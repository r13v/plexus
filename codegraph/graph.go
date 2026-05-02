package codegraph

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Node represents a code entity (function, type, method, file).
type Node struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Kind           string `json:"kind"`
	SourceFile     string `json:"source_file"`
	SourceLocation string `json:"source_location"`
	Community      int    `json:"community,omitempty"`
}

// Edge represents a relationship between two code entities.
type Edge struct {
	Source         string `json:"source"`
	Target         string `json:"target"`
	Relation       string `json:"relation"`
	SourceFile     string `json:"source_file,omitempty"`
	SourceLocation string `json:"source_location,omitempty"`
}

// Graph holds the full code graph with adjacency lists for fast traversal.
type Graph struct {
	Nodes       map[string]*Node   `json:"nodes"`
	Edges       []Edge             `json:"edges"`
	Adj         map[string][]*Edge `json:"-"`
	RevAdj      map[string][]*Edge `json:"-"`
	Communities map[int][]string   `json:"communities,omitempty"`
}

// SchemaVersion is bumped whenever extraction logic, node ID generation, or
// edge semantics change. A cache file with a different version is treated as
// stale regardless of HEAD SHA.
const SchemaVersion = 4

// gobCache is the on-disk gob serialization shape. Adj/RevAdj are excluded:
// gob ignores `json:"-"` tags, so encoding the Graph verbatim would double
// cache size and lose pointer-aliasing into the Edges slice on decode.
type gobCache struct {
	SchemaVersion int
	HeadCommit    string
	Nodes         map[string]*Node
	Edges         []Edge
	Communities   map[int][]string
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9_./-]`)

// MakeID builds a deterministic node ID from parts: joined by "_", lowercased,
// non-alphanumeric characters replaced with "_".
func MakeID(parts ...string) string {
	joined := strings.Join(parts, "_")
	joined = strings.ToLower(joined)
	return nonAlphaNum.ReplaceAllString(joined, "_")
}

// NewGraph creates an empty Graph with initialized maps.
func NewGraph() *Graph {
	return &Graph{
		Nodes:       make(map[string]*Node),
		Edges:       nil,
		Adj:         make(map[string][]*Edge),
		RevAdj:      make(map[string][]*Edge),
		Communities: make(map[int][]string),
	}
}

// AddNode adds or replaces a node by ID (last-wins dedup).
func (g *Graph) AddNode(n *Node) {
	if n == nil {
		return
	}
	g.Nodes[n.ID] = n
}

// AddEdge adds an edge if both source and target nodes exist.
// Populates Adj and RevAdj. Skips edges with unknown endpoints.
// Detects slice reallocation and rebuilds adjacency maps when it occurs.
func (g *Graph) AddEdge(e *Edge) bool {
	if _, ok := g.Nodes[e.Source]; !ok {
		return false
	}
	if _, ok := g.Nodes[e.Target]; !ok {
		return false
	}
	var oldBase *Edge
	if len(g.Edges) > 0 {
		oldBase = &g.Edges[0]
	}
	g.Edges = append(g.Edges, *e)
	if oldBase != nil && &g.Edges[0] != oldBase {
		g.rebuildAdj()
	} else {
		ep := &g.Edges[len(g.Edges)-1]
		g.Adj[e.Source] = append(g.Adj[e.Source], ep)
		g.RevAdj[e.Target] = append(g.RevAdj[e.Target], ep)
	}
	return true
}

// rebuildAdj reconstructs Adj and RevAdj from the Edges slice.
func (g *Graph) rebuildAdj() {
	g.Adj = make(map[string][]*Edge, len(g.Nodes))
	g.RevAdj = make(map[string][]*Edge, len(g.Nodes))
	for i := range g.Edges {
		ep := &g.Edges[i]
		g.Adj[ep.Source] = append(g.Adj[ep.Source], ep)
		g.RevAdj[ep.Target] = append(g.RevAdj[ep.Target], ep)
	}
}

// Save persists the graph and HEAD SHA to a gob cache file. Writes are atomic
// on POSIX: the encoder writes to a sibling tempfile, then renames into place.
// A crash mid-write leaves the prior cache (if any) intact.
func (g *Graph) Save(path, headSHA string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("codegraph: mkdir %s: %w", dir, err)
	}
	cache := gobCache{
		SchemaVersion: SchemaVersion,
		HeadCommit:    headSHA,
		Nodes:         g.Nodes,
		Edges:         g.Edges,
		Communities:   g.Communities,
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("codegraph: tempfile in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := gob.NewEncoder(tmp).Encode(&cache); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("codegraph: encode: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("codegraph: sync tempfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("codegraph: close tempfile: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("codegraph: rename %s -> %s: %w", tmpPath, path, err)
	}
	cleanup = false
	return nil
}

// LoadGraph reads a cache file, returns the graph, cached HEAD SHA, and any error.
// Adj and RevAdj are rebuilt from the loaded edges.
func LoadGraph(path string) (*Graph, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("codegraph: read %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	var cache gobCache
	if err := gob.NewDecoder(f).Decode(&cache); err != nil {
		return nil, "", fmt.Errorf("codegraph: decode %s: %w", path, err)
	}
	if cache.SchemaVersion != SchemaVersion {
		return nil, "", fmt.Errorf("codegraph: schema version mismatch in %s: got %d, want %d", path, cache.SchemaVersion, SchemaVersion)
	}
	g := &Graph{
		Nodes:       cache.Nodes,
		Edges:       cache.Edges,
		Communities: cache.Communities,
	}
	if g.Nodes == nil {
		g.Nodes = make(map[string]*Node)
	}
	if g.Communities == nil {
		g.Communities = make(map[int][]string)
	}
	g.rebuildAdj()
	return g, cache.HeadCommit, nil
}
