package codegraph

import (
	"path/filepath"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func extractPython(path string, src []byte) (*FileExtraction, error) {
	root, ctx, err := parseTS(path, src, grammars.PythonLanguage())
	if err != nil {
		return nil, err
	}

	e := &pyExtractor{
		tsCtx:   ctx,
		path:    path,
		nodeMap: make(map[string]*Node),
	}

	fileNode := &Node{
		ID:         MakeID(path),
		Label:      filepath.Base(path),
		Kind:       "file",
		SourceFile: path,
	}
	e.nodes = append(e.nodes, fileNode)
	e.nodeMap[fileNode.ID] = fileNode
	e.fileNodeID = fileNode.ID

	e.structurePass(root)
	e.callGraphPass(root)

	return &FileExtraction{
		Nodes:      e.nodes,
		Edges:      e.edges,
		Unresolved: e.unresolved,
		Imports:    e.imports,
	}, nil
}

type pyExtractor struct {
	tsCtx
	path       string
	nodes      []*Node
	edges      []Edge
	unresolved []UnresolvedCall
	imports    []PendingImport
	nodeMap    map[string]*Node
	fileNodeID string
}

func (e *pyExtractor) addNode(n *Node) {
	e.nodes = append(e.nodes, n)
	e.nodeMap[n.ID] = n
	e.edges = append(e.edges, Edge{
		Source:   e.fileNodeID,
		Target:   n.ID,
		Relation: "contains",
	})
}

// unwrapDecorated returns the inner function/class node wrapped by a
// decorated_definition, or n itself for plain definitions.
func (e *pyExtractor) unwrapDecorated(n *gts.Node) *gts.Node {
	if n == nil || e.kind(n) != "decorated_definition" {
		return n
	}
	if def := e.field(n, "definition"); def != nil {
		return def
	}
	for i := e.nchild(n) - 1; i >= 0; i-- {
		child := e.child(n, i)
		if child == nil {
			continue
		}
		if k := e.kind(child); k == "function_definition" || k == "class_definition" {
			return child
		}
	}
	return nil
}

func (e *pyExtractor) structurePass(root *gts.Node) {
	for i := range e.nchild(root) {
		child := e.child(root, i)
		if child == nil {
			continue
		}
		def := e.unwrapDecorated(child)
		if def == nil {
			continue
		}
		switch e.kind(def) {
		case "function_definition":
			e.extractFunc(def)
		case "class_definition":
			e.extractClass(def)
		case "import_statement":
			e.extractImport(def, false)
		case "import_from_statement":
			e.extractImport(def, true)
		}
	}
}

func (e *pyExtractor) extractFunc(n *gts.Node) {
	nameNode := e.field(n, "name")
	if nameNode == nil {
		return
	}
	name := e.text(nameNode)
	id := MakeID(e.path, name)
	e.addNode(&Node{
		ID:             id,
		Label:          name + "()",
		Kind:           "function",
		SourceFile:     e.path,
		SourceLocation: e.loc(n),
	})
}

func (e *pyExtractor) extractClass(n *gts.Node) {
	nameNode := e.field(n, "name")
	if nameNode == nil {
		return
	}
	className := e.text(nameNode)
	classID := MakeID(e.path, className)
	e.addNode(&Node{
		ID:             classID,
		Label:          className,
		Kind:           "class",
		SourceFile:     e.path,
		SourceLocation: e.loc(n),
	})

	if supers := e.field(n, "superclasses"); supers != nil {
		for i := range e.nchild(supers) {
			arg := e.child(supers, i)
			if arg == nil || e.kind(arg) != "identifier" {
				continue
			}
			baseName := e.text(arg)
			baseID := MakeID(e.path, baseName)
			if _, ok := e.nodeMap[baseID]; ok {
				e.edges = append(e.edges, Edge{
					Source:   classID,
					Target:   baseID,
					Relation: "inherits",
				})
			}
		}
	}

	body := e.field(n, "body")
	if body == nil {
		return
	}
	for i := range e.nchild(body) {
		child := e.child(body, i)
		if child == nil {
			continue
		}
		def := e.unwrapDecorated(child)
		if def == nil || e.kind(def) != "function_definition" {
			continue
		}
		mNameNode := e.field(def, "name")
		if mNameNode == nil {
			continue
		}
		mName := e.text(mNameNode)
		methodID := MakeID(e.path, className, mName)
		e.addNode(&Node{
			ID:             methodID,
			Label:          className + "." + mName + "()",
			Kind:           "method",
			SourceFile:     e.path,
			SourceLocation: e.loc(def),
		})
		e.edges = append(e.edges, Edge{
			Source:   classID,
			Target:   methodID,
			Relation: "method",
		})
	}
}

// extractImport records "import foo" and "from foo import bar" statements as
// PendingImports. Relative imports ("from . import util", "from .foo import x")
// are recorded too — the post-walk resolver turns them into edges to the
// corresponding file nodes when they exist.
//
// For "from . import a, b" (module is pure dots), we emit one PendingImport
// per imported name so each submodule reference is tracked independently.
func (e *pyExtractor) extractImport(n *gts.Node, isFrom bool) {
	if isFrom {
		modNode := e.field(n, "module_name")
		if modNode == nil {
			return
		}
		modText := strings.TrimSpace(e.text(modNode))
		if modText == "" {
			return
		}
		if isPureDots(modText) {
			for _, name := range e.fromImportedNames(n) {
				e.addImport(modText + name)
			}
			return
		}
		e.addImport(modText)
		return
	}

	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child == nil {
			continue
		}
		switch e.kind(child) {
		case "dotted_name":
			e.addImport(e.text(child))
		case "aliased_import":
			if p := e.field(child, "name"); p != nil {
				e.addImport(e.text(p))
			}
		}
	}
}

