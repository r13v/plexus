package codegraph

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"
)

func TestMakeID(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{"Foo", "Bar"}, "foo_bar"},
		{[]string{"pkg/util.go", "ReadAll"}, "pkg/util.go_readall"},
		{[]string{"Hello World!"}, "hello_world_"},
		{[]string{"foo-bar.go"}, "foo-bar.go"},
		{[]string{"foo/bar.go"}, "foo/bar.go"},
		{[]string{"already_lower"}, "already_lower"},
		{[]string{"A", "B", "C"}, "a_b_c"},
		{[]string{""}, ""},
	}
	for _, tt := range tests {
		got := MakeID(tt.parts...)
		if got != tt.want {
			t.Errorf("MakeID(%v) = %q, want %q", tt.parts, got, tt.want)
		}
	}
}

func TestAddNode_Dedup(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a", Label: "first"})
	g.AddNode(&Node{ID: "a", Label: "second"})

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	if g.Nodes["a"].Label != "second" {
		t.Errorf("expected last-wins dedup, got label %q", g.Nodes["a"].Label)
	}
}

func TestAddNode_Nil(t *testing.T) {
	g := NewGraph()
	g.AddNode(nil)
	if len(g.Nodes) != 0 {
		t.Fatal("nil AddNode should be a no-op")
	}
}

func TestAddEdge_Valid(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "A"})
	g.AddNode(&Node{ID: "b", Label: "B"})

	ok := g.AddEdge(&Edge{Source: "a", Target: "b", Relation: "calls"})
	if !ok {
		t.Fatal("expected AddEdge to succeed")
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if len(g.Adj["a"]) != 1 {
		t.Errorf("expected Adj[a] to have 1 entry, got %d", len(g.Adj["a"]))
	}
	if len(g.RevAdj["b"]) != 1 {
		t.Errorf("expected RevAdj[b] to have 1 entry, got %d", len(g.RevAdj["b"]))
	}
}

func TestAddEdge_UnknownSource(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "b", Label: "B"})

	ok := g.AddEdge(&Edge{Source: "missing", Target: "b", Relation: "calls"})
	if ok {
		t.Fatal("expected AddEdge to fail for unknown source")
	}
	if len(g.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(g.Edges))
	}
}

func TestAddEdge_UnknownTarget(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "A"})

	ok := g.AddEdge(&Edge{Source: "a", Target: "missing", Relation: "calls"})
	if ok {
		t.Fatal("expected AddEdge to fail for unknown target")
	}
}

