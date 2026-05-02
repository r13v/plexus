package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractJSFunc(t *testing.T) {
	src := []byte("function hello() { return 'hi'; }\n")
	fe, err := ExtractFile("main.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "hello()", "function")
	assertHasNode(t, fe.Nodes, "main.js", "file")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "contains", "main.js", "hello()")
}

func TestExtractJSClassAndMethod(t *testing.T) {
	src := []byte("class Svc { run() { return 'ok'; } }\n")
	fe, err := ExtractFile("svc.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Svc", "class")
	assertHasNode(t, fe.Nodes, "Svc.run()", "method")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "method", "Svc", "Svc.run()")
}

func TestExtractJSClassInheritance(t *testing.T) {
	src := []byte(`class Base {}
class Child extends Base {}
`)
	fe, err := ExtractFile("inh.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Base", "class")
	assertHasNode(t, fe.Nodes, "Child", "class")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "Child", "Base")
}

func TestExtractJSImport(t *testing.T) {
	src := []byte(`import fs from 'fs';
import { x } from './util.js';
`)
	fe, err := ExtractFile("imp.js", src)
	if err != nil {
		t.Fatal(err)
	}

	// Import nodes/edges are produced by the post-walk resolver, not the
	// per-file extractor; per-file output records PendingImports instead.
	assertHasPendingImport(t, fe.Imports, "fs", "js", "imp.js")
	assertHasPendingImport(t, fe.Imports, "./util.js", "js", "imp.js")
}

func TestExtractJSSameFileCall(t *testing.T) {
	src := []byte(`function helper() { return 'ok'; }
function main() { helper(); }
`)
	fe, err := ExtractFile("call.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "helper()")
}

func TestExtractJSMethodCallViaMember(t *testing.T) {
	src := []byte(`class Svc { run() {} }
function main() {
  const s = new Svc();
  s.run();
}
`)
	fe, err := ExtractFile("member.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Svc.run()")
}

func TestExtractJSExportedDecl(t *testing.T) {
	src := []byte(`export function exported() {}
export class Exp {}
`)
	fe, err := ExtractFile("exp.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "exported()", "function")
	assertHasNode(t, fe.Nodes, "Exp", "class")
}

func TestExtractJSQualifiedExtends(t *testing.T) {
	src := []byte(`class Mix {}
class Child extends ns.Mix {}
`)
	fe, err := ExtractFile("qe.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "Child", "Mix")
}

func TestExtractJSReExportAsImport(t *testing.T) {
	src := []byte(`export * from './a';
export { foo } from './b';
`)
	fe, err := ExtractFile("re.js", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasPendingImport(t, fe.Imports, "./a", "js", "re.js")
	assertHasPendingImport(t, fe.Imports, "./b", "js", "re.js")
}

func TestExtractJSWalkRepoCrossFile(t *testing.T) {
	dir := filepath.Join("testdata", "jsproj")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("testdata/jsproj not found")
	}

	nodes, edges, result, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Processed < 2 {
		t.Errorf("expected at least 2 processed files, got %d", result.Processed)
	}

	assertHasNode(t, nodes, "main()", "function")
	assertHasNode(t, nodes, "Service", "class")
	assertHasNode(t, nodes, "Service.run()", "method")
	assertHasEdgeByLabel(t, nodes, edges, "calls", "main()", "helper()")
	assertHasEdgeByLabel(t, nodes, edges, "calls", "main()", "Service.run()")
}
