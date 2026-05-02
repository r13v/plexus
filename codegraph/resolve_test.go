package codegraph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveJSRelativeWithExtension(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("src/pages/foo.tsx"), Kind: "file", SourceFile: "src/pages/foo.tsx"},
		{ID: MakeID("src/pages/messages.tsx"), Kind: "file", SourceFile: "src/pages/messages.tsx"},
	}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{
		Spec: "./messages", Language: "ts", SourceFile: "src/pages/foo.tsx",
	})
	if !internal {
		t.Fatalf("expected internal resolution; got external=%q", got)
	}
	if got != "src/pages/messages.tsx" {
		t.Errorf("got %q, want src/pages/messages.tsx", got)
	}
}

func TestResolveJSRelativeDedupesAcrossCallers(t *testing.T) {
	// "./messages" from two different source files in different dirs must
	// resolve to two DIFFERENT file targets — that's the point of the fix.
	nodes := []*Node{
		{ID: MakeID("src/a/caller.tsx"), Kind: "file", SourceFile: "src/a/caller.tsx"},
		{ID: MakeID("src/a/messages.tsx"), Kind: "file", SourceFile: "src/a/messages.tsx"},
		{ID: MakeID("src/b/caller.tsx"), Kind: "file", SourceFile: "src/b/caller.tsx"},
		{ID: MakeID("src/b/messages.tsx"), Kind: "file", SourceFile: "src/b/messages.tsx"},
	}
	r := NewResolver(nodes, nil)

	gotA, _ := r.Resolve(PendingImport{Spec: "./messages", Language: "ts", SourceFile: "src/a/caller.tsx"})
	gotB, _ := r.Resolve(PendingImport{Spec: "./messages", Language: "ts", SourceFile: "src/b/caller.tsx"})
	if gotA == gotB {
		t.Errorf("expected different resolutions for ./messages from different dirs, got both %q", gotA)
	}
}

func TestResolveJSRelativeUnifiesAcrossSpecForms(t *testing.T) {
	// "./messages" from src/pages/foo.tsx and "../messages" from
	// src/pages/subdir/bar.tsx both point at src/pages/messages.tsx.
	nodes := []*Node{
		{ID: MakeID("src/pages/foo.tsx"), Kind: "file", SourceFile: "src/pages/foo.tsx"},
		{ID: MakeID("src/pages/subdir/bar.tsx"), Kind: "file", SourceFile: "src/pages/subdir/bar.tsx"},
		{ID: MakeID("src/pages/messages.tsx"), Kind: "file", SourceFile: "src/pages/messages.tsx"},
	}
	r := NewResolver(nodes, nil)

	gotA, _ := r.Resolve(PendingImport{Spec: "./messages", Language: "ts", SourceFile: "src/pages/foo.tsx"})
	gotB, _ := r.Resolve(PendingImport{Spec: "../messages", Language: "ts", SourceFile: "src/pages/subdir/bar.tsx"})
	if gotA != gotB || gotA != "src/pages/messages.tsx" {
		t.Errorf("expected both to resolve to src/pages/messages.tsx, got %q and %q", gotA, gotB)
	}
}

func TestResolveJSRelativeIndexFile(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("src/foo.ts"), Kind: "file", SourceFile: "src/foo.ts"},
		{ID: MakeID("src/widgets/index.ts"), Kind: "file", SourceFile: "src/widgets/index.ts"},
	}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{Spec: "./widgets", Language: "ts", SourceFile: "src/foo.ts"})
	if !internal || got != "src/widgets/index.ts" {
		t.Errorf("got (%q, %v), want (src/widgets/index.ts, true)", got, internal)
	}
}

func TestResolveJSUnresolvedRelativeNormalized(t *testing.T) {
	// Missing target should fall back to a normalized repo-root-relative
	// label so multiple callers with the same unresolvable target dedupe.
	nodes := []*Node{
		{ID: MakeID("src/a/x.tsx"), Kind: "file", SourceFile: "src/a/x.tsx"},
		{ID: MakeID("src/b/x.tsx"), Kind: "file", SourceFile: "src/b/x.tsx"},
	}
	r := NewResolver(nodes, nil)

	// "./missing" from src/a/x.tsx normalizes to "src/a/missing".
	gotA, internalA := r.Resolve(PendingImport{Spec: "./missing", Language: "ts", SourceFile: "src/a/x.tsx"})
	if internalA {
		t.Fatal("expected external (unresolved) result")
	}
	if gotA != "src/a/missing" {
		t.Errorf("got %q, want src/a/missing", gotA)
	}

	// Same spec from b/ normalizes differently (different dir).
	gotB, _ := r.Resolve(PendingImport{Spec: "./missing", Language: "ts", SourceFile: "src/b/x.tsx"})
	if gotB == gotA {
		t.Errorf("expected different normalized labels for different dirs, got %q for both", gotA)
	}
}

