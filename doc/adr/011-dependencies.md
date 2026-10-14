# ADR-011: Dependency Management

## Status

Accepted

Follows the Python SDK's [ADR-011](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/011-dependencies.md), which adapts the platform's [ADR-015: Dependency Management](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/015-dependency-management.rst).

## Context

Go's minimal version selection gives every module in a build the highest minimum version that any of them asks for, so a library's dependency raises its callers' too. Every dependency is also code that its callers must trust.

## Decision

The SDK uses the standard library, and one module: `go.yaml.in/yaml/v3` (the YAML organization's maintained fork of `gopkg.in/yaml.v3`), for manifests and the cli's configuration file. Its `go.mod` asks for the lowest versions that the SDK needs. Development tools (golangci-lint) are installed, not required in `go.mod`, as in the cli. Dependabot and Mergify keep `go.mod` and the workflows current.

## Alternatives Considered

- viper and cobra, as the cli uses. A library has no flags, and viper is large.
- testify for assertions. The standard `testing` package suffices.

## Consequences

- **Positive:**
  - A tiny dependency graph, and nothing to audit but one YAML parser.
- **Negative:**
  - Some helpers (retries, SSE parsing, assertions) are the SDK's own code, which it tests.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-010: CI/CD](010-ci-cd.md)
