# plexus — full command reference

Mirrors `plexus --help` and `plexus <cmd> --help` in markdown form.

## Global flags

| Flag                | Default | Description                                                       |
|---------------------|---------|-------------------------------------------------------------------|
| `-r, --repo <path>` | `.`     | Path to the repository root (must be a git working tree).         |
| `--cache-dir <dir>` | —       | Override cache parent dir. Precedence: flag > `PLEXUS_CACHE_DIR` env > `os.UserCacheDir()/plexus`. |
| `--format <fmt>`    | `text`  | Output format: `text`, `json`, or `dot`. Ignored by `dump` (always JSON). |
| `-h, --help`        | —       | Show help.                                                        |

Cache layout: `<parent>/<basename>-<hash8>/code_graph.gob`, where
`basename = filepath.Base(toplevel)` and `hash8 = hex(sha256(toplevel))[:8]`.

## Commands

### `plexus build`

Build (or rebuild) the code graph cache for the current repo.

```
plexus build [global flags]
```

No command-specific flags. Resolves repo via `git rev-parse --show-toplevel`,
captures HEAD via `git rev-parse HEAD`, walks tracked files via
`git ls-files -co --exclude-standard -z`, then writes the cache.

### `plexus stats`

Show graph size and top-degree nodes.

```
plexus stats [global flags]
```

Output kinds: `text` table, `json` `{kind:"stats", stats:{node_count,
edge_count, community_count, top_by_degree:[{node, degree}]}}`.
DOT not supported (no graph payload).

### `plexus gods`

List top architectural hubs (non-file nodes by degree).

```
plexus gods [-n N] [global flags]
```

| Flag             | Default | Range  |
|------------------|---------|--------|
| `-n, --top-n N`  | 10      | 1–100  |

Output kinds: `text` table, `json` `{kind:"gods", gods:[{node, degree}]}`.
DOT not supported.

All `--format=json` output is wrapped in a `Result` envelope with a `kind`
discriminator and the per-kind payload. Use `plexus <cmd> --format=json | jq`
to inspect exact shapes; field names are lowercase per the JSON tags.

### `plexus search <query...>`

Keyword-search the graph and render a relevant subgraph.

```
plexus search <q1> [q2 ...] [--depth N] [--mode bfs|dfs] [--token-budget T]
```

| Flag               | Default | Range/values     |
|--------------------|---------|------------------|
| `--depth N`        | 3       | 1–6              |
| `--mode <m>`       | `bfs`   | `bfs` or `dfs`   |
| `--token-budget T` | 4000    | 1–50000 (text only) |

Multiple positional args are AND-matched; quote phrases. CLI verb
`search` maps to internal `Input.Action="query"`.

### `plexus node <label>`

Render a single node by exact label (case-insensitive).

```
plexus node <label> [global flags]
```

Errors with `not found` or `ambiguous` if the label doesn't resolve to
exactly one node.

### `plexus neighbors <label>`

Show outgoing edges from a node.

```
plexus neighbors <label> [--relation <rel>]
```

| Flag                 | Allowed values                                          |
|----------------------|---------------------------------------------------------|
| `--relation <rel>`   | `contains`, `method`, `inherits`, `imports`, `calls`    |

Omit `--relation` to see all outgoing edges.

### `plexus callers <label>`

Show reverse `calls` edges into a node (who calls this).

```
plexus callers <label> [global flags]
```

### `plexus path <source> <target>`

Shortest forward-edge path between two nodes.

```
plexus path <source> <target> [--token-budget T]
```

| Flag               | Default | Range          |
|--------------------|---------|----------------|
| `--token-budget T` | 4000    | 1–50000 (text) |

Walks forward edges only. Reports "no path found" when none exists.

### `plexus community <id>`

List all node labels in a Louvain community cluster.

```
plexus community <id> [global flags]
```

Discover IDs via `plexus stats` (community count) or by inspecting
`plexus dump`.

### `plexus dump`

Dump the cached graph as pretty-printed JSON. `--format` is ignored
(always JSON). Useful for inspection / scripting.

```
plexus dump [global flags]
```

### `plexus version`

Print version + build info (uses `runtime/debug.ReadBuildInfo`).

```
plexus version
```

### `plexus completion <shell>`

Generate shell-completion script (provided by Cobra). Supports
`bash`, `zsh`, `fish`, `powershell`.

## Environment variables

| Variable             | Purpose                                              |
|----------------------|------------------------------------------------------|
| `PLEXUS_CACHE_DIR`   | Override cache parent dir (lower precedence than `--cache-dir`). |

## Exit status

- `0` — success.
- non-zero — git missing, repo not a working tree, label unresolved,
  cache I/O error, or other failure (message printed to stderr).
