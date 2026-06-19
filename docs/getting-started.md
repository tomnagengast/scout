# Getting started

New to `scout`? This walks you from installation to your first few indexes. It takes a couple of minutes, and you don't need to configure anything to start.

## Install with Homebrew

```sh
brew tap tomnagengast/tap
brew install --cask tomnagengast/tap/scout-cli
scout --help
```

## Build from source

```sh
mise install
mise run build
mise run install-local
scout --help
```

## Run the first index

Point `scout` at one or more files, directories, or globs:

```sh
scout README.md
scout docs
scout "docs/**/*.md"
```

The default output is `list`: one path and one dense description per file. A good run looks like this:

```text
README.md  Describes scout, its install flow, CLI flags, output formats, configuration, and project goals.
```

That description is generated for you: `scout` hands each file to a headless CLI provider and asks for one tight sentence. By default it uses `codex exec` (run `codex login` once); pass `--provider claude` to use Claude Code instead. Without a working provider, scout still lists your files but leaves the descriptions blank, which is the most common first-run surprise (see [Troubleshooting](#troubleshooting)).

Summarize directories instead of files when you want a higher-level map:

```sh
scout docs --type dir --max-depth 2
```

Directory summaries are based on child file summaries. Pass a directory path for `--type dir`; shell-expanded file globs such as `**/*.md` match files, not directories.

## Choose an output format

Use JSON when another tool or agent will parse the result:

```sh
scout docs --format json
```

Use skill frontmatter when indexing capability documents:

```sh
scout skills --format skill
```

See [output.md](./output.md) for exact output shapes.

## Write a managed index

Append or refresh a managed block in a file:

```sh
scout docs --write README.md
```

`scout` replaces only the region between:

```md
<!-- scout:start -->
<!-- scout:end -->
```

For `--format skill`, `--write` updates the leading frontmatter for exactly one input file:

```sh
scout docs/gh.md --format skill --write docs/gh.md
```

## Troubleshooting

A few things that trip people up on the first run:

**Descriptions come back blank.** The provider isn't set up. Scout discovers files fine without one, but it can't summarize them. Run `codex login`, or switch with `--provider claude` if you use Claude Code.

**`command not found: scout`.** A source build leaves the binary in the repo until you install it locally. Run `mise run install-local` to put it on your `PATH`.

**A glob matched nothing, or way too much.** Quote globs so scout expands them instead of your shell: `scout "docs/**/*.md"`.

**`--type dir` summarized files, not directories.** Pass a directory path. A shell-expanded `**/*.md` matches files; use `scout docs --type dir` instead.

**Summaries look stale after switching provider or model.** That's expected. The cache key includes the provider and model, so changing either invalidates old entries. Re-run, or force fresh output with `--no-cache`.

**Files are missing from the index.** Check that they aren't excluded by `.gitignore`, `.scoutignore`, or an `ignore` glob in `scout.toml`.

## Common development commands

```sh
mise run fmt
mise run test
mise run build
```

Build directly:

```sh
go build -o scout ./cmd
```