func (e *pyExtractor) addImport(spec string) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return
	}
	e.imports = append(e.imports, PendingImport{
		Spec:       spec,
		Language:   "python",
		SourceFile: e.path,
	})
}

// fromImportedNames returns the names brought into scope by a
// "from X import a, b, c" statement (excluding aliases; the imported symbol
// is what names a submodule, not the local rebinding). The module_name child
// carries the "module_name" field; we iterate all children so we can skip that
// entry by field name — gotreesitter indexes fields by plain child position.
func (e *pyExtractor) fromImportedNames(n *gts.Node) []string {
	var names []string
	for i := range n.ChildCount() {
		child := n.Child(i)
		if child == nil || !child.IsNamed() {
			continue
		}
		if n.FieldNameForChild(i, e.lang) == "module_name" {
			continue
		}
		switch e.kind(child) {
		case "dotted_name":
			names = append(names, e.text(child))
		case "aliased_import":
			if p := e.field(child, "name"); p != nil {
				names = append(names, e.text(p))
			}
		}
	}
	return names
}

func isPureDots(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		if s[i] != '.' {
			return false
		}
	}
	return true
}

func (e *pyExtractor) callGraphPass(root *gts.Node) {
	e.walkCalls(root, "")
}

func (e *pyExtractor) walkCalls(n *gts.Node, enclosing string) {
	if n == nil {
		return
	}
	switch e.kind(n) {
	case "function_definition":
		if nameNode := e.field(n, "name"); nameNode != nil {
			enclosing = MakeID(e.path, e.text(nameNode))
		}
	case "class_definition":
		e.walkClassCalls(n)
		return
	case "call":
		if enclosing != "" {
			e.resolveCall(n, enclosing)
		}
	}
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child != nil {
			e.walkCalls(child, enclosing)
		}
	}
}

// walkClassCalls dispatches call-pass traversal for each method in a class body
// with the method's own enclosing ID. Calls outside method bodies (e.g. class
// decorators, default-value expressions) are not attributed.
func (e *pyExtractor) walkClassCalls(n *gts.Node) {
	classNode := e.field(n, "name")
	if classNode == nil {
		return
	}
	className := e.text(classNode)
	body := e.field(n, "body")
	if body == nil {
		return
	}
	for i := range e.nchild(body) {
		child := e.child(body, i)
		if child == nil {
			continue
		}
		def := e.unwrapDecorated(child)
		if def == nil || e.kind(def) != "function_definition" {
			continue
		}
		mName := e.field(def, "name")
		if mName == nil {
			continue
		}
		methodEnclosing := MakeID(e.path, className, e.text(mName))
		if dBody := e.field(def, "body"); dBody != nil {
			e.walkCalls(dBody, methodEnclosing)
		}
	}
}

func (e *pyExtractor) resolveCall(n *gts.Node, callerID string) {
	funcNode := e.field(n, "function")
	if funcNode == nil {
		return
	}
	switch e.kind(funcNode) {
	case "identifier":
		name := e.text(funcNode)
		calleeID := MakeID(e.path, name)
		if _, ok := e.nodeMap[calleeID]; ok {
			if callerID != calleeID {
				e.edges = append(e.edges, Edge{
					Source:         callerID,
					Target:         calleeID,
					Relation:       "calls",
					SourceFile:     e.path,
					SourceLocation: e.loc(n),
				})
			}
		} else {
			e.unresolved = append(e.unresolved, UnresolvedCall{
				CallerID:       callerID,
				CalleeName:     name,
				SourceFile:     e.path,
				SourceLocation: e.loc(n),
			})
		}
	case "attribute":
		attrNode := e.field(funcNode, "attribute")
		if attrNode == nil {
			return
		}
		methodName := e.text(attrNode)
		calleeID := findMethodByName(e.nodeMap, methodName)
		if calleeID != "" && callerID != calleeID {
			e.edges = append(e.edges, Edge{
				Source:         callerID,
				Target:         calleeID,
				Relation:       "calls",
				SourceFile:     e.path,
				SourceLocation: e.loc(n),
			})
		} else if calleeID == "" {
			e.unresolved = append(e.unresolved, UnresolvedCall{
				CallerID:       callerID,
				CalleeName:     methodName,
				IsMethod:       true,
				SourceFile:     e.path,
				SourceLocation: e.loc(n),
			})
		}
	}
}