func TestResolveJSBareModuleRemainsExternal(t *testing.T) {
	nodes := []*Node{{ID: MakeID("app.ts"), Kind: "file", SourceFile: "app.ts"}}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{Spec: "react", Language: "js", SourceFile: "app.ts"})
	if internal {
		t.Error("bare specifier should not resolve to a file node")
	}
	if got != "react" {
		t.Errorf("got %q, want react", got)
	}
}

func TestResolveJSTsconfigAlias(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("src/utils/format.ts"), Kind: "file", SourceFile: "src/utils/format.ts"},
		{ID: MakeID("src/app.ts"), Kind: "file", SourceFile: "src/app.ts"},
	}
	cfg := &tsConfig{
		BaseURL:  ".",
		Paths:    map[string][]string{"@/*": {"src/*"}},
		root:     "/fake/repo",
		repoRoot: "/fake/repo",
	}
	r := NewResolver(nodes, cfg)

	got, internal := r.Resolve(PendingImport{Spec: "@/utils/format", Language: "ts", SourceFile: "src/app.ts"})
	if !internal || got != "src/utils/format.ts" {
		t.Errorf("got (%q, %v), want (src/utils/format.ts, true)", got, internal)
	}
}

func TestResolvePythonRelativeSameDir(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("pkg/main.py"), Kind: "file", SourceFile: "pkg/main.py"},
		{ID: MakeID("pkg/util.py"), Kind: "file", SourceFile: "pkg/util.py"},
	}
	r := NewResolver(nodes, nil)

	// "from . import util" emits spec ".util"
	got, internal := r.Resolve(PendingImport{Spec: ".util", Language: "python", SourceFile: "pkg/main.py"})
	if !internal || got != "pkg/util.py" {
		t.Errorf("got (%q, %v), want (pkg/util.py, true)", got, internal)
	}
}

func TestResolvePythonRelativeDottedModule(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("pkg/main.py"), Kind: "file", SourceFile: "pkg/main.py"},
		{ID: MakeID("pkg/sub/mod.py"), Kind: "file", SourceFile: "pkg/sub/mod.py"},
	}
	r := NewResolver(nodes, nil)

	// "from .sub.mod import X" emits spec ".sub.mod"
	got, internal := r.Resolve(PendingImport{Spec: ".sub.mod", Language: "python", SourceFile: "pkg/main.py"})
	if !internal || got != "pkg/sub/mod.py" {
		t.Errorf("got (%q, %v), want (pkg/sub/mod.py, true)", got, internal)
	}
}

func TestResolvePythonRelativeParentPackage(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("pkg/sub/mod.py"), Kind: "file", SourceFile: "pkg/sub/mod.py"},
		{ID: MakeID("pkg/util.py"), Kind: "file", SourceFile: "pkg/util.py"},
	}
	r := NewResolver(nodes, nil)

	// "from ..util import X" from pkg/sub/mod.py → pkg/util.py
	got, internal := r.Resolve(PendingImport{Spec: "..util", Language: "python", SourceFile: "pkg/sub/mod.py"})
	if !internal || got != "pkg/util.py" {
		t.Errorf("got (%q, %v), want (pkg/util.py, true)", got, internal)
	}
}

func TestResolvePythonPackageInit(t *testing.T) {
	nodes := []*Node{
		{ID: MakeID("pkg/main.py"), Kind: "file", SourceFile: "pkg/main.py"},
		{ID: MakeID("pkg/sub/__init__.py"), Kind: "file", SourceFile: "pkg/sub/__init__.py"},
	}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{Spec: ".sub", Language: "python", SourceFile: "pkg/main.py"})
	if !internal || got != "pkg/sub/__init__.py" {
		t.Errorf("got (%q, %v), want (pkg/sub/__init__.py, true)", got, internal)
	}
}

