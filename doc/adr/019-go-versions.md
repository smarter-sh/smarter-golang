# ADR-019: Supported Go Versions

## Status

Accepted

Adapts the Python SDK's [ADR-019: Supported Python Versions](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/019-python-versions.md).

## Context

The Go team supports the two newest Go releases, and releases one every six months. The `go` line of a library's `go.mod` is the minimum Go version of every program that imports it. The cli builds with Go 1.27.

## Decision

The SDK supports the Go releases that the Go team supports: 1.26 and 1.27 as of October 2026. `go.mod` says `go 1.26.0`, and CI tests both (ADR-010). Code may use what Go 1.26 has, and nothing newer.

## Alternatives Considered

- Only the newest Go, as the cli builds with. A library would force its callers to upgrade.
- Older releases. The Go team no longer fixes their security bugs.

## Consequences

- **Positive:**
  - The SDK builds wherever a supported Go does.
- **Negative:**
  - The minimum rises every six months, which is a breaking change for callers on an unsupported Go.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-010: CI/CD](010-ci-cd.md)
