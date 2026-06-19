# Output

`scout` emits a thin description index. Output is designed for agents and scripts first.

## List

`list` is the default format.

```sh
scout docs
```

```text
docs/config.md  Documents scout configuration precedence, provider commands, env vars, and cache behavior.
docs/output.md  Describes scout output formats and managed writes.
```

Use `list` when a human or agent will skim the index directly.

## JSON

```sh
scout docs --format json
```

```json
[
  {
    "type": "file",
    "path": "docs/config.md",
    "name": "config",
    "description": "Documents scout configuration precedence, provider commands, env vars, and cache behavior."
  }
]
```

Use `json` when another tool needs structured records.

The `type` field is either `file` or `dir`.

## Skill

```sh
scout skills --format skill
```

```text
skills/gh.md
---
name: gh
description: GitHub CLI for repos, PRs, CI checks, and settings.
---
```

Use `skill` when generating frontmatter-shaped descriptions for skill or capability documents.

## Managed Writes

For `list` and `json`, `--write` updates a managed block in the target file:

```sh
scout docs --write README.md
```

```md
<!-- scout:start -->
- docs/config.md  Documents scout configuration precedence, provider commands, env vars, and cache behavior.
<!-- scout:end -->
```

Repeated runs replace the same block instead of appending duplicates. If only one marker is present, `scout` exits with an error instead of guessing where to write.

For `skill`, `--write` expects exactly one input file and updates leading frontmatter:

```sh
scout docs/config.md --format skill --write docs/config.md
```

```md
---
name: config
description: Documents scout configuration precedence, provider commands, env vars, and cache behavior.
---
```