func TestResolvePythonUnresolvedNormalized(t *testing.T) {
	nodes := []*Node{{ID: MakeID("pkg/main.py"), Kind: "file", SourceFile: "pkg/main.py"}}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{Spec: ".missing", Language: "python", SourceFile: "pkg/main.py"})
	if internal {
		t.Error("expected unresolved for missing target")
	}
	if got != "pkg/missing" {
		t.Errorf("got %q, want pkg/missing", got)
	}
}

func TestResolvePythonAbsoluteRemainsExternal(t *testing.T) {
	nodes := []*Node{{ID: MakeID("pkg/main.py"), Kind: "file", SourceFile: "pkg/main.py"}}
	r := NewResolver(nodes, nil)

	got, internal := r.Resolve(PendingImport{Spec: "os.path", Language: "python", SourceFile: "pkg/main.py"})
	if internal {
		t.Error("absolute/external import should not resolve internally")
	}
	if got != "os.path" {
		t.Errorf("got %q, want os.path", got)
	}
}

func TestWalkRepoResolvesRelativeJSImport(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.js"), "import { Service } from './service.js';\n")
	mustWrite(t, filepath.Join(dir, "service.js"), "export class Service { run() {} }\n")
	gitInitFixture(t, dir)

	nodes, edges, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	mainID := MakeID("main.js")
	svcID := MakeID("service.js")
	var found bool
	for _, e := range edges {
		if e.Relation == "imports" && e.Source == mainID && e.Target == svcID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected main.js -imports-> service.js edge; edges: %+v", edges)
	}

	for _, n := range nodes {
		if n.Kind == "import" && n.Label == "./service.js" {
			t.Errorf("expected no import node for ./service.js — it should resolve to file node")
		}
	}
}

func TestWalkRepoExternalJSImportNode(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.js"), "import React from 'react';\n")
	gitInitFixture(t, dir)

	nodes, edges, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	var have bool
	for _, n := range nodes {
		if n.Kind == "import" && n.Label == "react" {
			have = true
			break
		}
	}
	if !have {
		t.Fatal("expected external import node 'react'")
	}
	mainID := MakeID("main.js")
	reactID := MakeID("import", "react")
	var edgeFound bool
	for _, e := range edges {
		if e.Relation == "imports" && e.Source == mainID && e.Target == reactID {
			edgeFound = true
			break
		}
	}
	if !edgeFound {
		t.Fatal("expected main.js -imports-> react edge")
	}
}

func TestWalkRepoUnresolvedRelativeDedupes(t *testing.T) {
	// Two files in different dirs each import "./missing". Without
	// normalization, they'd become one node "./missing" with an inflated
	// degree. With normalization, they become two distinct nodes reflecting
	// the actual (different) target paths.
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "a"))
	mustMkdir(t, filepath.Join(dir, "b"))
	mustWrite(t, filepath.Join(dir, "a", "x.js"), "import './missing';\n")
	mustWrite(t, filepath.Join(dir, "b", "x.js"), "import './missing';\n")
	gitInitFixture(t, dir)

	nodes, _, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	var labels []string
	for _, n := range nodes {
		if n.Kind == "import" {
			labels = append(labels, n.Label)
		}
	}
	hasA := false
	hasB := false
	for _, l := range labels {
		if l == "a/missing" {
			hasA = true
		}
		if l == "b/missing" {
			hasB = true
		}
		if l == "./missing" {
			t.Errorf("raw './missing' label should have been normalized; have: %+v", labels)
		}
	}
	if !hasA || !hasB {
		t.Errorf("expected both 'a/missing' and 'b/missing' labels, got: %+v", labels)
	}
}

func TestWalkRepoPythonRelativeResolves(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "pkg"))
	mustWrite(t, filepath.Join(dir, "pkg", "__init__.py"), "")
	mustWrite(t, filepath.Join(dir, "pkg", "main.py"), "from . import util\nfrom .util import helper\n")
	mustWrite(t, filepath.Join(dir, "pkg", "util.py"), "def helper(): pass\n")
	gitInitFixture(t, dir)

	_, edges, _, err := WalkRepo(dir)
	if err != nil {
		t.Fatal(err)
	}

	mainID := MakeID(filepath.ToSlash(filepath.Join("pkg", "main.py")))
	utilID := MakeID(filepath.ToSlash(filepath.Join("pkg", "util.py")))
	count := 0
	for _, e := range edges {
		if e.Relation == "imports" && e.Source == mainID && e.Target == utilID {
			count++
		}
	}
	if count < 1 {
		t.Fatalf("expected at least one imports edge main.py → util.py; got %d", count)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
