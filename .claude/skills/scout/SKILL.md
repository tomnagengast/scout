---
name: scout
description: "Use the scout CLI to build a cheap, thin description index of a document tree: one dense one-line summary per file, so you can survey what exists and load only the files you actually need. Reach for this skill whenever you are about to explore an unfamiliar repo or docs folder, want a token-cheap map of a tree before pulling files into context, need to generate or refresh SKILL.md / agentskills.io frontmatter across a directory, want a self-updating table of contents written into a README, or want a CI check that fails when the index drifts. Also use it whenever the user mentions scout, scout the docs, a description index, a context map, or progressive-disclosure-style reconnaissance over files."
---

# scout

`scout` walks a tree of documents and emits a thin, machine-readable description
layer: one tight summary per file. It is reconnaissance for your context window:
read the cheap map first (a few hundred tokens), decide what's relevant, then load
only those files. Scout ahead, then load deliberately.

It generates each summary by shelling out to a headless local CLI agent
(`codex exec` by default, or `claude -p` with `--provider claude`) and caches
results, so repeated runs are fast and nearly free.

## When to use this skill

Use scout instead of reading a whole tree when the goal is to *decide what to
read*. The point is to avoid dumping 200KB of docs into context to answer a
question that needed one file. Good moments:

- **Pre-flight before a task**: survey an unfamiliar repo or `docs/` folder and
  load only files whose descriptions match the goal.
- **Generate/refresh skill frontmatter**: `--format skill` emits SKILL.md-style
  blocks across a directory of capabilities.
- **Self-updating README index**: `--write` folds a managed table of contents
  into a file, idempotently.
- **CI freshness check**: run `--write` and fail if the managed block changed.

If you instead need the *full contents* of a known file, just read it directly:
scout is the layer that decides *whether* you need to.

## Prerequisites

Scout is a Go CLI. Confirm it's available (`scout --version`); if not, install via
Homebrew (the recommended path):

```sh
brew tap tomnagengast/tap
brew install --cask scout
```

To build from source instead (e.g. for development):

```sh
mise install && mise run build   # produces ./scout
# or
go build -o scout ./cmd
```

Real summaries require a configured headless provider. By default scout uses
`codex exec` (run `codex login` first); pass `--provider claude` to use Claude
Code instead. Without a working provider scout still discovers files but cannot
summarize them.

## Core usage

```
scout [paths...] [flags]
```

`paths` accepts files, directories, and globs. Directories are walked recursively
unless bounded by `--max-depth`. `.gitignore` is respected by default; add a
`.scoutignore` for scout-specific exclusions.

The most common first move is a JSON map you can parse, or a list you skim:

```sh
scout docs --format json     # structured; prefer this when an agent consumes it
scout docs                   # list (default); one path + description per line
scout "docs/**/*.md"         # globs work; quote them so the shell doesn't expand early
```

Read the resulting index, pick the entries whose descriptions match your goal,
and only then open those specific files.

## Flags worth knowing

| Flag | Why it matters |
| ---- | -------------- |
| `-f, --format <list\|skill\|json>` | `json` for programmatic use, `skill` for frontmatter, `list` for skimming. |
| `-w, --write <file>` | Fold the index into a managed region of a file (idempotent). |
| `--type <file\|dir>` | `dir` summarizes directories from their child file summaries, giving a higher-level map. |
| `--max-depth <n>` | Bound directory walking; `0` = unlimited. |
| `--provider <name>` | `codex` (default), `claude`, or a configured provider. Overrides config for one run. |
| `-m, --model <model>` | Model passed to the provider. |
| `-c, --concurrency <n>` | Files summarized in parallel (default 2; each worker starts a CLI agent process). |
| `--max-bytes <n>` | Bytes read per file before truncation (default 16384). Scout summarizes the head. |
| `--no-cache` | Re-summarize everything, bypassing the cache. |
| `--quiet` | Suppress stderr progress. |

## Output formats

- **`list`** (default): `path  description`, one per line. Compact and greppable;
  drop it straight into context.
- **`json`**: array of `{type, path, name, description}` records where `type` is
  `file` or `dir`. Prefer this when scout is a tool feeding another program/agent.
- **`skill`**: SKILL.md / agentskills.io frontmatter blocks (`name` +
  `description`), for seeding or refreshing a skills/capability index.

## Higher-level maps with `--type dir`

To get a coarser overview, summarize directories from their children. Pass a
directory path (not a file glob; `**/*.md` matches files, not directories):

```sh
scout docs --type dir --max-depth 2
```

## Writing a managed index with `--write`

`--write` updates a file in place instead of printing. Writes are idempotent:
scout maintains a managed region so re-runs refresh the index rather than
appending duplicates, so it is safe for pre-commit hooks and CI.

For `list`/`json`, scout **appends** a managed block at the bottom:

```sh
scout docs --write README.md
```

```md
<!-- scout:start -->
- docs/gh.md  GitHub CLI for repos, PRs, CI checks, and settings.
<!-- scout:end -->
```

For `--format skill`, scout **prepends/refreshes leading frontmatter** (valid only
at the top of a file) and expects exactly one input file:

```sh
scout docs/gh.md --format skill --write docs/gh.md
```

If only one of the two markers is present, scout errors out instead of guessing
where to write. Fix the markers rather than retrying.

## Caching

Summaries are cached, keyed on file content + path + provider + provider
command/args + model + prompt version. Unchanged files are never re-summarized, so
re-runs are cheap (this is what makes scout viable in CI). Changing provider or
model intentionally invalidates old summaries. Use `--no-cache` to force fresh
ones. Cache lives at `$XDG_CACHE_HOME/scout` (override with `--cache-dir` or
`SCOUT_CACHE_DIR`).

## Configuration (only if defaults aren't enough)

Precedence, lowest to highest: built-in defaults → user config
(`$XDG_CONFIG_HOME/scout.toml` or `~/.config/scout.toml`) → project `scout.toml`
in the working dir or repo root → environment variables (`SCOUT_PROVIDER`,
`SCOUT_MODEL`, `SCOUT_CACHE_DIR`) → CLI flags.

You only need a `scout.toml` to point at a provider wrapper, change provider
flags, define a new provider name, or add `ignore` globs on top of
`.gitignore`/`.scoutignore`. For a full provider table example and the arg
placeholders (`{model_args}`, `{output}`, `{prompt}`), see the project's
`docs/config.md`.

## Workflow recipes

**Agent pre-flight**: map first, then load deliberately:

```sh
scout docs --format json --quiet
# read the map, choose files whose descriptions match the goal, open only those
```

**Self-describing skill tree**: keep frontmatter honest as files change:

```sh
scout skills/ --format skill
```

**Repo onboarding**: a navigable, self-updating README TOC:

```sh
scout docs --write README.md
```

**CI freshness check**: fail when the index drifts:

```sh
scout docs --write README.md
git diff --exit-code README.md   # nonzero if the managed block changed
```

## Tips

- Quote globs (`"docs/**/*.md"`) so scout, not the shell, expands them.
- Keep concurrency modest; each worker spawns a local CLI agent process.
- If summaries look wrong after switching providers/models, that's expected; the
  cache key changed; the next run regenerates them.
- For a one-shot "pack the whole repo and answer this" need, scout is the wrong
  tool. Reach for repomix/gitingest. Scout decides *whether* you need them.
