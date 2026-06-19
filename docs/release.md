# Release

Releases are tag driven. A pushed semver tag builds `scout` with GoReleaser, creates a GitHub release, uploads checksummed archives, and publishes the Homebrew cask to `tomnagengast/homebrew-tap`.

## Release setup

The Homebrew tap lives at `tomnagengast/homebrew-tap`. Keep `HOMEBREW_TAP_GITHUB_TOKEN` set in the `tomnagengast/scout` repository secrets. The token needs content write access to that tap repository.

The release workflow uses the repository `GITHUB_TOKEN` for the GitHub release itself.

## Local checks

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

## Cut a release

Update `CHANGELOG.md`, merge the release commit to `main`, then create and push the next semver tag:

```sh
git tag -a v0.1.3 -m "v0.1.3"
git push origin v0.1.3
```

The `release` workflow only publishes from tag refs matching `v*.*.*`.

## Homebrew

GoReleaser publishes a Homebrew cask. Install the latest release with:

```sh
brew tap tomnagengast/tap
brew install --cask tomnagengast/tap/scout-cli
```
