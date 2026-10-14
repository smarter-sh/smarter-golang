# ADR-014: Code Quality

## Status

Accepted

Follows the Python SDK's [ADR-014](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/014-code-quality.md), the platform's [ADR-028: Code Quality](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/028-code-quality.rst), and the smarter cli's `.pre-commit-config.yaml`.

## Context

Consistent code is easier to review. The cli already has a Go lint standard.

## Decision

`.pre-commit-config.yaml` is the standard, with the cli's Go hooks (`go-fmt`, `go-vet`, `go-imports`, `golangci-lint`, `go-unit-tests` from `dnephin/pre-commit-golang`), the platform's generic hooks (whitespace, end of file, JSON and YAML syntax, merge conflicts, private keys, AWS credentials), codespell, Prettier for Markdown and YAML, and commitlint on commit messages. `.golangci.yml` enables golangci-lint's standard linters plus `revive`, `gosec`, `errorlint` and `misspell`. Rules aren't disabled indiscriminately: a `//nolint` names its linter and says why.

## Alternatives Considered

- `go vet` alone.

## Consequences

- **Positive:**
  - The SDK reads like the cli, and like most Go code.
- **Negative:**
  - Hook and linter versions must be kept in step with the cli's.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-012: Doc Comments and Documentation](012-documentation.md)
