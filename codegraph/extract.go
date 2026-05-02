package codegraph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ignoreDirs = map[string]bool{
	".git":         true,
	".plexus":      true,
	"vendor":       true,
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"testdata":     true,
}

// UnresolvedCall records a call site that couldn't be resolved within a single file.
// WalkRepo resolves these across files in a post-pass.
type UnresolvedCall struct {
	CallerID       string
	CalleeName     string
	IsMethod       bool
	SourceFile     string
	SourceLocation string
}

// PendingImport records an import statement to be resolved against the full
// file set in a post-walk pass. The resolver turns relative specs into edges
// to existing file nodes and bare specs into external import nodes.
type PendingImport struct {
	Spec       string // raw specifier: "./messages", "react", ".foo.bar"
	Language   string // "js", "ts", "python"
	SourceFile string // repo-root-relative path of the importing file
}

// FileExtraction holds the output of extracting a single file.
type FileExtraction struct {
	Nodes      []*Node
	Edges      []Edge
	Unresolved []UnresolvedCall
	Imports    []PendingImport
}

// ExtractResult holds per-file extraction counters.
type ExtractResult struct {
	Processed int
	Skipped   int
	Errors    int
}

// ExtractFile dispatches to a language-specific extractor based on file extension.
func ExtractFile(path string, src []byte) (*FileExtraction, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return extractGo(path, src)
	case ".py":
		return extractPython(path, src)
	case ".js", ".mjs", ".cjs", ".jsx":
		return extractJS(path, src)
	case ".ts":
		return extractTS(path, src, false)
	case ".tsx":
		return extractTS(path, src, true)
	default:
		return nil, fmt.Errorf("unsupported extension: %s", ext)
	}
}

// WalkRepo walks a repository, extracting nodes and edges from supported files.
// File discovery uses `git ls-files` (so .gitignore is respected natively); the
// hardcoded ignoreDirs are an additional AND-filter applied to every path
// component to defend against committed `vendor/`, `node_modules/`, etc.
// Per-file errors degrade gracefully (counted, not propagated).
func WalkRepo(root string) ([]*Node, []Edge, ExtractResult, error) {
	var allNodes []*Node
	var allEdges []Edge
	var result ExtractResult

	var allUnresolved []UnresolvedCall
	var allImports []PendingImport

	files, err := GitLsFiles(root)
	if err != nil {
		return nil, nil, result, fmt.Errorf("codegraph: walk %s: %w", root, err)
	}

	for _, relPath := range files {
		if hasIgnoredComponent(relPath) {
			continue
		}
		base := filepath.Base(relPath)
		ext := strings.ToLower(filepath.Ext(base))
		if !supportedExtensions[ext] || isTestFile(base) {
			result.Skipped++
			continue
		}

		absPath := filepath.Join(root, relPath)
		src, readErr := os.ReadFile(absPath)
		if readErr != nil {
			result.Errors++
			continue
		}

		fe, extractErr := ExtractFile(relPath, src)
		if extractErr != nil {
			result.Errors++
			continue
		}

		allNodes = append(allNodes, fe.Nodes...)
		allEdges = append(allEdges, fe.Edges...)
		allUnresolved = append(allUnresolved, fe.Unresolved...)
		allImports = append(allImports, fe.Imports...)
		result.Processed++
	}

	allEdges = append(allEdges, resolveCrossFileCalls(allNodes, allUnresolved)...)

	tsconfig, _ := loadTSConfig(root)
	importNodes, importEdges := resolvePendingImports(allNodes, allImports, tsconfig)
	allNodes = append(allNodes, importNodes...)
	allEdges = append(allEdges, importEdges...)

	return allNodes, allEdges, result, nil
}

// hasIgnoredComponent reports whether any path component in relPath is in
// ignoreDirs. Path is split on "/" since git ls-files emits forward slashes.
func hasIgnoredComponent(relPath string) bool {
	for p := range strings.SplitSeq(filepath.ToSlash(relPath), "/") {
		if ignoreDirs[p] {
			return true
		}
	}
	return false
}

// resolvePendingImports turns each PendingImport into either (a) an edge
// to an existing file node, when the spec resolves to a known repo file, or
// (b) an edge to an external import node (created once per unique label).
// Unresolved relative imports fall into case (b) with a normalized,
// repo-root-relative label so that multiple callers pointing at the same
// missing target dedupe into a single node.
func resolvePendingImports(nodes []*Node, pending []PendingImport, tsconfig *tsConfig) ([]*Node, []Edge) {
	if len(pending) == 0 {
		return nil, nil
	}
	resolver := NewResolver(nodes, tsconfig)

	existing := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		existing[n.ID] = struct{}{}
	}

	var newNodes []*Node
	var newEdges []Edge
	for _, pi := range pending {
		target, internal := resolver.Resolve(pi)
		sourceID := MakeID(pi.SourceFile)
		var targetID string
		if internal {
			targetID = MakeID(target)
		} else {
			targetID = MakeID("import", target)
			if _, ok := existing[targetID]; !ok {
				newNodes = append(newNodes, &Node{
					ID:         targetID,
					Label:      target,
					Kind:       "import",
					SourceFile: pi.SourceFile,
				})
				existing[targetID] = struct{}{}
			}
		}
		newEdges = append(newEdges, Edge{
			Source:     sourceID,
			Target:     targetID,
			Relation:   "imports",
			SourceFile: pi.SourceFile,
		})
	}
	return newNodes, newEdges
}

// resolveCrossFileCalls matches unresolved calls against the global node set.
// When a name is ambiguous (multiple candidates), the call is skipped to avoid
// creating incorrect edges.
func resolveCrossFileCalls(nodes []*Node, unresolved []UnresolvedCall) []Edge {
	funcByName := make(map[string][]string)
	methodByName := make(map[string][]string)
	for _, n := range nodes {
		switch n.Kind {
		case "function":
			name := strings.TrimSuffix(n.Label, "()")
			funcByName[name] = append(funcByName[name], n.ID)
		case "method":
			parts := strings.SplitN(n.Label, ".", 2)
			if len(parts) == 2 {
				methodName := strings.TrimSuffix(parts[1], "()")
				methodByName[methodName] = append(methodByName[methodName], n.ID)
			}
		}
	}

	var edges []Edge
	for _, u := range unresolved {
		var candidates []string
		if u.IsMethod {
			candidates = methodByName[u.CalleeName]
		} else {
			candidates = funcByName[u.CalleeName]
		}
		if len(candidates) != 1 || candidates[0] == u.CallerID {
			continue
		}
		edges = append(edges, Edge{
			Source:         u.CallerID,
			Target:         candidates[0],
			Relation:       "calls",
			SourceFile:     u.SourceFile,
			SourceLocation: u.SourceLocation,
		})
	}
	return edges
}
