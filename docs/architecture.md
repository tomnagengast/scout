# Architecture

`scout` is a local CLI for building a compact map of a document tree. It discovers files or directories, reads bounded file heads, asks a local CLI agent for compact descriptions, caches unchanged summaries, and renders an index.

## Layers

The Cobra entrypoint lives in `cmd/main.go`. It delegates to the `internal` package.

Configuration lives in `internal/config.go`. It merges defaults, user config, project config, environment variables, and changed CLI flags.

Discovery lives in `internal/discover.go`. It resolves files, directories, and globs, applies `.gitignore`, `.scoutignore`, and config-level ignore patterns, then selects file or directory entries with optional max-depth bounds.

Summarization lives in `internal/summarize.go`. It reads at most `max_bytes` from each file, trims incomplete UTF-8 at the boundary, checks the cache, and runs file summaries concurrently. Directory summaries are generated from child file summaries.

Provider execution lives in `internal/provider.go`. Built-in providers shell out to `codex` or `claude`; custom providers can be configured in TOML. The built-in Codex provider isolates the subprocess from Codex user config and project rules so summary generation stays lightweight. Providers receive the summarization prompt on stdin unless `{prompt}` is used in args.

Rendering lives in `internal/render.go`. Managed writes live in `internal/write.go`.

## Flow

1. Parse CLI flags through Cobra.
2. Load and validate config.
3. Discover matching file or directory entries.
4. Read each file head, or collect child file summaries for directory entries.
5. Return cached summaries when possible.
6. Run the configured CLI provider for cache misses.
7. Render `list`, `json`, or `skill`.
8. Print to stdout or update a managed region with `--write`.

## Cache flow

The cache is rebuildable derived data. A cache record stores only the generated description. File keys include file content, path, provider, provider command, provider args, model, and prompt version. Directory keys use the directory rollup content and a directory prompt version.

Changing provider configuration intentionally invalidates old summaries, because different agents or flags may produce different descriptions.

## Current boundaries

`scout` reads the head of each file, not the full file, after `max_bytes`.

`scout` shells out to local CLI agents. It does not call hosted model APIs directly.

`scout` does not run a daemon or MCP server yet.
