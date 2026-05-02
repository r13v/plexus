package codegraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractGoFunc(t *testing.T) {
	src := []byte("package main\n\nfunc Hello() string {\n\treturn \"hi\"\n}\n")
	fe, err := ExtractFile("main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Hello()", "function")
	assertHasNode(t, fe.Nodes, "main.go", "file")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "contains", "main.go", "Hello()")
}

func TestExtractGoMethodWithPointerReceiver(t *testing.T) {
	src := []byte("package main\n\ntype Svc struct{}\n\nfunc (s *Svc) Run() string {\n\treturn \"running\"\n}\n")
	fe, err := ExtractFile("svc.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Svc", "struct")
	assertHasNode(t, fe.Nodes, "Svc.Run()", "method")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "method", "Svc", "Svc.Run()")
}

func TestExtractGoTypeDecl(t *testing.T) {
	src := []byte("package main\n\ntype Runner interface {\n\tRun() string\n}\n\ntype Config struct {\n\tName string\n}\n")
	fe, err := ExtractFile("types.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Runner", "interface")
	assertHasNode(t, fe.Nodes, "Config", "struct")
}

func TestExtractGoStructEmbedding(t *testing.T) {
	src := []byte("package main\n\ntype Base struct{}\n\ntype Child struct {\n\tBase\n\tname string\n}\n")
	fe, err := ExtractFile("embed.go", src)
	if err != nil {
		t.Fatal(err)
	}
	assertHasNode(t, fe.Nodes, "Base", "struct")
	assertHasNode(t, fe.Nodes, "Child", "struct")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "Child", "Base")
}

func TestExtractGoStructPointerEmbedding(t *testing.T) {
	src := []byte("package main\n\ntype Base struct{}\n\ntype Child struct {\n\t*Base\n}\n")
	fe, err := ExtractFile("pembed.go", src)
	if err != nil {
		t.Fatal(err)
	}
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "Child", "Base")
}

func TestExtractGoInterfaceEmbedding(t *testing.T) {
	src := []byte("package main\n\ntype Reader interface {\n\tRead() error\n}\n\ntype ReadWriter interface {\n\tReader\n\tWrite() error\n}\n")
	fe, err := ExtractFile("iembed.go", src)
	if err != nil {
		t.Fatal(err)
	}
	assertHasNode(t, fe.Nodes, "Reader", "interface")
	assertHasNode(t, fe.Nodes, "ReadWriter", "interface")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "ReadWriter", "Reader")
}

func TestExtractGoImport(t *testing.T) {
	src := []byte("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n")
	fe, err := ExtractFile("imp.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "fmt", "import")
	assertHasNode(t, fe.Nodes, "os", "import")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "imports", "imp.go", "fmt")
}

