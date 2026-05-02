# plexus — end-to-end recipes

Each recipe assumes you've `cd`'d into a git working tree of a Go /
Python / JS / TS repo. `plexus build` is implicit on the first run of
any query (the cache is built on demand).

Symbol labels are how the graph stores names: functions and methods
include `()` (e.g. `BuildFromRepo()`, `Service.Run()`); types/structs
do not (e.g. `Service`). All `--format=json` output is wrapped in a
`Result` envelope with a `kind` discriminator and the per-kind payload.

## 1. Find every caller of a function

Goal: who invokes `BuildFromRepo`, and from which file?

```bash
plexus callers 'BuildFromRepo()' --format=json \
  | jq '.callers.callers[] | {src: .id, file: .source_file}'
```

Use the plain text form for a quick read:

```bash
plexus callers 'BuildFromRepo()'
```

If the label is ambiguous or unknown, locate it first:

```bash
plexus search BuildFromRepo --depth 1
```

## 2. Find god classes / architectural hubs

Goal: list the 20 most-connected non-file symbols.

```bash
plexus gods --top-n 20 --format=json \
  | jq -r '.gods[] | "\(.degree)\t\(.node.label)"'
```

Combine with `community` to see what cluster each hub anchors:

```bash
for id in $(plexus gods --top-n 5 --format=json | jq -r '.gods[].node.community'); do
  echo "== community $id =="
  plexus community "$id"
done
```

## 3. Visualize an architecture slice as SVG

Goal: render the "auth" subgraph as an SVG diagram.

```bash
plexus search auth --depth 2 --format=dot | dot -Tsvg > auth.svg
open auth.svg   # macOS; xdg-open on Linux
```

Wider blast radius, fewer nodes via a stricter token budget on the
text fallback:

```bash
plexus search auth --depth 4 --token-budget 8000
```

## 4. Find a path between two functions

Goal: how does `main` reach `BuildFromRepo`?

```bash
plexus path 'main()' 'BuildFromRepo()'
```

Output is a forward edge sequence (caller → callee, importer → imported).
If `path` reports "no path found", try the reverse direction or use
`callers` to walk back from the target.

## 5. List members of a community / module cluster

Goal: see every symbol grouped into Louvain community 3.

```bash
plexus community 3 --format=json | jq -r '.community.nodes[].label'
```

Discover community IDs first via `plexus stats` (prints community
count) or by inspecting `plexus dump`:

```bash
plexus dump | jq '.communities | keys'
```

## Bonus: scripting against `plexus dump`

`plexus dump` always emits JSON with the full `Graph` shape (lowercase
keys: `nodes`, `edges`, `communities`). Useful for one-off audits:

```bash
# Total nodes vs. unique files
plexus dump | jq '{nodes: (.nodes|length), files: ([.nodes[] | select(.kind=="file")] | length)}'

# Every symbol that imports a given file
plexus dump | jq --arg f 'internal/auth/session.go' \
  '[.edges[] | select(.relation=="imports" and .target==$f) | .source]'
```
