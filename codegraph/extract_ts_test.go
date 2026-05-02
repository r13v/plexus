package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTSFunc(t *testing.T) {
	src := []byte("function hello(): string { return 'hi'; }\n")
	fe, err := ExtractFile("main.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "hello()", "function")
	assertHasNode(t, fe.Nodes, "main.ts", "file")
}

func TestExtractTSClassAndMethod(t *testing.T) {
	src := []byte(`class Svc {
  run(): string { return "ok"; }
}
`)
	fe, err := ExtractFile("svc.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Svc", "class")
	assertHasNode(t, fe.Nodes, "Svc.run()", "method")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "method", "Svc", "Svc.run()")
}

func TestExtractTSInterface(t *testing.T) {
	src := []byte(`interface Runnable {
  run(): void;
}
`)
	fe, err := ExtractFile("iface.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Runnable", "interface")
}

func TestExtractTSTypeAlias(t *testing.T) {
	src := []byte("type ID = string;\n")
	fe, err := ExtractFile("alias.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "ID", "type")
}

func TestExtractTSAbstractClass(t *testing.T) {
	src := []byte(`abstract class Base {
  abstract run(): void;
}
`)
	fe, err := ExtractFile("abs.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Base", "class")
}

func TestExtractTSImport(t *testing.T) {
	src := []byte(`import { x } from './util';
import fs from 'fs';
`)
	fe, err := ExtractFile("imp.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasPendingImport(t, fe.Imports, "./util", "ts", "imp.ts")
	assertHasPendingImport(t, fe.Imports, "fs", "ts", "imp.ts")
}

func TestExtractTSSameFileCall(t *testing.T) {
	src := []byte(`function helper(): string { return "ok"; }
function main(): void { helper(); }
`)
	fe, err := ExtractFile("call.ts", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "helper()")
}

func TestExtractTSXComponent(t *testing.T) {
	src := []byte(`function Button(): JSX.Element {
  return <button>Click</button>;
}
`)
	fe, err := ExtractFile("btn.tsx", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Button()", "function")
}

func TestExtractTSWalkRepoCrossFile(t *testing.T) {
	dir := filepath.Join("testdata", "tsproj")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("testdata/tsproj not found")
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
