# ADR-015: Semantic Versioning and API Versions

## Status

Accepted

Follows the Python SDK's [ADR-015](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/015-versioning.md), which adapts the platform's [ADR-020: Semantic Versioning](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/020-semantic-versioning.rst) and [ADR-023: API Versioning](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/023-api-versioning.rst).

## Context

The platform's api version is `smarter.sh/v1`, and its releases are versioned by semantic-release. A Go module's version is its git tag, and Go's import compatibility rule requires a new module path (`.../v2`) for every major version from 2 on.

## Decision

The SDK is versioned by semantic-release from conventional commits, independently of the platform and of the other SDKs; its tags (`vX.Y.Z`) are the module's versions, and `smarter.Version`, which the release writes into `version.go`, reports it. It targets the `smarter.sh/v1` api, which every manifest's `apiVersion` names. Compatibility across platform releases comes from forward-compatible structs (ADR-004) and contract tests (ADR-005), not from matching version numbers. `client.Version(ctx)` reports the platform's version.

While the SDK is at `v0`, a minor release may break the api, as semver allows. A move to `v2` and later must change the module path in `go.mod` and every import, in the same release.

## Alternatives Considered

- Release the SDK in lockstep with the platform.

## Consequences

- **Positive:**
  - The SDK can be fixed and released without a platform release.
- **Negative:**
  - Compatibility must be tested, not assumed, and a major release is costly for callers, so the api should be stable before `v1`.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
- [ADR-010: CI/CD](010-ci-cd.md)
