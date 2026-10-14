# ADR-010: CI/CD

## Status

Accepted

Follows the Python SDK's [ADR-010](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/010-ci-cd.md), and the platform's [ADR-014: CI/CD](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/014-ci-cd.rst).

## Context

Every change should be tested, and every release built the same way. A Go module is published by pushing a semver tag: the Go module proxy fetches it from GitHub.

## Decision

GitHub Actions only, as on the platform, the cli and the other SDKs. `test.yml` runs `go vet`, golangci-lint and `go test -race` with coverage on every supported Go version (ADR-019) for every push and pull request, enforces 90% coverage, and uploads coverage to Codecov. Releases are versioned and tagged (`vX.Y.Z`) by semantic-release (ADR-015), which also creates the GitHub Release; the tag is the module's version. `build-release-github.yml`, the counterpart of the cli's workflow of the same name, then publishes it: it builds, vets and tests the tagged commit on Linux, macOS and Windows with every supported Go release, checks that `version.go` says the tag's version, asks the Go module proxy to fetch the tag (which records its checksum in sum.golang.org) and pkg.go.dev to index it, and imports the published version into a new module. It runs when a GitHub Release is published, and by hand to publish a tag again.

## Alternatives Considered

- Other CI services.
- GoReleaser, which the cli would use for binaries. A library has none.
- Waiting for the first `go get` of a new version to make the proxy fetch it. The release would be untested on macOS and Windows, and absent from pkg.go.dev until someone asked for it.

## Consequences

- **Positive:**
  - Consistent with the platform's and the other SDKs' workflows, and no manual publishing.
  - Every release is known to build on every supported OS and Go release, and to be importable from the proxy.
- **Negative:**
  - A bad tag is cached by the module proxy forever: it can be retracted (`retract` in `go.mod`), not deleted.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-009: Testing](009-testing.md)
- [ADR-015: Semantic Versioning and API Versions](015-versioning.md)
