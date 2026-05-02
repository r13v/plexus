package codegraph

import (
	"fmt"
	"path/filepath"
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// tsCtx bundles the pure-Go tree-sitter language, source bytes, and the
// convenience methods each extractor needs. Embedding it into extractor
// structs keeps per-call boilerplate (Lang arg on ChildByFieldName/Type,
// src arg on Text) out of the walk code.
type tsCtx struct {
	src  []byte
	lang *gts.Language
}

func (c *tsCtx) text(n *gts.Node) string { return n.Text(c.src) }
func (c *tsCtx) kind(n *gts.Node) string { return n.Type(c.lang) }
func (c *tsCtx) nchild(n *gts.Node) int  { return n.NamedChildCount() }
func (c *tsCtx) child(n *gts.Node, i int) *gts.Node {
	return n.NamedChild(i)
}

func (c *tsCtx) field(n *gts.Node, name string) *gts.Node {
	if n == nil {
		return nil
	}
	return n.ChildByFieldName(name, c.lang)
}

// loc returns "L<row>" for a tree-sitter node.
func (c *tsCtx) loc(n *gts.Node) string {
	return fmt.Sprintf("L%d", n.StartPoint().Row+1)
}

// parseTS parses src with the given language and returns the tree's root node
// alongside a tsCtx pre-populated with lang+src. Callers don't need to Close
// anything — gotreesitter is pure Go, tree memory is GC'd.
func parseTS(path string, src []byte, lang *gts.Language) (*gts.Node, tsCtx, error) {
	p := gts.NewParser(lang)
	tree, err := p.Parse(src)
	if err != nil {
		return nil, tsCtx{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if tree == nil {
		return nil, tsCtx{}, fmt.Errorf("parse failed: %s", path)
	}
	return tree.RootNode(), tsCtx{src: src, lang: lang}, nil
}

// supportedExtensions enumerates file extensions WalkRepo will dispatch.
// Keep this synchronised with ExtractFile's switch.
var supportedExtensions = map[string]bool{
	".go":  true,
	".py":  true,
	".js":  true,
	".mjs": true,
	".cjs": true,
	".jsx": true,
	".ts":  true,
	".tsx": true,
}

// findMethodByName returns the unique method-node ID whose label ends in
// ".name()". Returns "" when the name is ambiguous or unknown, matching the
// resolver's conservative stance in extract.go:resolveCrossFileCalls.
func findMethodByName(nodeMap map[string]*Node, name string) string {
	var found string
	suffix := "." + name + "()"
	for id, n := range nodeMap {
		if n.Kind != "method" || !strings.HasSuffix(n.Label, suffix) {
			continue
		}
		if found != "" {
			return ""
		}
		found = id
	}
	return found
}

// isTestFile reports whether a filename matches per-language test conventions
// the extractors skip at walk time.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)

	switch ext {
	case ".go":
		return strings.HasSuffix(lower, "_test.go")
	case ".py":
		return strings.HasPrefix(lower, "test_") || strings.HasSuffix(lower, "_test.py")
	case ".js", ".mjs", ".cjs", ".jsx", ".ts", ".tsx":
		base := strings.TrimSuffix(lower, ext)
		return strings.HasSuffix(base, ".test") || strings.HasSuffix(base, ".spec")
	}
	return false
}
