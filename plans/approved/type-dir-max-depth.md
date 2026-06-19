# Plan: `--type dir --max-depth`
> updated: 2026-06-18T17:32:27-07:00
> status: shipped - `--type` and `--max-depth` are in the CLI; kept as a decision record.

## Research

Pre-implementation, the flow was file-only:

- `internal/command.go`: Cobra binds config-backed flags.
- `internal/config.go`: defaults/config/env/changed flags merge, validates format/concurrency/max-bytes.
- `internal/discover.go`: `discoverFiles` resolves paths/globs, walks dirs recursively, ignores `.git/`, `.gitignore`, `.scoutignore`, config ignore.
- `internal/run.go`: defaults paths to `.`, discovers files, summarizes files, renders entries, writes optional managed block.
- `internal/summarize.go`: `Summarizer` accepts `(path, content, truncated)`, cache key includes path/content/provider/model/prompt version.
- `internal/types.go`: `Entry` has only `path`, `name`, `description`; no entry kind.
- `docs/architecture.md`: at the time, explicitly said directory rollups were roadmap.
- `README.md`: roadmap then said `--depth`; docs needed updating to `--type dir --max-depth`.

## Decisions

- Add `--type <file|dir>`; default `file`.
- Add `--max-depth <n>`; default `0` means unlimited.
- `--type file`: existing behavior, except directory walking stops after `--max-depth` relative to each input directory.
- `--type dir`: emit directory rollups only. Directory entries are selected during discovery, then summarized from child file summaries.
- Directory depth counts below each input directory: `docs --type dir --max-depth 1` can emit `docs`, `docs/foo`, but not `docs/foo/bar`.
- For glob inputs, depth is relative to each matched directory. Matched files do not produce dir entries for `--type dir`.
- Keep `--format skill --write` restricted to exactly one entry. Do not special-case directory entries.
- JSON should include `"type"` so consumers can distinguish future mixed outputs.

## Implementation

1. Extend config/CLI:
   - Add `Type string` and `MaxDepth int` to `Config`.
   - Bind `--type` and `--max-depth`.
   - Include both in changed flag handling.
   - Validate `type in {"file","dir"}` and `max-depth >= 0`.

2. Replace file-only discovery internals:
   - Introduce a small discovered target type, probably `{Path string, Type string}`.
   - Keep `discoverFiles` as a compatibility wrapper if it simplifies existing tests.
   - Add `discoverTargets(paths, cfg)` or `discoverPaths(paths, ignore, type, maxDepth)`.
   - Preserve ignore handling and stable ordering.
   - For `file`, include files only.
   - For `dir`, include directories only, including the input directory itself when not ignored.

3. Add directory rollup summarization:
   - Keep `summarizeFiles` path for `--type file`.
   - Add `summarizeDirs(ctx, dirs, cfg, summarizer, stderr)`.
   - For each dir, discover contained files with no extra depth limit for content unless implementation proves too expensive.
   - Summarize/cache child files first using existing file summary cache.
   - Build rollup content from child summaries, not raw file contents:
     `path<TAB>description` lines sorted by path.
   - Add a directory-specific prompt/cache key path, with prompt version bump or separate `dirPromptVersion`.
   - Directory prompt asks for one sentence describing what the directory covers and its boundaries.

4. Extend entry model/rendering:
   - Add `Type string json:"type"` to `Entry`.
   - For file entries set `Type: "file"`; dir entries set `Type: "dir"`.
   - Keep list/skill output visually unchanged except they display directory paths normally.
   - Accept JSON shape change because current CLI is alpha and this adds needed structure.

5. Update docs:
   - README flags and usage.
   - README roadmap item from `--depth` to implemented/current behavior.
   - `docs/getting-started.md` examples for directory rollups.
   - `docs/config.md` fields for `type` and `max_depth`.
   - `docs/output.md` JSON `type` field.
   - `docs/architecture.md` remove file-only boundary and describe directory rollup path.

6. Tests:
   - Config parses/validates `--type` and `--max-depth`.
   - Discovery respects max depth for files.
   - Discovery emits directories for `--type dir`, with stable order and ignores.
   - Directory rollup uses child summaries, not raw concatenated file content.
   - JSON includes `type`.
   - End-to-end fake provider smoke for `scout docs --type dir --max-depth 1 --quiet --no-cache`.

7. Verify:
   - `gofmt -w ./cmd ./internal`
   - `go test ./...`
   - `go build -o scout ./cmd`
   - `mise run test`
   - `mise run build`
   - Smoke: fake provider file mode still works.
   - Smoke: fake provider dir mode emits expected directory path.
   - Remove local `./scout` build artifact.

## Unresolved questions

- Should `--type dir` include the root input directory itself? Recommendation: yes.
- Should `--max-depth 0` mean unlimited or root-only? Recommendation: unlimited, matching common CLI convention for unset numeric config.
- Should directory rollups include nested child directory summaries when available, or only file summaries? Recommendation: first implementation uses file summaries only for simpler deterministic behavior.
