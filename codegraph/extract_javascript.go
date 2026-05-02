// File is named extract_javascript.go (not extract_js.go) because Go treats
// _js.go as a GOOS build-tag suffix and would exclude the file from all
// non-"js" builds. The same trap applies to any future _<GOOS>.go filename.

package codegraph

import (
	"path/filepath"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func extractJS(path string, src []byte) (*FileExtraction, error) {
	return extractJSLike(path, src, grammars.JavascriptLanguage(), false)
}

// extractJSLike is the shared extraction body for JavaScript and TypeScript —
// their ASTs overlap substantially; TS adds a few extra structural kinds that
// are gated by the isTS flag.
func extractJSLike(path string, src []byte, lang *gts.Language, isTS bool) (*FileExtraction, error) {
	root, ctx, err := parseTS(path, src, lang)
	if err != nil {
		return nil, err
	}

	e := &jsExtractor{
		tsCtx:   ctx,
		path:    path,
		nodeMap: make(map[string]*Node),
		isTS:    isTS,
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

type jsExtractor struct {
	tsCtx
	path       string
	nodes      []*Node
	edges      []Edge
	unresolved []UnresolvedCall
	imports    []PendingImport
	nodeMap    map[string]*Node
	fileNodeID string
	isTS       bool
}

func (e *jsExtractor) lang() string {
	if e.isTS {
		return "ts"
	}
	return "js"
}

// addNamed emits a contained Node keyed by the declaration's "name" field.
// Used for TS interface and type-alias declarations.
func (e *jsExtractor) addNamed(n *gts.Node, kind string) {
	nameNode := e.field(n, "name")
	if nameNode == nil {
		return
	}
	name := e.text(nameNode)
	e.addNode(&Node{
		ID:             MakeID(e.path, name),
		Label:          name,
		Kind:           kind,
		SourceFile:     e.path,
		SourceLocation: e.loc(n),
	})
}

// superBaseName returns the trailing identifier in a class-heritage super
// expression, or "" if the shape isn't one we understand.
func (e *jsExtractor) superBaseName(super *gts.Node) string {
	if super == nil {
		return ""
	}
	switch e.kind(super) {
	case "identifier":
		return e.text(super)
	case "member_expression":
		if prop := e.field(super, "property"); prop != nil {
			return e.text(prop)
		}
	}
	return ""
}

func (e *jsExtractor) addNode(n *Node) {
	e.nodes = append(e.nodes, n)
	e.nodeMap[n.ID] = n
	e.edges = append(e.edges, Edge{
		Source:   e.fileNodeID,
		Target:   n.ID,
		Relation: "contains",
	})
}

// unwrapExport returns the inner declaration of an export_statement, or n itself.
func (e *jsExtractor) unwrapExport(n *gts.Node) *gts.Node {
	if n == nil {
		return nil
	}
	if e.kind(n) != "export_statement" {
		return n
	}
	if d := e.field(n, "declaration"); d != nil {
		return d
	}
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child == nil {
			continue
		}
		switch e.kind(child) {
		case "function_declaration", "class_declaration",
			"interface_declaration", "type_alias_declaration",
			"abstract_class_declaration":
			return child
		}
	}
	return nil
}

func (e *jsExtractor) structurePass(root *gts.Node) {
	for i := range e.nchild(root) {
		child := e.child(root, i)
		if child == nil {
			continue
		}
		def := e.unwrapExport(child)
		if def == nil {
			// Re-export forms carry a "source" field on the export_statement
			// itself (export * from 'x'; export { y } from 'x'). We model the
			// external module the same as a direct import.
			if e.kind(child) == "export_statement" && e.field(child, "source") != nil {
				e.extractImport(child)
			}
			continue
		}
		switch e.kind(def) {
		case "function_declaration", "generator_function_declaration":
			e.extractFunc(def)
		case "class_declaration", "abstract_class_declaration":
			e.extractClass(def)
		case "import_statement":
			e.extractImport(def)
		case "interface_declaration":
			if e.isTS {
				e.addNamed(def, "interface")
			}
		case "type_alias_declaration":
			if e.isTS {
				e.addNamed(def, "type")
			}
		}
	}
}

func (e *jsExtractor) extractFunc(n *gts.Node) {
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

func (e *jsExtractor) extractClass(n *gts.Node) {
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

	// Heritage: "class Foo extends Bar" or "class Foo extends NS.Base" — the
	// super expression lives under a class_heritage child. We take the trailing
	// identifier in the qualified form (same strategy as Python attributes) so
	// that same-file base classes resolve even when the user wrote a qualified
	// reference like `extends ns.Base`.
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child == nil || e.kind(child) != "class_heritage" {
			continue
		}
		for j := range e.nchild(child) {
			super := e.child(child, j)
			baseName := e.superBaseName(super)
			if baseName == "" {
				continue
			}
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
		member := e.child(body, i)
		if member == nil {
			continue
		}
		if e.kind(member) != "method_definition" {
			continue
		}
		mNameNode := e.field(member, "name")
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
			SourceLocation: e.loc(member),
		})
		e.edges = append(e.edges, Edge{
			Source:   classID,
			Target:   methodID,
			Relation: "method",
		})
	}
}

// extractImport records the module specifier from an ECMAScript import_statement
// as a PendingImport. The actual graph node/edge is created in WalkRepo's
// post-walk resolver so relative specs ("./messages") can resolve to the real
// file node rather than producing a distinct node per caller directory.
func (e *jsExtractor) extractImport(n *gts.Node) {
	srcNode := e.field(n, "source")
	if srcNode == nil {
		return
	}
	raw := e.text(srcNode)
	mod := strings.Trim(raw, "\"'`")
	mod = strings.TrimSpace(mod)
	if mod == "" {
		return
	}
	e.imports = append(e.imports, PendingImport{
		Spec:       mod,
		Language:   e.lang(),
		SourceFile: e.path,
	})
}

func (e *jsExtractor) callGraphPass(root *gts.Node) {
	e.walkCalls(root, "")
}

func (e *jsExtractor) walkCalls(n *gts.Node, enclosing string) {
	if n == nil {
		return
	}
	switch e.kind(n) {
	case "function_declaration", "generator_function_declaration":
		if nameNode := e.field(n, "name"); nameNode != nil {
			enclosing = MakeID(e.path, e.text(nameNode))
		}
	case "class_declaration", "abstract_class_declaration":
		e.walkClassCalls(n)
		return
	case "call_expression":
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

func (e *jsExtractor) walkClassCalls(n *gts.Node) {
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
		member := e.child(body, i)
		if member == nil || e.kind(member) != "method_definition" {
			continue
		}
		mName := e.field(member, "name")
		if mName == nil {
			continue
		}
		methodEnclosing := MakeID(e.path, className, e.text(mName))
		if mBody := e.field(member, "body"); mBody != nil {
			e.walkCalls(mBody, methodEnclosing)
		}
	}
}

func (e *jsExtractor) resolveCall(n *gts.Node, callerID string) {
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
	case "member_expression":
		propNode := e.field(funcNode, "property")
		if propNode == nil {
			return
		}
		methodName := e.text(propNode)
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
