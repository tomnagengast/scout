# Config

`scout` reads TOML config from a user config and an optional project config.

## Precedence

Configuration is applied in this order:

1. Built-in defaults
2. User config at `$XDG_CONFIG_HOME/scout.toml` or `~/.config/scout.toml`
3. Project `scout.toml` in the working directory or repo root
4. Environment variables
5. CLI flags

## Example

```toml
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

ignore = ["**/CHANGELOG.md", "**/vendor/**"]
```

## Fields

`provider` selects the summarizer provider. Built-in providers are `codex` and `claude`. Custom provider names are allowed when a matching `[providers.<name>]` table exists.

The built-in `codex` provider ignores Codex user config and project rules by default. This keeps Scout summaries from inheriting interactive-agent settings such as high reasoning effort, hooks, MCP servers, or custom sandbox defaults. Override `[providers.codex]` if you want a different invocation.

`model` is passed to the provider only when set. It expands through `{model_args}`.

`type` selects what `scout` emits. Use `file` for one entry per file, or `dir` for directory rollups based on child file summaries.

`max_depth` limits directory walking below each input directory. The default `0` means unlimited.

`concurrency` controls how many files are summarized at once. The default is `2` because each worker starts a local CLI agent process.

`max_bytes` limits how many bytes of each file are read before summarization. Large files are summarized from the head only.

`cache_dir` changes where summary cache records are stored. The default is `$XDG_CACHE_HOME/scout` or `~/.cache/scout`.

`ignore` adds scout-specific ignore globs on top of `.gitignore` and `.scoutignore`.

## Environment Variables

| Variable          | Description                              |
| ----------------- | ---------------------------------------- |
| `SCOUT_PROVIDER`  | Default summarizer provider.             |
| `SCOUT_MODEL`     | Model passed to the summarizer provider. |
| `SCOUT_CACHE_DIR` | Override the cache directory.            |

## Provider Args

Provider `args` are passed directly to the configured command after placeholder expansion.

| Placeholder    | Description                                           |
| -------------- | ----------------------------------------------------- |
| `{model_args}` | Expands to `<model_arg> <model>` when `model` is set. |
| `{output}`     | Path to a temp file that `scout` reads after exit.    |
| `{prompt}`     | Inserts the prompt as an argv value instead of stdin. |

If `{prompt}` is not used, `scout` writes the prompt to stdin. If `{output}` is not used, `scout` reads the provider response from stdout.

## Cache

Cache keys include the prompt version, provider, model, provider command, provider args, path, and summary input content. Use `--no-cache` to force fresh summaries.