func TestExtractGoSameFileCall(t *testing.T) {
	src := []byte("package main\n\nfunc main() {\n\thelper()\n}\n\nfunc helper() string {\n\treturn \"ok\"\n}\n")
	fe, err := ExtractFile("call.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "helper()")
}

func TestExtractGoMethodCallViaSelector(t *testing.T) {
	src := []byte("package main\n\ntype Svc struct{ name string }\n\nfunc NewSvc() *Svc { return &Svc{} }\n\nfunc (s *Svc) Run() string { return s.name }\n\nfunc main() {\n\ts := NewSvc()\n\ts.Run()\n}\n")
	fe, err := ExtractFile("sel.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "NewSvc()")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Svc.Run()")
}

func TestExtractGoCrossFileCall(t *testing.T) {
	src := []byte("package main\n\nfunc caller() {\n\thelper()\n}\n\nfunc helper() {}\n")
	fe, err := ExtractFile("a.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "caller()", "helper()")
}

func TestExtractUnresolvedCrossFile(t *testing.T) {
	src := []byte("package main\n\nfunc caller() {\n\tOtherFunc()\n}\n")
	fe, err := ExtractFile("a.go", src)
	if err != nil {
		t.Fatal(err)
	}

	if len(fe.Unresolved) == 0 {
		t.Fatal("expected unresolved calls for cross-file reference")
	}
	found := false
	for _, u := range fe.Unresolved {
		if u.CalleeName == "OtherFunc" && !u.IsMethod {
			found = true
		}
	}
	if !found {
		t.Error("expected unresolved call to OtherFunc")
	}
}

func TestInferPackageName(t *testing.T) {
	tests := []struct {
		importPath string
		want       string
	}{
		{"fmt", "fmt"},
		{"os", "os"},
		{"net/http", "http"},
		{"github.com/redis/go-redis/v9", "redis"},
		{"github.com/user/repo/v2", "repo"},
		{"gopkg.in/yaml.v3", "yaml"},
		{"gopkg.in/check.v1", "check"},
		{"github.com/tree-sitter/go-tree-sitter", "tree_sitter"},
		{"github.com/spf13/cobra", "cobra"},
		{"v2", "v2"},
		{"github.com/go-playground/validator", "validator"},
		{"github.com/chi-middleware/proxy", "proxy"},
		{"github.com/getsentry/sentry-go", "sentry"},
		{"github.com/quic-go/quic-go", "quic"},
		{"github.com/example/foo-go", "foo"},
		// Go keyword as path tail — declared name unknowable from path
		{"github.com/json-iterator/go", ""},
		{"example.com/foo/type", ""},
		{"example.com/bar/func", ""},
	}
	for _, tt := range tests {
		got := inferPackageName(tt.importPath)
		if got != tt.want {
			t.Errorf("inferPackageName(%q) = %q, want %q", tt.importPath, got, tt.want)
		}
	}
}

func TestExtractGoImportVersionSuffix(t *testing.T) {
	src := []byte(`package main

import (
	"github.com/spf13/cobra/v2"
	"gopkg.in/yaml.v3"
)

type Cmd struct{}
func (c *Cmd) Execute() {}

type Marshal struct{}
func (m *Marshal) Marshal() {}

func main() {
	cobra.New()
	yaml.Marshal()
}
`)
	fe, err := ExtractFile("ver.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "main()" && tgtNode != nil {
				t.Errorf("unexpected calls edge from main() to %s — external package call should be suppressed", tgtNode.Label)
			}
		}
	}
}

func findNodeByID(nodes []*Node, id string) *Node {
	for _, n := range nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

func TestExtractGoDashedImportSuppression(t *testing.T) {
	src := []byte(`package main

import "github.com/redis/go-redis/v9"

type Client struct{}
func (c *Client) NewClient() {}

func main() {
	redis.NewClient()
}
`)
	fe, err := ExtractFile("dashed.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "main()" && tgtNode != nil {
				t.Errorf("unexpected calls edge from main() to %s — dashed import call should be suppressed", tgtNode.Label)
			}
		}
	}
}

func TestExtractGoSuffixDashedImportSuppression(t *testing.T) {
	src := []byte(`package main

import "github.com/getsentry/sentry-go"

type Hub struct{}
func (h *Hub) CaptureException() {}

func main() {
	sentry.CaptureException()
}
`)
	fe, err := ExtractFile("suffix.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "main()" && tgtNode != nil {
				t.Errorf("unexpected calls edge from main() to %s — sentry-go import call should be suppressed", tgtNode.Label)
			}
		}
	}
}

func TestExtractGoOpaqueImportSuppression(t *testing.T) {
	// Simulates github.com/json-iterator/go whose declared package name is
	// "jsoniter" but inferPackageName can't determine this from the path.
	// Without the opaque-import guard, jsoniter.Marshal() would fall through
	// to findMethodByName and create a bogus edge to Svc.Marshal().
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Marshal() []byte { return nil }

func main() {
	jsoniter.Marshal(nil)
}
`)
	fe, err := ExtractFile("opaque.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "main()" && tgtNode != nil {
				t.Errorf("unexpected calls edge from main() to %s — opaque import call should not create bogus edge", tgtNode.Label)
			}
		}
	}
}

func TestExtractGoOpaqueImportPreservesTypeCalls(t *testing.T) {
	// Even with opaque imports, method expressions on known types should
	// still resolve (e.g. Type.Method used as a method value).
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Run() {}

func caller() {
	Svc.Run(nil)
}
`)
	fe, err := ExtractFile("opaque_type.go", src)
	if err != nil {
		t.Fatal(err)
	}

	// Svc is a known type, so Svc.Run() should still resolve
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "caller()", "Svc.Run()")
}

