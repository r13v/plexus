package codegraph

import (
	"encoding/json"
	"reflect"
	"testing"
)

func roundTrip(t *testing.T, r *Result) *Result {
	t.Helper()
	data, err := r.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("JSON output is not valid JSON: %s", data)
	}
	var got Result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &got
}

func TestRenderJSON_Stats(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "stats"})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "stats" {
		t.Errorf("Kind = %q, want stats", got.Kind)
	}
	if got.Stats == nil {
		t.Fatal("Stats is nil")
	}
	if got.Stats.NodeCount != res.Stats.NodeCount {
		t.Errorf("NodeCount = %d, want %d", got.Stats.NodeCount, res.Stats.NodeCount)
	}
	if got.Stats.EdgeCount != res.Stats.EdgeCount {
		t.Errorf("EdgeCount = %d, want %d", got.Stats.EdgeCount, res.Stats.EdgeCount)
	}
	if got.Stats.CommunityCount != res.Stats.CommunityCount {
		t.Errorf("CommunityCount = %d, want %d", got.Stats.CommunityCount, res.Stats.CommunityCount)
	}
	if !reflect.DeepEqual(got.Stats.TopByDegree, res.Stats.TopByDegree) {
		t.Errorf("TopByDegree mismatch\n got: %+v\nwant: %+v", got.Stats.TopByDegree, res.Stats.TopByDegree)
	}
}

func TestRenderJSON_Gods(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "gods", TopN: 5})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "gods" {
		t.Errorf("Kind = %q, want gods", got.Kind)
	}
	if !reflect.DeepEqual(got.Gods, res.Gods) {
		t.Errorf("Gods mismatch\n got: %+v\nwant: %+v", got.Gods, res.Gods)
	}
}

func TestRenderJSON_Node(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "node", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "node" {
		t.Errorf("Kind = %q, want node", got.Kind)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "func_a" {
		t.Errorf("Nodes mismatch: %+v", got.Nodes)
	}
}

func TestRenderJSON_Neighbors(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "neighbors", Label: "FuncA()"})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "neighbors" {
		t.Errorf("Kind = %q, want neighbors", got.Kind)
	}
	if got.Neighbors == nil || got.Neighbors.Node == nil || got.Neighbors.Node.ID != "func_a" {
		t.Errorf("Neighbors.Node mismatch: %+v", got.Neighbors)
	}
	if len(got.Neighbors.Neighbors) != len(res.Neighbors.Neighbors) {
		t.Errorf("len(Neighbors.Neighbors) = %d, want %d", len(got.Neighbors.Neighbors), len(res.Neighbors.Neighbors))
	}
}

func TestRenderJSON_Callers(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "callers", Label: "FuncB()"})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "callers" {
		t.Errorf("Kind = %q, want callers", got.Kind)
	}
	if got.Callers == nil || got.Callers.Node.ID != "func_b" {
		t.Errorf("Callers.Node mismatch: %+v", got.Callers)
	}
	if len(got.Callers.Callers) != len(res.Callers.Callers) {
		t.Errorf("len(Callers.Callers) = %d, want %d", len(got.Callers.Callers), len(res.Callers.Callers))
	}
}

func TestRenderJSON_Path(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "path", Source: "FuncA()", Target: "FuncC()"})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "path" {
		t.Errorf("Kind = %q, want path", got.Kind)
	}
	if got.Path == nil {
		t.Fatal("Path is nil")
	}
	if got.Path.Source.ID != "func_a" || got.Path.Target.ID != "func_c" {
		t.Errorf("Path source/target mismatch: %+v / %+v", got.Path.Source, got.Path.Target)
	}
	if len(got.Path.Nodes) != len(res.Path.Nodes) {
		t.Errorf("len(Path.Nodes) = %d, want %d", len(got.Path.Nodes), len(res.Path.Nodes))
	}
}

func TestRenderJSON_Community(t *testing.T) {
	g := buildRenderTextGraph()
	zero := 0
	res, err := Dispatch(g, &Input{Action: "community", CommunityID: &zero})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "community" {
		t.Errorf("Kind = %q, want community", got.Kind)
	}
	if got.Community == nil || got.Community.ID != 0 {
		t.Errorf("Community mismatch: %+v", got.Community)
	}
	if len(got.Community.Nodes) != len(res.Community.Nodes) {
		t.Errorf("len(Community.Nodes) = %d, want %d", len(got.Community.Nodes), len(res.Community.Nodes))
	}
}

func TestRenderJSON_Subgraph(t *testing.T) {
	g := buildRenderTextGraph()
	res, err := Dispatch(g, &Input{Action: "query", Query: "FuncA", Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip(t, res)
	if got.Kind != "subgraph" {
		t.Errorf("Kind = %q, want subgraph", got.Kind)
	}
	if got.Subgraph == nil {
		t.Fatal("Subgraph is nil")
	}
	if len(got.Subgraph.Nodes) != len(res.Subgraph.Nodes) {
		t.Errorf("len(Subgraph.Nodes) = %d, want %d", len(got.Subgraph.Nodes), len(res.Subgraph.Nodes))
	}
}

func TestRenderJSON_OmitsEmpty(t *testing.T) {
	r := &Result{Kind: "stats", Stats: &StatsResult{NodeCount: 1}}
	data, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"nodes", "edges", "gods", "path", "community", "neighbors", "callers", "subgraph"} {
		if _, ok := raw[k]; ok {
			t.Errorf("expected %q to be omitted, got: %v", k, raw[k])
		}
	}
	if _, ok := raw["stats"]; !ok {
		t.Error("expected stats to be present")
	}
}

func TestRenderJSON_NilResult(t *testing.T) {
	var r *Result
	data, err := r.JSON()
	if err != nil {
		return
	}
	if string(data) != "null" {
		t.Fatalf("expected %q for nil receiver, got %q", "null", string(data))
	}
}
