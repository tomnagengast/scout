# Changelog

All notable changes to `scout` are tracked here.

## v0.1.3 - 2026-06-19

- Deepens summary generation so file and directory summaries share one cache/provider lifecycle.
- Deepens discovery around an explicit request/session shape with root-local ignore handling.
- Deepens output emission so write destinations own their destination-specific rendering.
- Reduces repeated configuration flag metadata while preserving config precedence.

## v0.1.2 - 2026-06-19

- Renames the Homebrew cask token to `scout-cli` to avoid a collision with Homebrew's disabled `scout` cask.

## v0.1.1 - 2026-06-19

- Updates `github.com/BurntSushi/toml` from `1.5.0` to `1.6.0`.
- Updates `github.com/bmatcuk/doublestar/v4` from `4.9.1` to `4.10.0`.
- Adds a Claude Code skill for using `scout`.
- Corrects Homebrew cask install documentation.

## v0.1.0 - 2026-06-19

- Initial public release.
- Adds document discovery, file and directory summaries, summary caching, multiple output formats, managed writes, provider configuration, and release automation.
