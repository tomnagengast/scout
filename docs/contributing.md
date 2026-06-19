# Contributing

`scout` is intended to stay small, local-first, and easy to reason about.

## Development Loop

Install tools:

```sh
mise install
```

Run the standard checks:

```sh
mise run fmt
mise run test
mise run build
```

Release packaging is checked in CI with GoReleaser. If you touch `.goreleaser.yaml` or release workflows, also run:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

The direct Go equivalents are:

```sh
gofmt -w ./cmd ./internal
go test ./...
go build -o scout ./cmd
```

## Project Shape

The binary entrypoint lives in `cmd/main.go`.

The CLI command, config loading, discovery, provider execution, summarization, rendering, and managed writes live in `internal`.

Tests live beside the code they exercise inside `internal`.

## Documentation

Update `README.md` and the relevant file in `docs/` when changing commands, config fields, provider behavior, output formats, managed writes, caching, install steps, or project layout.

Docs should describe current behavior plainly. If something is a future direction, mark it as such or leave it out.

## Design Principles

Keep the index cheap to read and cheap to regenerate.

Prefer local CLI providers over direct hosted API integrations.

Keep output deterministic so generated blocks are safe in hooks and CI.

Keep managed writes narrow: only the managed block or leading skill frontmatter should change.

Avoid introducing long-lived state beyond the summary cache unless the feature needs it.