func TestInferImportInfo(t *testing.T) {
	tests := []struct {
		importPath    string
		wantName      string
		wantUncertain bool
	}{
		{"fmt", "fmt", false},
		{"net/http", "http", false},
		{"github.com/spf13/cobra", "cobra", false},
		{"github.com/redis/go-redis/v9", "redis", false},
		{"github.com/getsentry/sentry-go", "sentry", false},
		{"github.com/example/foo-go", "foo", false},
		// Hyphens survive go- trim → uncertain
		{"github.com/tree-sitter/go-tree-sitter", "tree_sitter", true},
		{"github.com/cyphar/filepath-securejoin", "filepath_securejoin", true},
		{"github.com/hashicorp/go-immutable-radix", "immutable_radix", true},
		{"github.com/chi-middleware/proxy", "proxy", false},
		// Go keywords → empty + uncertain
		{"github.com/json-iterator/go", "", true},
		{"example.com/foo/type", "", true},
	}
	for _, tt := range tests {
		name, uncertain := inferImportInfo(tt.importPath)
		if name != tt.wantName || uncertain != tt.wantUncertain {
			t.Errorf("inferImportInfo(%q) = (%q, %v), want (%q, %v)",
				tt.importPath, name, uncertain, tt.wantName, tt.wantUncertain)
		}
	}
}

func TestExtractGoOpaqueImportPreservesVarCalls(t *testing.T) {
	// When a file has opaque imports, method calls on local variables
	// (identified by short_var_declaration) must still resolve locally.
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Run() {}

func main() {
	svc := &Svc{}
	svc.Run()
}
`)
	fe, err := ExtractFile("varfix.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Svc.Run()")
}

func TestExtractGoUncertainImportSuppression(t *testing.T) {
	// Imports like filepath-securejoin produce an uncertain inferred name.
	// The real qualifier (securejoin) should not create bogus edges.
	src := []byte(`package main

import "github.com/cyphar/filepath-securejoin"

type Svc struct{}
func (s *Svc) SecureJoin() string { return "" }

func main() {
	securejoin.SecureJoin("/", "foo")
}
`)
	fe, err := ExtractFile("uncertain.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "main()" && tgtNode != nil {
				t.Errorf("unexpected calls edge from main() to %s — uncertain import call should not create bogus edge", tgtNode.Label)
			}
		}
	}
}

func TestOpaqueImportNotResolvedCrossFile(t *testing.T) {
	// When the opaque guard fires, the call must NOT be added to unresolved,
	// so cross-file resolution cannot create a false edge by bare method name.
	dir := t.TempDir()
	fileA := []byte(`package main

import "github.com/json-iterator/go"

func caller() {
	jsoniter.Marshal(nil)
}
`)
	fileB := []byte(`package main

type Svc struct{}
func (s *Svc) Marshal() []byte { return nil }
`)
	if err := os.WriteFile(filepath.Join(dir, "a.go"), fileA, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), fileB, 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, dir)

	nodes, edges, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range edges {
		if e.Relation == "calls" {
			src := findNodeByID(nodes, e.Source)
			tgt := findNodeByID(nodes, e.Target)
			if src != nil && src.Label == "caller()" && tgt != nil && tgt.Label == "Svc.Marshal()" {
				t.Error("opaque import call jsoniter.Marshal() was incorrectly resolved to Svc.Marshal() at repo level")
			}
		}
	}
}

func TestCollectVarNamesRangeClause(t *testing.T) {
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Run() {}

func main() {
	items := []Svc{{}}
	for _, svc := range items {
		svc.Run()
	}
}
`)
	fe, err := ExtractFile("range.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Svc.Run()")
}

