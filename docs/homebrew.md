## Updating the Homebrew package

Homebrew package is automatically updated by GitHub Actions on turso-cli repository.
Those GitHub Actions are triggered by a new tag being pushed to the repository.

For example:

```console
git tag v0.1.3
git push --tags
```

## Setup details

Turso Homebrew Tap is stored in [homebrew-tap](https://github.com/tursodatabase/homebrew-tap) repository.

There's a `ACCESS_TOKEN_TO_TAP` GitHub personal access token that has read/write access to Content and Actions homebrew-tap repositories.
It will expire on Jul 15 2024.
It is used by GitHub Actions in turso-cli repository to give them access to both turso-cli and homebrew-tap repositories.

[GoReleaser](https://github.com/goreleaser/goreleaser) is used to package everything up.
[GoReleaser GitHub Actions](https://github.com/goreleaser/goreleaser-action) are used for CI.

To install run:
```console
brew install tursodatabase/tap/turso
```
