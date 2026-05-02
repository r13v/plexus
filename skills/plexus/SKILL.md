---
name: plexus
description: Query a code graph for callers, neighbors, shortest paths, god nodes, Louvain communities, and keyword-driven subgraphs in Go / Python / JavaScript / TypeScript repos. Use when the user asks "who calls X", "what does X depend on", "find the path between A and B", "what are the central / god classes / hubs in this repo", "show me the auth slice", "list modules / communities", "кто вызывает", "найти путь", "архитектурные узлы". Backed by the `plexus` CLI; runs locally against a git working tree.
when_to_use: |
  Trigger on any of these intents about the current repository:
  - "who calls <symbol>" / "find callers of <symbol>" / "reverse usages"
  - "what does <symbol> call / depend on" / "neighbors of <symbol>"
  - "shortest path between <A> and <B>" / "how does A reach B"
  - "god nodes" / "hubs" / "most-connected" / "central classes"
  - "communities" / "modules" / "clusters" in the code
  - "search the codebase for <topic> as a graph" / "subgraph for <topic>"
  - "render an architecture diagram of <slice>" (DOT → SVG)
  Do NOT trigger for: plain text grep ("find string X"), single-file edits,
  build/test orchestration, or non-code-graph questions.
allowed-tools: Bash(plexus *), Bash(dot *)
---

# plexus — code-graph queries

`plexus` answers structural questions about a repo by parsing it with
tree-sitter and persisting a node/edge graph to an XDG cache. Use it
instead of grep when the question is *relational* (who calls, what
depends, shortest path, central nodes).

## Pre-flight

Run once per session against the target repo:

```bash
cd <repo-root>
plexus build   # parses + caches; safe to skip — other commands auto-build
```

Requirements: `git` on PATH, target dir is a git working tree (file
walking goes through `git ls-files`).

## Decision table — which command

| User asks                                           | Command                                  |
|-----------------------------------------------------|------------------------------------------|
| "who calls FOO"                                     | `plexus callers FOO`                     |
| "what does FOO call / depend on"                    | `plexus neighbors FOO`                   |
| "filter neighbors to imports / calls / inherits"    | `plexus neighbors FOO --relation=calls`  |
| "shortest path A → B"                               | `plexus path A B`                        |
| "show node FOO"                                     | `plexus node FOO`                        |
| "search graph for <topic>"                          | `plexus search <topic> --depth 3`        |
| "god classes / hubs / most-connected"               | `plexus gods --top-n 20`                 |
| "list modules / clusters"                           | `plexus stats` then `plexus community N` |
| "render architecture as svg"                        | `plexus search <topic> --format=dot \| dot -Tsvg > arch.svg` |
| "dump the whole graph for inspection"               | `plexus dump` (always JSON)              |

## Output format

Pass `--format=text|json|dot` (default `text`).

- `text` — human-readable; respects `--token-budget` for `search`/`path`.
- `json` — pipe to `jq` for programmatic processing.
- `dot` — feed to Graphviz `dot` for SVG/PNG diagrams.
  Available for `subgraph` / `path` / `neighbors` / `callers` / `community`;
  NOT for `stats` / `gods` (no graph payload).

`plexus dump` always emits JSON regardless of `--format`.

## Conventions

- **Labels are case-insensitive** for `node` / `neighbors` / `callers` / `path`.
- **Symbol labels** look like `pkg.Func`, `Class.method`, or `path/to/file.go`.
  Use `plexus search <term>` first if unsure of the exact label.
- **Depth** for `search`: 1–6 (default 3). Higher = wider blast radius.
- **Token budget**: caps text output for `search`/`path` only — tabular
  commands (`stats`, `gods`, etc.) are never truncated. Bump
  `--token-budget` (max 50000) if results get cut off.
- **`--top-n`** for `gods`: 1–100 (default 10).
- **Cache**: `--cache-dir <parent>` or `PLEXUS_CACHE_DIR=<parent>`
  overrides default XDG path. Cache invalidates when `git rev-parse HEAD`
  changes; stale-cache warning goes to stderr.

## Common pitfalls

- **"node not found"** — try `plexus search <substring>` to discover the
  actual label form, then re-run with that exact label.
- **"no path found"** — `path` walks forward edges only (caller → callee,
  importer → imported). Reverse questions need `callers`, not `path`.
- **Empty `gods` / `stats`** — repo has no committed code in a supported
  language, or you forgot to `git add` & `git commit` after writing fixtures.
  `plexus build --repo <path>` requires a real git working tree.
- **DOT request on `stats`/`gods`** — returns a comment line; switch to
  `--format=text` or `--format=json`.

## Supported languages

Go, Python, JavaScript (`.js`/`.mjs`/`.cjs`/`.jsx`), TypeScript (`.ts`/`.tsx`).

## Edge relations

`contains` (file → symbol), `method` (type → method), `calls`
(function → callee), `imports` (file → file), `inherits` (class → base).

## See also

- `reference.md` — full subcommand + flag reference
- `examples.md` — five end-to-end recipes
