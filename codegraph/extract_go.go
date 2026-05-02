package codegraph

import (
	"path/filepath"
	"regexp"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

var (
	versionSuffix = regexp.MustCompile(`^v\d+$`)
	gopkgVersion  = regexp.MustCompile(`\.v\d+$`)

	goKeywords = map[string]bool{
		"break": true, "case": true, "chan": true, "const": true,
		"continue": true, "default": true, "defer": true, "else": true,
		"fallthrough": true, "for": true, "func": true, "go": true,
		"goto": true, "if": true, "import": true, "interface": true,
		"map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true,
		"var": true,
	}
)

// inferImportInfo returns the best-guess package name for an import path and
// whether that guess is uncertain. The name is "" when the path tail is a Go
// keyword (e.g. github.com/json-iterator/go). The uncertain flag is true when
// the path contains hyphens that survive go- prefix/suffix trimming — the
// declared name often differs from the path in those cases (e.g.
// github.com/cyphar/filepath-securejoin declares "securejoin", not
// "filepath_securejoin").
func inferImportInfo(importPath string) (name string, uncertain bool) {
	parts := strings.Split(importPath, "/")
	last := parts[len(parts)-1]

	if versionSuffix.MatchString(last) && len(parts) >= 2 {
		last = parts[len(parts)-2]
	}

	last = gopkgVersion.ReplaceAllString(last, "")

	if strings.Contains(last, "-") {
		last = strings.TrimPrefix(last, "go-")
		last = strings.TrimSuffix(last, "-go")
		uncertain = strings.Contains(last, "-")
		last = strings.ReplaceAll(last, "-", "_")
	}

	if goKeywords[last] {
		return "", true
	}

	return last, uncertain
}

func inferPackageName(importPath string) string {
	name, _ := inferImportInfo(importPath)
	return name
}

func extractGo(path string, src []byte) (*FileExtraction, error) {
	root, ctx, err := parseTS(path, src, grammars.GoLanguage())
	if err != nil {
		return nil, err
	}

	e := &goExtractor{
		tsCtx:       ctx,
		path:        path,
		nodeMap:     make(map[string]*Node),
		importNames: make(map[string]bool),
		typeNames:   make(map[string]bool),
		varNames:    make(map[string]bool),
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
	e.collectPackageVarNames(root)
	e.callGraphPass(root)

	return &FileExtraction{
		Nodes:      e.nodes,
		Edges:      e.edges,
		Unresolved: e.unresolved,
	}, nil
}

type goExtractor struct {
	tsCtx
	path             string
	nodes            []*Node
	edges            []Edge
	unresolved       []UnresolvedCall
	nodeMap          map[string]*Node
	fileNodeID       string
	importNames      map[string]bool
	typeNames        map[string]bool
	varNames         map[string]bool
	hasOpaqueImports bool
}

func (e *goExtractor) addNode(n *Node) {
	e.nodes = append(e.nodes, n)
	e.nodeMap[n.ID] = n
	e.edges = append(e.edges, Edge{
		Source:   e.fileNodeID,
		Target:   n.ID,
		Relation: "contains",
	})
}

func (e *goExtractor) structurePass(root *gts.Node) {
	for i := range e.nchild(root) {
		child := e.child(root, i)
		if child == nil {
			continue
		}
		switch e.kind(child) {
		case "function_declaration":
			e.extractFunc(child)
		case "method_declaration":
			e.extractMethod(child)
		case "type_declaration":
			e.extractTypeDecl(child)
		case "import_declaration":
			e.extractImport(child)
		}
	}
}

func (e *goExtractor) extractFunc(n *gts.Node) {
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

func (e *goExtractor) extractMethod(n *gts.Node) {
	nameNode := e.field(n, "name")
	if nameNode == nil {
		return
	}
	name := e.text(nameNode)

	receiverType := e.resolveReceiver(n)
	if receiverType == "" {
		return
	}

	methodID := MakeID(e.path, receiverType, name)
	e.addNode(&Node{
		ID:             methodID,
		Label:          receiverType + "." + name + "()",
		Kind:           "method",
		SourceFile:     e.path,
		SourceLocation: e.loc(n),
	})

	typeID := MakeID(e.path, receiverType)
	if _, ok := e.nodeMap[typeID]; ok {
		e.edges = append(e.edges, Edge{
			Source:   typeID,
			Target:   methodID,
			Relation: "method",
		})
	}
}

func (e *goExtractor) resolveReceiver(n *gts.Node) string {
	recvField := e.field(n, "receiver")
	if recvField == nil {
		return ""
	}
	// receiver is a parameter_list, get first parameter_declaration
	for i := range e.nchild(recvField) {
		param := e.child(recvField, i)
		if param == nil || e.kind(param) != "parameter_declaration" {
			continue
		}
		typeNode := e.field(param, "type")
		if typeNode == nil {
			continue
		}
		return stripPointer(e.text(typeNode))
	}
	return ""
}

func stripPointer(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "*")
	return s
}

func (e *goExtractor) extractTypeDecl(n *gts.Node) {
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child == nil {
			continue
		}
		switch e.kind(child) {
		case "type_spec":
			e.extractTypeSpec(child)
		case "type_alias":
			e.extractTypeSpec(child)
		}
	}
}

func (e *goExtractor) extractTypeSpec(n *gts.Node) {
	nameNode := e.field(n, "name")
	if nameNode == nil {
		return
	}
	name := e.text(nameNode)
	kind := "type"

	typeNode := e.field(n, "type")
	if typeNode != nil {
		switch e.kind(typeNode) {
		case "interface_type":
			kind = "interface"
		case "struct_type":
			kind = "struct"
		}
	}

	e.typeNames[name] = true

	id := MakeID(e.path, name)
	e.addNode(&Node{
		ID:             id,
		Label:          name,
		Kind:           kind,
		SourceFile:     e.path,
		SourceLocation: e.loc(n),
	})

	if typeNode != nil {
		e.extractEmbeddings(id, typeNode)
	}
}

// extractEmbeddings finds embedded types in struct/interface bodies and emits
// "inherits" edges to same-file types.
func (e *goExtractor) extractEmbeddings(hostID string, typeNode *gts.Node) {
	switch e.kind(typeNode) {
	case "struct_type":
		e.extractStructEmbeddings(hostID, typeNode)
	case "interface_type":
		e.extractInterfaceEmbeddings(hostID, typeNode)
	}
}

func (e *goExtractor) extractStructEmbeddings(hostID string, structNode *gts.Node) {
	for i := range e.nchild(structNode) {
		fieldList := e.child(structNode, i)
		if fieldList == nil || e.kind(fieldList) != "field_declaration_list" {
			continue
		}
		for j := range e.nchild(fieldList) {
			field := e.child(fieldList, j)
			if field == nil || e.kind(field) != "field_declaration" {
				continue
			}
			if e.field(field, "name") != nil {
				continue
			}
			typeName := e.embeddedTypeName(e.field(field, "type"))
			if typeName == "" {
				continue
			}
			targetID := MakeID(e.path, typeName)
			if _, ok := e.nodeMap[targetID]; ok {
				e.edges = append(e.edges, Edge{
					Source:   hostID,
					Target:   targetID,
					Relation: "inherits",
				})
			}
		}
	}
}

func (e *goExtractor) extractInterfaceEmbeddings(hostID string, ifaceNode *gts.Node) {
	for i := range e.nchild(ifaceNode) {
		child := e.child(ifaceNode, i)
		if child == nil || e.kind(child) != "type_elem" {
			continue
		}
		for j := range e.nchild(child) {
			typeChild := e.child(child, j)
			if typeChild == nil || e.kind(typeChild) != "type_identifier" {
				continue
			}
			typeName := e.text(typeChild)
			targetID := MakeID(e.path, typeName)
			if _, ok := e.nodeMap[targetID]; ok {
				e.edges = append(e.edges, Edge{
					Source:   hostID,
					Target:   targetID,
					Relation: "inherits",
				})
			}
		}
	}
}

// embeddedTypeName extracts the local type name from an embedded field's type node.
// Returns "" for cross-package or unrecognized types.
func (e *goExtractor) embeddedTypeName(n *gts.Node) string {
	if n == nil {
		return ""
	}
	switch e.kind(n) {
	case "type_identifier":
		return e.text(n)
	case "pointer_type":
		for i := range e.nchild(n) {
			child := e.child(n, i)
			if child != nil && e.kind(child) == "type_identifier" {
				return e.text(child)
			}
		}
	}
	return ""
}

func (e *goExtractor) extractImport(n *gts.Node) {
	var specs []*gts.Node
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child == nil {
			continue
		}
		switch e.kind(child) {
		case "import_spec":
			specs = append(specs, child)
		case "import_spec_list":
			for j := range e.nchild(child) {
				spec := e.child(child, j)
				if spec != nil && e.kind(spec) == "import_spec" {
					specs = append(specs, spec)
				}
			}
		}
	}

	fileID := MakeID(e.path)
	for _, spec := range specs {
		pathNode := e.field(spec, "path")
		if pathNode == nil {
			continue
		}
		importPath := strings.Trim(e.text(pathNode), `"`)

		aliasNode := e.field(spec, "name")
		if aliasNode != nil {
			alias := e.text(aliasNode)
			if alias != "." && alias != "_" {
				e.importNames[alias] = true
			}
		} else {
			name, uncertain := inferImportInfo(importPath)
			if name != "" && !uncertain {
				e.importNames[name] = true
			}
			if uncertain || name == "" {
				e.hasOpaqueImports = true
			}
		}
		importID := MakeID("import", importPath)
		if _, exists := e.nodeMap[importID]; !exists {
			node := &Node{
				ID:         importID,
				Label:      importPath,
				Kind:       "import",
				SourceFile: e.path,
			}
			e.nodes = append(e.nodes, node)
			e.nodeMap[importID] = node
		}
		e.edges = append(e.edges, Edge{
			Source:         fileID,
			Target:         importID,
			Relation:       "imports",
			SourceFile:     e.path,
			SourceLocation: e.loc(pathNode),
		})
	}
}