func TestAddEdge_AdjRevAdj(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "A"})
	g.AddNode(&Node{ID: "b", Label: "B"})
	g.AddNode(&Node{ID: "c", Label: "C"})

	g.AddEdge(&Edge{Source: "a", Target: "b", Relation: "calls"})
	g.AddEdge(&Edge{Source: "a", Target: "c", Relation: "imports"})
	g.AddEdge(&Edge{Source: "b", Target: "c", Relation: "calls"})

	if len(g.Adj["a"]) != 2 {
		t.Errorf("expected Adj[a]=2, got %d", len(g.Adj["a"]))
	}
	if len(g.RevAdj["c"]) != 2 {
		t.Errorf("expected RevAdj[c]=2, got %d", len(g.RevAdj["c"]))
	}
	if len(g.Adj["c"]) != 0 {
		t.Errorf("expected Adj[c]=0, got %d", len(g.Adj["c"]))
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "fn_main", Label: "main()", Kind: "function", SourceFile: "main.go", SourceLocation: "L1"})
	g.AddNode(&Node{ID: "fn_helper", Label: "helper()", Kind: "function", SourceFile: "util.go", SourceLocation: "L10"})
	g.AddNode(&Node{ID: "type_config", Label: "Config", Kind: "type", SourceFile: "config.go", SourceLocation: "L5", Community: 1})
	g.AddEdge(&Edge{Source: "fn_main", Target: "fn_helper", Relation: "calls", SourceFile: "main.go", SourceLocation: "L3"})
	g.AddEdge(&Edge{Source: "fn_main", Target: "type_config", Relation: "imports"})
	g.Communities[0] = []string{"fn_main", "fn_helper"}
	g.Communities[1] = []string{"type_config"}

	dir := t.TempDir()
	path := filepath.Join(dir, "plexus", "code_graph.gob")
	headSHA := "abc123def"

	if err := g.Save(path, headSHA); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	loaded, sha, err := LoadGraph(path)
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if sha != headSHA {
		t.Errorf("HeadCommit = %q, want %q", sha, headSHA)
	}

	if len(loaded.Nodes) != len(g.Nodes) {
		t.Errorf("loaded %d nodes, want %d", len(loaded.Nodes), len(g.Nodes))
	}
	for id, orig := range g.Nodes {
		ln, ok := loaded.Nodes[id]
		if !ok {
			t.Errorf("missing node %q after load", id)
			continue
		}
		if ln.Label != orig.Label || ln.Kind != orig.Kind || ln.SourceFile != orig.SourceFile {
			t.Errorf("node %q mismatch: got %+v, want %+v", id, ln, orig)
		}
	}

	if len(loaded.Edges) != len(g.Edges) {
		t.Errorf("loaded %d edges, want %d", len(loaded.Edges), len(g.Edges))
	}
	for i, e := range g.Edges {
		le := loaded.Edges[i]
		if le.Source != e.Source || le.Target != e.Target || le.Relation != e.Relation {
			t.Errorf("edge %d mismatch: got %+v, want %+v", i, le, e)
		}
	}

	if len(loaded.Adj["fn_main"]) != 2 {
		t.Errorf("Adj[fn_main] = %d, want 2", len(loaded.Adj["fn_main"]))
	}
	if len(loaded.RevAdj["fn_helper"]) != 1 {
		t.Errorf("RevAdj[fn_helper] = %d, want 1", len(loaded.RevAdj["fn_helper"]))
	}
	if len(loaded.RevAdj["type_config"]) != 1 {
		t.Errorf("RevAdj[type_config] = %d, want 1", len(loaded.RevAdj["type_config"]))
	}

	if len(loaded.Communities) != len(g.Communities) {
		t.Errorf("loaded %d communities, want %d", len(loaded.Communities), len(g.Communities))
	}
}

func TestSave_BinaryFormat(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Label: "A"})
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.gob")
	if err := g.Save(path, "sha"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// gob streams begin with a non-printable type-id varint, never with '{'.
	if len(data) == 0 || data[0] == '{' || data[0] == '[' {
		t.Errorf("cache appears to be JSON text, not gob (first byte = %q)", data[0])
	}
}

func TestLoadGraph_NotFound(t *testing.T) {
	_, _, err := LoadGraph("/nonexistent/path.gob")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadGraph_MalformedGob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gob")
	if err := os.WriteFile(path, []byte("{not gob}"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadGraph(path)
	if err == nil {
		t.Fatal("expected error for malformed gob")
	}
}

func TestLoadGraph_SchemaMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.gob")
	g := NewGraph()
	if err := g.Save(path, "abc"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	// Corrupt the schema by re-encoding with a wrong version.
	bad := gobCache{SchemaVersion: SchemaVersion - 1, HeadCommit: "abc"}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := gob.NewEncoder(f).Encode(&bad); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadGraph(path)
	if err == nil {
		t.Fatal("expected error for schema version mismatch")
	}
}

func TestLoadGraph_EmptyGraph(t *testing.T) {
	g := NewGraph()
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.gob")
	if err := g.Save(path, "sha1"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, sha, err := LoadGraph(path)
	if err != nil {
		t.Fatalf("LoadGraph: %v", err)
	}
	if sha != "sha1" {
		t.Errorf("sha = %q, want %q", sha, "sha1")
	}
	if len(loaded.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(loaded.Nodes))
	}
	if loaded.Adj == nil || loaded.RevAdj == nil {
		t.Error("Adj/RevAdj should be initialized even for empty graph")
	}
}
