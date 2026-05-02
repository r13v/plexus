package codegraph

import (
	"path"
	"path/filepath"
	"strings"
)

// jsExtCandidates lists file extensions tried when resolving a JS/TS import
// with no explicit extension. Order reflects TypeScript's preference (TS > JS,
// TSX > JSX) so `.tsx` wins over `.jsx` when both exist.
var jsExtCandidates = []string{".tsx", ".ts", ".jsx", ".js", ".mjs", ".cjs"}

// jsIndexNames lists candidate index file basenames (without extension) tried
// when an import points to a directory rather than a file.
var jsIndexNames = []string{"index"}

// Resolver turns PendingImports into either internal file-node edges or
// external import-node labels. It is built once per walk from the set of
// known file paths and, optionally, a parsed tsconfig.
type Resolver struct {
	files    map[string]struct{}
	tsconfig *tsConfig
}

// NewResolver builds a Resolver from the file nodes produced during a walk.
// Only file-kind nodes contribute to the lookup set.
func NewResolver(nodes []*Node, tsconfig *tsConfig) *Resolver {
	files := make(map[string]struct{})
	for _, n := range nodes {
		if n.Kind != "file" {
			continue
		}
		files[filepath.ToSlash(n.SourceFile)] = struct{}{}
	}
	return &Resolver{files: files, tsconfig: tsconfig}
}

// Resolve returns either a known-file target path (internal=true) or an
// external label to use for an import node (internal=false). The language
// selects JS/TS vs Python rules. Go imports are not routed through here;
// extract_go.go creates its own import nodes.
func (r *Resolver) Resolve(pi PendingImport) (string, bool) {
	switch pi.Language {
	case "js", "ts":
		return r.resolveJS(pi)
	case "python":
		return r.resolvePython(pi)
	default:
		return pi.Spec, false
	}
}

func (r *Resolver) resolveJS(pi PendingImport) (string, bool) {
	spec := pi.Spec
	if isRelativeJS(spec) {
		base := path.Join(filepath.ToSlash(filepath.Dir(pi.SourceFile)), spec)
		base = path.Clean(base)
		if hit, ok := r.tryJSCandidates(base); ok {
			return hit, true
		}
		// Unresolved relative: return the normalized repo-root-relative
		// label so that `./messages` from different dirs dedupe correctly.
		return base, false
	}

	if r.tsconfig != nil {
		for _, cand := range r.tsconfig.resolveAlias(spec) {
			cand = filepath.ToSlash(cand)
			if hit, ok := r.tryJSCandidates(cand); ok {
				return hit, true
			}
		}
	}

	return spec, false
}

// tryJSCandidates checks base against each extension and index-file form,
// returning the first repo-root-relative path present in the file set.
func (r *Resolver) tryJSCandidates(base string) (string, bool) {
	if _, ok := r.files[base]; ok {
		return base, true
	}
	for _, ext := range jsExtCandidates {
		cand := base + ext
		if _, ok := r.files[cand]; ok {
			return cand, true
		}
	}
	for _, idx := range jsIndexNames {
		for _, ext := range jsExtCandidates {
			cand := path.Join(base, idx+ext)
			if _, ok := r.files[cand]; ok {
				return cand, true
			}
		}
	}
	return "", false
}

func isRelativeJS(spec string) bool {
	return strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") ||
		spec == "." || spec == ".."
}

func (r *Resolver) resolvePython(pi PendingImport) (string, bool) {
	spec := pi.Spec
	if !strings.HasPrefix(spec, ".") {
		return spec, false
	}

	dots := 0
	for dots < len(spec) && spec[dots] == '.' {
		dots++
	}
	rest := spec[dots:] // may be "", "foo", "foo.bar"

	dir := filepath.ToSlash(filepath.Dir(pi.SourceFile))
	if dir == "." {
		dir = ""
	}

	// Walk up (dots - 1) directory levels. `from . import x` stays in the
	// current package; `from ..` goes one level up, and so on.
	base := dir
	for range dots - 1 {
		if base == "" {
			// Walked past repo root; keep the normalized unresolved label.
			return normalizePyLabel(dir, spec), false
		}
		base = path.Dir(base)
		if base == "." {
			base = ""
		}
	}

	restPath := strings.ReplaceAll(rest, ".", "/")
	var candidates []string
	if restPath == "" {
		candidates = []string{path.Join(base, "__init__.py")}
	} else {
		candidates = []string{
			path.Join(base, restPath+".py"),
			path.Join(base, restPath, "__init__.py"),
		}
	}
	for _, cand := range candidates {
		if _, ok := r.files[cand]; ok {
			return cand, true
		}
	}

	return normalizePyLabel(dir, spec), false
}

// normalizePyLabel produces a stable label for an unresolved Python relative
// import so that the same reference from different files dedupes. It walks up
// the same number of levels the import implied and joins the tail, matching
// what resolvePython computes for candidates.
func normalizePyLabel(dir, spec string) string {
	dots := 0
	for dots < len(spec) && spec[dots] == '.' {
		dots++
	}
	rest := strings.ReplaceAll(spec[dots:], ".", "/")
	base := dir
	for range dots - 1 {
		if base == "" {
			break
		}
		base = path.Dir(base)
		if base == "." {
			base = ""
		}
	}
	return path.Clean(path.Join(base, rest))
}
