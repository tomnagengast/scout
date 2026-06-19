# scout

**Reconnaissance for your context window.** `scout` walks a tree of documents and emits a thin, machine-readable description layer, one tight summary per file, so an agent can survey what's there and load only what it needs.

[![Status](https://img.shields.io/badge/status-alpha-orange.svg)](#roadmap)
[![CI](https://github.com/tomnagengast/scout/actions/workflows/ci.yml/badge.svg)](https://github.com/tomnagengast/scout/actions/workflows/ci.yml)

---

## What it does

Point `scout` at a path or glob. For each matched document it writes a one-line description of what the file is for and where its boundaries are, then prints an index. The output is meant to be read by an agent, or written into a file an agent reads.

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
brew tap tomnagengast/tap
brew install --cask tomnagengast/tap/scout-cli
```

<details>
<summary><strong>From source</strong></summary>

```sh
git clone https://github.com/tomnagengast/scout
cd scout
mise install
mise run build
```

</details>

Scout generates summaries by shelling out to a headless CLI agent already installed on your machine. By default it uses `codex exec`; `claude -p` is also supported when Claude Code is installed.

```sh
codex login   # or configure Claude Code if you use --provider claude
```

## Usage

```
scout [paths...] [flags]

Flags:
  -f, --format <fmt>      Output format: list (default), skill, json
  -w, --write <file>      Write the index into a file (idempotent; see "Writing to files")
      --type <type>       Entry type to summarize: file (default) or dir
      --max-depth <n>     Maximum directory depth to walk, 0 for unlimited
      --provider <name>   Summarizer provider: codex, claude, or a configured provider
  -m, --model <model>     Model passed to the summarizer provider (default: provider default)
  -c, --concurrency <n>   Files summarized in parallel (default: 2)
      --max-bytes <n>     Max bytes read per file before truncation (default: 16384)
      --no-cache          Bypass the summary cache and re-summarize everything
      --cache-dir <path>  Cache location (default: $XDG_CACHE_HOME/scout)
      --quiet             Suppress progress output on stderr
  -v, --version           Print version information
```

`paths` accepts files, directories, and globs. Directories are walked recursively unless bounded with `--max-depth`. `.gitignore` is respected by default, and you can add a `.scoutignore` for scout-specific exclusions.

By default `scout` summarizes files. Use `--type dir` to summarize directories from their child file summaries:

```sh
scout docs --type dir --max-depth 2
```

### Output formats

**`list`** (default) - one line per file: relative path + description. Compact, greppable, ideal for dropping straight into an agent's context.

**`skill`** - emits standard frontmatter blocks in the `SKILL.md` / agentskills.io convention, so the output can seed a skills index or a capability manifest:

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

**`json`** - structured output for programmatic consumers. Agents that call `scout` as a tool should prefer this:

```sh
scout docs --format json
```

```json
[
  { "type": "file", "path": "docs/qmd.md", "name": "qmd", "description": "Search and navigate indexed markdown collections via the qmd CLI." },
  { "type": "file", "path": "docs/gh.md",  "name": "gh",  "description": "GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects." }
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
- docs/gh.md  GitHub CLI for repos, PRs, CI checks, and settings. Does NOT cover issues or projects.
- docs/qmd.md  Search and navigate indexed markdown collections via the qmd CLI.
<!-- scout:end -->
```

With `--format skill`, scout **prepends** instead, refreshing the leading frontmatter block, because frontmatter is only valid at the very top of a file:

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

## Why scout

Most "feed my repo to an LLM" tools (`repomix`, `gitingest`, `code2prompt`) pack everything into one giant blob and hand it to the model. That's fine for a one-shot question, but it's the wrong primitive for an agent that runs in a loop on a token budget. Dumping 200KB of docs into context to answer a question that needed one file is slow, expensive, and noisy.

`scout` does the opposite. It produces a cheap map you read first: a list of files, each with a one-line description of what it is and what it covers. The agent reads the map (a few hundred tokens), decides which files are relevant, and pulls only those into the window. This is progressive disclosure, the same pattern `SKILL.md` frontmatter and [agentskills.io](https://agentskills.io) use to keep capability descriptions in front of an agent without loading every detail.

Scout ahead. Then load deliberately.

## How it works

1. **Discover** - resolve paths/globs, walk directories, apply `.gitignore` and `.scoutignore`, and select file or directory entries.
2. **Read** - load each file up to `--max-bytes` (large files are truncated at a token-safe boundary; scout summarizes the head, which carries the intent for most docs).
3. **Summarize** - send each file to a headless local CLI agent with a prompt tuned to produce a single dense description in the agentskills.io style; directory entries are summarized from child file summaries.
4. **Cache** - keyed on a hash of file content + provider + model + provider command + prompt version. Unchanged files are never re-summarized, so re-runs are fast and nearly free. This is what makes scout cheap enough to wire into CI.
5. **Emit** - render in the requested format, optionally into a managed block in a target file.

Summaries are generated concurrently (`--concurrency`), and output ordering is stable regardless of completion order, so diffs stay clean. The default concurrency is intentionally conservative because each worker starts a local CLI agent process.

## Configuration

Flags win over environment variables, which win over a project `scout.toml` in the working directory or repo root, which wins over the user config at `$XDG_CONFIG_HOME/scout.toml` or `~/.config/scout.toml`.

```toml
# ~/.config/scout.toml
provider    = "codex"
type        = "file"
max_depth   = 0
concurrency = 2
max_bytes   = 16384

[providers.codex]
command = "codex"
args = [
  "exec",
  "--ephemeral",
  "--skip-git-repo-check",
  "--ignore-user-config",
  "--ignore-rules",
  "--sandbox", "read-only",
  "--color", "never",
  "-c", "model_reasoning_effort=\"none\"",
  "--output-last-message", "{output}",
  "{model_args}",
  "-"
]
model_arg = "--model"

[providers.claude]
command = "claude"
args = [
  "-p",
  "--output-format", "text",
  "--permission-mode", "plan",
  "--max-turns", "1",
  "{model_args}",
  "{prompt}"
]
model_arg = "--model"

# extra ignore globs, layered on top of .gitignore / .scoutignore
ignore = ["**/CHANGELOG.md", "**/vendor/**"]
```

The built-in `codex` provider intentionally ignores Codex user config and project rules so summaries do not inherit interactive-agent settings like high reasoning effort, MCP servers, hooks, or custom sandbox defaults. Add a `[providers.codex]` block if you want Scout to inherit or customize that behavior. The built-in `codex` and `claude` providers use those command shapes automatically; you only need to add provider blocks when you want to point at a wrapper, change flags, or define a new provider name. `--provider claude` overrides the configured provider for one run.

Provider arg placeholders:

| Placeholder    | Description                                        |
| -------------- | -------------------------------------------------- |
| `{model_args}` | Expands to `<model_arg> <model>` when a model is set. |
| `{output}`     | Path to a temp file scout reads after the CLI exits. |
| `{prompt}`     | Inserts the prompt as an argv value instead of stdin. |

| Variable          | Description                                  |
| ----------------- | -------------------------------------------- |
| `SCOUT_PROVIDER`  | Default summarizer provider.                 |
| `SCOUT_MODEL`     | Model passed to the summarizer provider.     |
| `SCOUT_CACHE_DIR` | Override the cache directory.                |

## Context-engineering patterns

**Agent pre-flight.** Have an agent run `scout docs --format json` at the start of a task and load only the files whose descriptions match the goal. The map costs a few hundred tokens; the savings are the difference between loading one file and loading the whole tree.

**Self-describing skill trees.** Run `scout skills/ --format skill` to generate or refresh frontmatter across a directory of capabilities, keeping a skills index honest as files change.

**Repo onboarding.** `scout docs --write README.md` gives a human (or a new agent) a navigable table of contents that updates itself.

**CI freshness check.** Run `scout … --write` in CI and fail if the managed block changed, the same idempotency guarantee that keeps `go generate`-style artifacts in sync.

## Docs

| Page | Purpose |
| ---- | ------- |
| [Docs index](./docs/README.md) | Where to start and how the guides fit together. |
| [Install](./docs/install.md) | Build the binary, install locally, and verify the CLI. |
| [Getting started](./docs/getting-started.md) | Run the first index, choose formats, write managed blocks, troubleshoot. |
| [Config](./docs/config.md) | Precedence, provider config, env vars, and caching. |
| [Output](./docs/output.md) | `list`, `json`, `skill`, and idempotent `--write`. |
| [Architecture](./docs/architecture.md) | How discovery, summarization, caching, and writes flow. |
| [Release](./docs/release.md) | Cut tagged releases and publish the Homebrew cask. |
| [Contributing](./docs/contributing.md) | Run checks and work within repo conventions. |

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

- [ ] Additional built-in CLI providers (OpenAI Codex variants, local via Ollama)
- [ ] Incremental `--watch` mode
- [ ] Custom prompt/description templates per project
- [ ] MCP server mode so agents can call `scout` directly as a tool

## Contributing

Issues and PRs welcome. Please run the test suite and update the relevant docs before submitting.
