# Install

`scout` is distributed as a single Go binary. The repository uses `mise` to pin the Go toolchain and expose common development tasks.

## System requirements

| Requirement           | Notes                                                                   |
| --------------------- | ----------------------------------------------------------------------- |
| Go                    | Managed by `mise`; see `.mise.toml` for the pinned version.             |
| macOS or Linux        | The current build is pure Go.                                           |
| A summarizer CLI      | Required for live summaries. The built-in default provider uses Codex.  |
| Codex or Claude Code  | Optional individually; install whichever provider you configure.        |

## Homebrew

```sh
brew tap tomnagengast/tap
brew install --cask tomnagengast/tap/scout-cli
```

If you installed the pre-v0.1.2 cask token, migrate once:

```sh
brew uninstall --cask tomnagengast/tap/scout
brew install --cask tomnagengast/tap/scout-cli
```

## Build from source

Install project tools and build the binary:

```sh
mise install
mise run build
```

The binary is written to `./scout`.

Verify the build:

```sh
./scout --version
```

The same build can be run without `mise`:

```sh
go build -o scout ./cmd
```

`mise run build` stamps the binary with `git describe --tags --always --dirty`, the short commit, and the UTC build time. Set `SCOUT_VERSION` to override the displayed version for a release build.

Tagged releases are built by GoReleaser and published as GitHub release archives plus a Homebrew cask. See [release.md](./release.md) for the release workflow.

## Install locally

Install to `~/.local/bin/scout`:

```sh
mise run install-local
```

Make sure `~/.local/bin` is on your `PATH`, then verify:

```sh
scout --version
```

## Summarizer setup

The default provider shells out to `codex exec`. Authenticate Codex before running a real summary:

```sh
codex login
```

To use Claude Code instead, install and authenticate Claude Code, then run with:

```sh
scout --provider claude <paths...>
```

See [config.md](./config.md) for provider command configuration.