func (e *goExtractor) collectPackageVarNames(root *gts.Node) {
	for i := range e.nchild(root) {
		child := e.child(root, i)
		if child != nil && e.kind(child) == "var_declaration" {
			e.collectVarNamesTo(child, e.varNames)
		}
	}
}

func (e *goExtractor) collectFuncLocalVars(funcNode *gts.Node) map[string]bool {
	localVars := make(map[string]bool)
	for _, name := range []string{"parameters", "receiver", "result", "body"} {
		if n := e.field(funcNode, name); n != nil {
			e.collectVarNamesTo(n, localVars)
		}
	}
	return localVars
}

func (e *goExtractor) collectVarNamesTo(n *gts.Node, dest map[string]bool) {
	if n == nil {
		return
	}
	switch e.kind(n) {
	case "short_var_declaration":
		left := e.field(n, "left")
		e.collectIdentifiersTo(left, dest)
	case "var_spec", "parameter_declaration":
		for i := range e.nchild(n) {
			child := e.child(n, i)
			if child == nil {
				continue
			}
			if e.kind(child) == "identifier" {
				dest[e.text(child)] = true
			} else {
				break
			}
		}
	case "range_clause", "receive_statement":
		left := e.field(n, "left")
		e.collectIdentifiersTo(left, dest)
	case "type_switch_statement":
		if alias := e.field(n, "alias"); alias != nil {
			e.collectIdentifiersTo(alias, dest)
		}
	}
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child != nil {
			e.collectVarNamesTo(child, dest)
		}
	}
}

