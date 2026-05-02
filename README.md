# plexus

Standalone CLI for code-graph queries: callers, neighbors, shortest paths, god
nodes, Louvain communities, and keyword search across a repository.

Plexus parses a repo with tree-sitter (pure Go, no cgo), resolves imports
across files, and persists the resulting node/edge graph in an XDG cache.
Subsequent queries hit the cache and answer in milliseconds.

## Supported languages

v0.1 ships extractors for:

- Go
- Python
- JavaScript (`.js`, `.mjs`, `.cjs`, `.jsx`)
- TypeScript (`.ts`, `.tsx`)

Additional languages are explicit future work.

## Edge relations

| Relation   | Meaning                                          |
|------------|--------------------------------------------------|
| `contains` | file → top-level symbol declared in that file    |
| `method`   | type → method declared on that type              |
| `calls`    | function/method → callee resolved in scope       |
| `imports`  | file → file resolved via import / require / from |
| `inherits` | class → base class                               |

## Install

```bash
brew install r13v/apps/plexus
# or
curl -fsSL https://raw.githubusercontent.com/r13v/plexus/main/install.sh | sh
# or
go install github.com/r13v/plexus/cmd/plexus@latest
```

## Quick start

```bash
cd path/to/your/repo
plexus build           # parse + cache the graph (re-run after large changes)
plexus stats           # nodes / edges / communities
plexus gods --top-n 20 # architectural hubs by degree
```

`plexus` requires `git` on PATH and the target directory to be a git
working tree — file walking goes through `git ls-files`, so `.gitignore`
is respected by construction.

## Examples

### 1. Build the graph

```bash
plexus build
```

The cache is written to `$XDG_CACHE_HOME/plexus/<basename>-<hash8>/code_graph.gob`
(override with `--cache-dir` or `PLEXUS_CACHE_DIR`). Most commands auto-build
on the first invocation if no cache is present.

### 2. Find callers of a symbol

```bash
plexus callers 'BuildFromRepo()'
```

Use `--format=json` to pipe into `jq`:

```bash
plexus callers 'BuildFromRepo()' --format=json | jq '.callers.callers[].id'
```

### 3. Keyword search → relevant subgraph

```bash
plexus search auth --depth 2
```

`--mode=bfs|dfs` controls graph traversal strategy; `--depth` (default 3,
max 6) caps how far to expand from matched nodes; `--token-budget` caps
text output (text format only).

### 4. List god nodes (architectural hubs)

```bash
plexus gods --top-n 10 --format=json | jq '.gods[] | {label: .node.label, degree}'
```

### 5. Shortest path between two symbols

```bash
plexus path 'main()' 'BuildFromRepo()'
```

Labels are the symbol names as they appear in the graph (functions and
methods include `()`; types/structs do not). Use `plexus search` or
`plexus dump | jq '.nodes | keys'` to discover exact labels.

Returns a path of forward edges, or reports that no path exists.

### 6. Render an architecture slice as SVG (DOT pipeline)

```bash
plexus search auth --format=dot | dot -Tsvg > arch.svg
```

DOT output is available for `subgraph` / `path` / `neighbors` / `callers` /
`community` results. `stats` and `gods` have no graph payload — use
`--format=text` or `--format=json` for those.

## Output formats

Pass `--format=text|json|dot` (default `text`). The token budget applies
only to text rendering of the larger result kinds (subgraph, path) — tabular
kinds (stats, gods, community, node, neighbors, callers) are not truncated.

## Cache

- Default: `os.UserCacheDir()/plexus/<basename>-<hash8>/code_graph.gob`
  (typically `~/Library/Caches/plexus/...` on macOS or `~/.cache/plexus/...`
  on Linux).
- Override with `--cache-dir <parent>` or `PLEXUS_CACHE_DIR=<parent>`.
- `plexus dump` prints the cached graph as pretty JSON for debugging.
- The cache is invalidated when `git rev-parse HEAD` changes; `plexus`
  warns to stderr when serving a stale cache and rebuilds on demand.

## Releases

Plexus uses [Conventional Commits](https://www.conventionalcommits.org/) on
`main` to drive automatic version bumps:

| Commit prefix                             | Bump  |
|-------------------------------------------|-------|
| `feat:`                                   | minor |
| `fix:`                                    | patch |
| `feat!:` or `BREAKING CHANGE:` in body    | major |
| `chore:` / `docs:` / `refactor:` / `test:` / `ci:` | no release |

A merge to `main` with a qualifying commit triggers a tag bump via
`mathieudutour/github-tag-action`, which fires the goreleaser workflow that
publishes multi-OS archives + updates the Homebrew formula in
`r13v/homebrew-apps`.