func TestVarNamesScopedPerFunction(t *testing.T) {
	// A variable "jsoniter" in funcA must NOT prevent the opaque guard from
	// firing for jsoniter.Marshal() in funcB (cross-function scope leakage).
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Marshal() []byte { return nil }

func funcA() {
	jsoniter := &Svc{}
	jsoniter.Marshal()
}

func funcB() {
	jsoniter.Marshal(nil)
}
`)
	fe, err := ExtractFile("scope.go", src)
	if err != nil {
		t.Fatal(err)
	}

	// funcA defines jsoniter locally — funcA→Svc.Marshal() should resolve
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "funcA()", "Svc.Marshal()")

	// funcB has no local jsoniter — opaque guard should suppress the edge
	for _, e := range fe.Edges {
		if e.Relation == "calls" {
			srcNode := findNodeByID(fe.Nodes, e.Source)
			tgtNode := findNodeByID(fe.Nodes, e.Target)
			if srcNode != nil && srcNode.Label == "funcB()" && tgtNode != nil && tgtNode.Label == "Svc.Marshal()" {
				t.Error("cross-function scope leakage: funcB→Svc.Marshal() should be suppressed by opaque guard")
			}
		}
	}
}

func TestCollectVarNamesTypeSwitchAlias(t *testing.T) {
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Run() {}

type Runner interface{ Run() }

func handle(r Runner) {
	switch svc := r.(type) {
	case *Svc:
		svc.Run()
	}
}
`)
	fe, err := ExtractFile("typeswitch.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "handle()", "Svc.Run()")
}

