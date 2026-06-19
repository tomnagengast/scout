# Release

Releases are tag driven. A pushed semver tag builds `scout` with GoReleaser, creates a GitHub release, uploads checksummed archives, and publishes the Homebrew cask to `tomnagengast/homebrew-tap`.

## Required GitHub Secret

Create `tomnagengast/homebrew-tap` before publishing the first release. Set `HOMEBREW_TAP_GITHUB_TOKEN` in the `tomnagengast/scout` repository secrets. The token needs content write access to that tap repository.

The release workflow uses the repository `GITHUB_TOKEN` for the GitHub release itself.

## Local Checks

Run the standard checks:

```sh
go test ./...
go build -o scout ./cmd
```

Verify release packaging without publishing:

```sh
goreleaser check
goreleaser release --snapshot --clean
```

## Cut a Release

Create and push a semver tag:

```sh
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

The `release` workflow only publishes from tag refs matching `v*.*.*`.

## Homebrew

GoReleaser publishes a Homebrew cask. Install the latest release with:

```sh
brew install --cask tomnagengast/tap/scout
```
