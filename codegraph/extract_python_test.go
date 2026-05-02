package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractPythonFunc(t *testing.T) {
	src := []byte("def hello():\n    return 'hi'\n")
	fe, err := ExtractFile("main.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "hello()", "function")
	assertHasNode(t, fe.Nodes, "main.py", "file")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "contains", "main.py", "hello()")
}

func TestExtractPythonClassAndMethod(t *testing.T) {
	src := []byte(`class Svc:
    def run(self):
        return "running"
`)
	fe, err := ExtractFile("svc.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Svc", "class")
	assertHasNode(t, fe.Nodes, "Svc.run()", "method")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "method", "Svc", "Svc.run()")
}

func TestExtractPythonClassInheritance(t *testing.T) {
	src := []byte(`class Base:
    pass

class Child(Base):
    pass
`)
	fe, err := ExtractFile("inh.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "Base", "class")
	assertHasNode(t, fe.Nodes, "Child", "class")
	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "inherits", "Child", "Base")
}

func TestExtractPythonImport(t *testing.T) {
	src := []byte("import os\nimport json\nfrom typing import List\n")
	fe, err := ExtractFile("imp.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasPendingImport(t, fe.Imports, "os", "python", "imp.py")
	assertHasPendingImport(t, fe.Imports, "json", "python", "imp.py")
	assertHasPendingImport(t, fe.Imports, "typing", "python", "imp.py")
}

func TestExtractPythonSameFileCall(t *testing.T) {
	src := []byte(`def helper():
    return "ok"

def main():
    helper()
`)
	fe, err := ExtractFile("call.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "helper()")
}

func TestExtractPythonMethodCallViaAttribute(t *testing.T) {
	src := []byte(`class Svc:
    def run(self):
        return "ok"

def main():
    s = Svc()
    s.run()
`)
	fe, err := ExtractFile("attr.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasEdgeByLabel(t, fe.Nodes, fe.Edges, "calls", "main()", "Svc.run()")
}

func TestExtractPythonDecoratedDefinition(t *testing.T) {
	src := []byte(`def guard(f):
    return f

@guard
def target():
    pass

class Svc:
    @guard
    def run(self):
        pass
`)
	fe, err := ExtractFile("dec.py", src)
	if err != nil {
		t.Fatal(err)
	}

	assertHasNode(t, fe.Nodes, "target()", "function")
	assertHasNode(t, fe.Nodes, "Svc.run()", "method")
}

func TestExtractPythonRelativeImportsAsPending(t *testing.T) {
	src := []byte(`from . import util
from .sub import other
from os import path
`)
	fe, err := ExtractFile("rel.py", src)
	if err != nil {
		t.Fatal(err)
	}

	// "from . import util" → one PendingImport per imported name so that
	// each submodule reference is independently resolvable.
	assertHasPendingImport(t, fe.Imports, ".util", "python", "rel.py")
	assertHasPendingImport(t, fe.Imports, ".sub", "python", "rel.py")
	assertHasPendingImport(t, fe.Imports, "os", "python", "rel.py")
}

func TestExtractPythonWalkRepoCrossFile(t *testing.T) {
	dir := filepath.Join("testdata", "pyproj")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("testdata/pyproj not found")
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
