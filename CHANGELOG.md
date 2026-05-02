# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-02

Initial release. Standalone CLI for code-graph queries.

### Added

- `plexus version` — print binary version, commit, and build date
- `plexus build` — build code graph from a git repository, cached to XDG cache dir
- `plexus stats` — node/edge/community counts
- `plexus gods` — top architectural hubs by degree
- `plexus search <query>` — keyword search returning a relevant subgraph
- `plexus node <label>` — inspect a single symbol/file
- `plexus neighbors <label>` — direct neighbors of a node, optionally filtered by relation
- `plexus callers <label>` — callers/usages of a symbol
- `plexus path <from> <to>` — shortest path between two symbols
- `plexus community <id>` — list members of a Louvain community
- `plexus dump` — dump full graph as JSON for debugging
- Output formats: `--format=text|json|dot`
- Languages supported: Go, Python, JavaScript (`.js/.mjs/.cjs/.jsx`),
  TypeScript (`.ts/.tsx`)
- Cache: gob format under `os.UserCacheDir()/plexus/<basename>-<hash8>/code_graph.gob`
  with `--cache-dir` flag and `PLEXUS_CACHE_DIR` env override
- Git integration: subprocess `git` calls (hard requirement, no fallback);
  file walking respects `.gitignore` via `git ls-files`
- AI-agent skill bundle under `skills/plexus/` (Anthropic Agent Skills format),
  installable via `npx skills add r13v/plexus`
- Distribution: GitHub Releases (multi-OS), Homebrew tap `r13v/homebrew-apps`,
  `install.sh` (curl|sh), `go install github.com/r13v/plexus/cmd/plexus@latest`
- CI: GitHub Actions test + lint, conventional-commits-driven auto-tagging
- Release: goreleaser on tag push (linux/darwin amd64+arm64, windows amd64)

[Unreleased]: https://github.com/r13v/plexus/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/r13v/plexus/releases/tag/v0.1.0