func TestNamedResultParamsRecognized(t *testing.T) {
	// Named return parameters must be recognized as local vars so the opaque
	// guard doesn't suppress method calls on them.
	src := []byte(`package main

import "github.com/json-iterator/go"

type Svc struct{}
func (s *Svc) Run() {}

func factory() (svc *Svc) {
	svc = &Svc{}
	svc.Run()
	return
}
`)
	fe, err := ExtractFile("named_result.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "factory()", "Svc.Run()")
}

func TestUncertainImportDoesNotBlockLocalVar(t *testing.T) {
	// An uncertain import guess (e.g. filepath_securejoin) must not
	// unconditionally suppress calls on a local variable with the same name.
	src := []byte(`package main

import "github.com/cyphar/filepath-securejoin"

type Joiner struct{}
func (j *Joiner) Join() string { return "" }

func main() {
	filepath_securejoin := &Joiner{}
	filepath_securejoin.Join()
}
`)
	fe, err := ExtractFile("uncertain_local.go", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Joiner.Join()")
}

func TestExtractUnsupportedExtension(t *testing.T) {
	_, err := ExtractFile("foo.rb", []byte("puts 'hi'"))
	if err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestWalkRepo(t *testing.T) {
	if _, err := os.Stat(filepath.Join("testdata", "goproj")); err != nil {
		t.Skip("testdata/goproj not found")
	}
	dir := prepareGoproj(t)

	nodes, edges, result, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Processed < 2 {
		t.Errorf("expected at least 2 processed files, got %d", result.Processed)
	}

	assertHasNode(t, nodes, "main()", "function")
	assertHasNode(t, nodes, "Service", "struct")
	assertHasNode(t, nodes, "NewService()", "function")
	assertHasNode(t, nodes, "Service.Run()", "method")
	assertHasEdgeByLabel(t, nodes, edges, "calls", "main()", "helper()")
	assertHasEdgeByLabel(t, nodes, edges, "calls", "main()", "NewService()")
	assertHasEdgeByLabel(t, nodes, edges, "calls", "main()", "Service.Run()")
	assertHasEdgeByLabel(t, nodes, edges, "method", "Service", "Service.Run()")
}

func TestGracefulDegradation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses file permission checks")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte("package main\nfunc good() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	badPath := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(badPath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, dir)
	// Make bad.go unreadable AFTER git has indexed it, so it still appears in
	// `git ls-files` but os.ReadFile fails — exercising the per-file error path.
	if err := os.Chmod(badPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(badPath, 0o644) })

	nodes, _, result, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	if result.Processed < 1 {
		t.Error("expected at least 1 processed file")
	}
	if result.Errors < 1 {
		t.Error("expected at least 1 error for unreadable file")
	}
	assertHasNode(t, nodes, "good()", "function")
}

func TestGracefulDegradation_BrokenSyntax(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte("package main\nfunc good() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Syntactically invalid Go — tree-sitter does partial parse, build must not abort.
	broken := []byte("package main\nfunc broken( {\n\treturn\n}\ntype Incomplete struct {\n")
	if err := os.WriteFile(filepath.Join(dir, "broken.go"), broken, 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, dir)

	nodes, _, result, err := WalkRepo(dir)
	if err != nil {
		t.Fatalf("WalkRepo should succeed with broken syntax: %v", err)
	}
	if result.Processed < 1 {
		t.Error("expected at least 1 processed file")
	}
	assertHasNode(t, nodes, "good()", "function")
}

func TestWalkRepo_RespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte("package main\nfunc good() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.go"), []byte("package main\nfunc ignored() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, dir)

	nodes, _, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	assertHasNode(t, nodes, "good()", "function")
	for _, n := range nodes {
		if n.Label == "ignored()" {
			t.Errorf("gitignored file ignored.go was walked: node %q present", n.Label)
		}
	}
}

func TestWalkRepo_FiltersIgnoreDirsByPathComponent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "good.go"), []byte("package main\nfunc good() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// vendor is in ignoreDirs — committed (not gitignored) files under it
	// must still be filtered out by the AND-mask in WalkRepo.
	vendor := filepath.Join(dir, "vendor", "pkg")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "vendored.go"), []byte("package pkg\nfunc Vendored() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitInitFixture(t, dir)

	nodes, _, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	assertHasNode(t, nodes, "good()", "function")
	for _, n := range nodes {
		if n.Label == "Vendored()" {
			t.Errorf("vendored file under vendor/ was walked: node %q present", n.Label)
		}
	}
}

func assertHasNode(t *testing.T, nodes []*Node, label, kind string) {
	t.Helper()
	for _, n := range nodes {
		if n.Label == label && n.Kind == kind {
			return
		}
	}
	var have []string
	for _, n := range nodes {
		have = append(have, n.Label+"("+n.Kind+")")
	}
	t.Errorf("node label=%q kind=%q not found; have: %s", label, kind, strings.Join(have, ", "))
}

func assertHasPendingImport(t *testing.T, imports []PendingImport, spec, lang, source string) {
	t.Helper()
	for _, pi := range imports {
		if pi.Spec == spec && pi.Language == lang && pi.SourceFile == source {
			return
		}
	}
	var have []string
	for _, pi := range imports {
		have = append(have, pi.Spec+"("+pi.Language+"@"+pi.SourceFile+")")
	}
	t.Errorf("pending import spec=%q lang=%q source=%q not found; have: %s", spec, lang, source, strings.Join(have, ", "))
}

func assertHasEdgeByLabel(t *testing.T, nodes []*Node, edges []Edge, relation, srcLabel, tgtLabel string) {
	t.Helper()
	idByLabel := make(map[string]string, len(nodes))
	for _, n := range nodes {
		idByLabel[n.Label] = n.ID
	}
	srcID := idByLabel[srcLabel]
	tgtID := idByLabel[tgtLabel]
	if srcID == "" {
		t.Errorf("edge check: source node %q not found", srcLabel)
		return
	}
	if tgtID == "" {
		t.Errorf("edge check: target node %q not found", tgtLabel)
		return
	}
	for _, e := range edges {
		if e.Relation == relation && e.Source == srcID && e.Target == tgtID {
			return
		}
	}
	var have []string
	for _, e := range edges {
		have = append(have, e.Source+" --"+e.Relation+"--> "+e.Target)
	}
	t.Errorf("edge %s(%s) --%s--> %s(%s) not found; have:\n  %s", srcLabel, srcID, relation, tgtLabel, tgtID, strings.Join(have, "\n  "))
}
