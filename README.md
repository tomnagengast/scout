# scout

**Reconnaissance for your context window.** `scout` walks a tree of documents and emits a thin, machine-readable description layer — one tight summary per file — so an agent can survey what's there and load only what it needs.

[![Status](https://img.shields.io/badge/status-alpha-orange.svg)](#roadmap)

---

## Why scout

Most "feed my repo to an LLM" tools — `repomix`, `gitingest`, `code2prompt` — pack everything into one giant blob and hand it to the model. That's fine for a one-shot question, but it's the wrong primitive for an agent that runs in a loop on a token budget. Dumping 200KB of docs into context to answer a question that needed one file is slow, expensive, and noisy.

`scout` does the opposite. It produces the *cheap map you read first*: a list of files, each with a one-line description of what it is and what it covers. The agent reads the map (a few hundred tokens), decides which files are actually relevant, and pulls only those into the window. This is **progressive disclosure** — the same pattern that `SKILL.md` frontmatter and [agentskills.io](https://agentskills.io) use to keep capability descriptions in front of an agent without loading every implementation detail.

Scout ahead. Then load deliberately.

## What it does

Point `scout` at a path or glob. For each matched document it generates an action-oriented description — what the file is for, and where its boundaries are — and prints an index. Output is designed to be consumed by an agent (or piped into a file an agent reads), not admired by a human.

```sh
cd path/to/repo/
tree
# .
# ├── README.md
# └── docs
#     ├── resources
#     │   ├── foo.md
#     │   └── bar.md
#     ├── gh.md
#     └── qmd.md

scout docs/**
# docs/resources/foo.md  Defines the canonical event schema and field-level semantics for the ingestion pipeline.
# docs/resources/bar.md  Reference for environment variables and secret resolution order across local, CI, and prod.
# docs/gh.md             GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects.
# docs/qmd.md            Search and navigate indexed markdown collections via the qmd CLI (notes, knowledge bases, retrieval).
```

## Install

**Homebrew**

```sh
brew install tomnagengast/tap/scout
```

**Go**

```sh
go install github.com/tomnagengast/scout@latest
```

**From source**

```sh
git clone https://github.com/tomnagengast/scout
cd scout
just build   # or: go build -o scout .
```

Scout generates summaries with an LLM, so it needs an API key:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
```

## Usage

```
scout [paths...] [flags]

Flags:
  -f, --format <fmt>      Output format: list (default), skill, json
  -w, --write <file>      Write the index into a file (idempotent; see "Writing to files")
  -m, --model <model>     Model used for summaries (default: $SCOUT_MODEL or a small, fast model)
  -c, --concurrency <n>   Files summarized in parallel (default: 8)
      --max-bytes <n>     Max bytes read per file before truncation (default: 16384)
      --no-cache          Bypass the summary cache and re-summarize everything
      --cache-dir <path>  Cache location (default: $XDG_CACHE_HOME/scout)
      --quiet             Suppress progress output on stderr
```

`paths` accepts files, directories, and globs. Directories are walked recursively. `.gitignore` is respected by default, and you can add a `.scoutignore` for scout-specific exclusions.

### Output formats

**`list`** (default) — one line per file: relative path + description. Compact, greppable, ideal for dropping straight into an agent's context.

**`skill`** — emits standard frontmatter blocks in the `SKILL.md` / agentskills.io convention, so the output can seed a skills index or a capability manifest:

```sh
scout docs --format skill
```

```
docs/qmd.md
---
name: qmd
description: Search and navigate indexed markdown collections via the qmd CLI (notes, docs, knowledge bases, document retrieval).
---

docs/gh.md
---
name: gh
description: GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects.
---
```

**`json`** — structured output for programmatic consumers. Agents that call `scout` as a tool should prefer this:

```sh
scout docs --format json
```

```json
[
  { "path": "docs/qmd.md", "name": "qmd", "description": "Search and navigate indexed markdown collections via the qmd CLI." },
  { "path": "docs/gh.md",  "name": "gh",  "description": "GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects." }
]
```

### Writing to files

`--write` folds the index into an existing file instead of printing it. Writes are **idempotent**: scout maintains a managed region so repeated runs refresh the index in place rather than appending duplicates.

```sh
scout docs --write README.md
```

By default scout **appends** the index inside a managed block at the bottom of the file:

```md
<!-- scout:start -->
- docs/gh.md  — GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects.
- docs/qmd.md — Search and navigate indexed markdown collections via the qmd CLI.
<!-- scout:end -->
```

With `--format skill`, scout **prepends** instead, refreshing the leading frontmatter block — because frontmatter is only valid at the very top of a file:

```sh
scout docs/gh.md --format skill --write docs/gh.md
```

```md
---
name: gh
description: GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects.
---

# ...existing file body, untouched...
```

This makes `scout --write` safe to run in a pre-commit hook or CI job: the managed region (or frontmatter) is the only thing that changes.

## How it works

1. **Discover** — resolve paths/globs, walk directories, apply `.gitignore` and `.scoutignore`.
2. **Read** — load each file up to `--max-bytes` (large files are truncated at a token-safe boundary; scout summarizes the head, which carries the intent for most docs).
3. **Summarize** — send each file to the model with a prompt tuned to produce a single dense description in the agentskills.io style: what the file is for, and explicit boundaries (the "Does NOT cover…" pattern) so an agent doesn't over-trust a file's scope.
4. **Cache** — keyed on a hash of file content + model + prompt version. Unchanged files are never re-summarized, so re-runs are fast and nearly free. This is what makes scout cheap enough to wire into CI.
5. **Emit** — render in the requested format, optionally into a managed block in a target file.

Summaries are generated concurrently (`--concurrency`), and output ordering is stable regardless of completion order, so diffs stay clean.

## Configuration

Flags win over environment variables, which win over a `scout.toml` in the working directory or repo root.

```toml
# scout.toml
model       = "claude-haiku-4-5"
concurrency = 8
max_bytes   = 16384

# extra ignore globs, layered on top of .gitignore / .scoutignore
ignore = ["**/CHANGELOG.md", "**/vendor/**"]
```

| Variable          | Description                                  |
| ----------------- | -------------------------------------------- |
| `ANTHROPIC_API_KEY` | API key for summary generation.            |
| `SCOUT_MODEL`     | Default model.                               |
| `SCOUT_CACHE_DIR` | Override the cache directory.                |

## Context-engineering patterns

**Agent pre-flight.** Have an agent run `scout docs --format json` at the start of a task and load only the files whose descriptions match the goal. The map costs a few hundred tokens; the savings are the difference between loading one file and loading the whole tree.

**Self-describing skill trees.** Run `scout skills/ --format skill` to generate or refresh frontmatter across a directory of capabilities, keeping a skills index honest as files change.

**Repo onboarding.** `scout docs --write README.md` gives a human (or a new agent) a navigable table of contents that updates itself.

**CI freshness check.** Run `scout … --write` in CI and fail if the managed block changed — the same idempotency guarantee that keeps `go generate`-style artifacts in sync.

## Compared to

| Tool | Output | Token cost | Best for |
| ---- | ------ | ---------- | -------- |
| **scout** | A thin description index | Tiny | Letting an agent decide what to load, in a loop |
| `repomix`, `gitingest`, `code2prompt` | The whole repo packed into one blob | Large | One-shot "here's everything, answer this" |
| RAG / vector search | Chunks ranked by similarity | Medium | Retrieval over corpora too big to index by hand |

Scout is complementary to all of these. It's the layer that decides *whether* you even need to reach for the others.

## Design principles

- **Cheap by default.** Caching and head-truncation keep repeated runs close to free.
- **Idempotent writes.** Managed regions mean `--write` is safe in hooks and CI.
- **Deterministic output.** Stable ordering, clean diffs.
- **Agent-first.** Output is structured for machines; `json` and `skill` are first-class, not afterthoughts.
- **No lock-in.** Plain text and standard frontmatter out; nothing proprietary to parse.

## Roadmap

- [ ] Pluggable model backends (OpenAI, local via Ollama)
- [ ] `--depth` to summarize directories as well as files (rolled-up tree descriptions)
- [ ] Incremental `--watch` mode
- [ ] Custom prompt/description templates per project
- [ ] MCP server mode so agents can call `scout` directly as a tool

## Contributing

Issues and PRs welcome. Please run the test suite and `scout --write` before submitting so the index stays current.