func (e *goExtractor) collectIdentifiersTo(n *gts.Node, dest map[string]bool) {
	if n == nil {
		return
	}
	if e.kind(n) == "identifier" {
		dest[e.text(n)] = true
		return
	}
	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child != nil && e.kind(child) == "identifier" {
			dest[e.text(child)] = true
		}
	}
}

func (e *goExtractor) callGraphPass(root *gts.Node) {
	e.walkCalls(root, "", nil)
}

func (e *goExtractor) walkCalls(n *gts.Node, enclosingFunc string, localVars map[string]bool) {
	if n == nil {
		return
	}

	switch e.kind(n) {
	case "function_declaration":
		nameNode := e.field(n, "name")
		if nameNode != nil {
			enclosingFunc = MakeID(e.path, e.text(nameNode))
		}
		localVars = e.collectFuncLocalVars(n)
	case "method_declaration":
		nameNode := e.field(n, "name")
		receiverType := e.resolveReceiver(n)
		if nameNode != nil && receiverType != "" {
			enclosingFunc = MakeID(e.path, receiverType, e.text(nameNode))
		}
		localVars = e.collectFuncLocalVars(n)
	case "call_expression":
		if enclosingFunc != "" {
			e.resolveCall(n, enclosingFunc, localVars)
		}
	}

	for i := range e.nchild(n) {
		child := e.child(n, i)
		if child != nil {
			e.walkCalls(child, enclosingFunc, localVars)
		}
	}
}

func (e *goExtractor) resolveCall(n *gts.Node, callerID string, localVars map[string]bool) {
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
	case "selector_expression":
		operandNode := e.field(funcNode, "operand")
		opaqueGuard := false
		if operandNode != nil && e.kind(operandNode) == "identifier" {
			operandName := e.text(operandNode)
			if e.importNames[operandName] {
				return
			}
			// When we have imports whose qualifier can't be inferred from the
			// path (e.g. github.com/json-iterator/go → "jsoniter"), an unknown
			// operand that isn't a locally-defined type is likely one of those
			// opaque import qualifiers — skip per-file method resolution and
			// defer to cross-file resolution to avoid bogus edges.
			if e.hasOpaqueImports && !e.typeNames[operandName] && !e.varNames[operandName] && !localVars[operandName] {
				opaqueGuard = true
			}
		}
		fieldNode := e.field(funcNode, "field")
		if fieldNode == nil {
			return
		}
		methodName := e.text(fieldNode)
		if opaqueGuard {
			return
		}
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
