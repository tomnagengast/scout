# Getting Started

This guide walks through a local source build and the first `scout` runs.

## Build

```sh
mise install
mise run build
./scout --help
```

## Run The First Index

Point `scout` at one or more files, directories, or globs:

```sh
./scout README.md
./scout docs
./scout "docs/**/*.md"
```

The default output is `list`: one path and one dense description per file.

```text
README.md  Describes scout, its install flow, CLI flags, output formats, configuration, and project goals.
```

Real summaries require a configured headless CLI provider. By default, `scout` uses `codex exec`. Use `--provider claude` to use the built-in Claude Code provider.

Summarize directories instead of files when you want a higher-level map:

```sh
./scout docs --type dir --max-depth 2
```

Directory summaries are based on child file summaries. Pass a directory path for `--type dir`; shell-expanded file globs such as `**/*.md` match files, not directories.

## Choose An Output Format

Use JSON when another tool or agent will parse the result:

```sh
./scout docs --format json
```

Use skill frontmatter when indexing capability documents:

```sh
./scout skills --format skill
```

See [output.md](./output.md) for exact output shapes.

## Write A Managed Index

Append or refresh a managed block in a file:

```sh
./scout docs --write README.md
```

`scout` replaces only the region between:

```md
<!-- scout:start -->
<!-- scout:end -->
```

For `--format skill`, `--write` updates the leading frontmatter for exactly one input file:

```sh
./scout docs/gh.md --format skill --write docs/gh.md
```

## Common Development Commands

```sh
mise run fmt
mise run test
mise run build
```

Build directly:

```sh
go build -o scout ./cmd
```
