# ADR-012: Doc Comments and Documentation

## Status

Accepted

Follows the Python SDK's [ADR-012](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/012-documentation.md).

## Context

Go's api reference is [pkg.go.dev](https://pkg.go.dev), which renders every module's doc comments and runnable examples, and which every Go programmer already reads. The SDK's users also read docs.smarter.sh, which is built with Sphinx from the platform's repository.

## Decision

- Every exported identifier has a **Go doc comment**, in the Go style: a sentence that starts with the identifier's name. It says the same as the Python SDK's docstring of the same object. The package has a `doc.go` overview, and the common tasks have runnable `Example` functions, which `go test` compiles and runs.
- The **api reference is pkg.go.dev**, generated from those comments. Nothing is generated into the platform's docs.
- The SDK's **guides** live in the platform's repository, under `docs/source/smarter-framework/smarter-golang*`, beside the other SDKs', so that docs.smarter.sh is one site, and link to pkg.go.dev for the reference.

## Alternatives Considered

- Generate reST reference pages into the platform's docs, as the Python SDK does. Sphinx has no Go domain, and pkg.go.dev already does it better.

## Consequences

- **Positive:**
  - A reference that can't drift from the code, where Go programmers look for it, and examples that are tested.
- **Negative:**
  - Documenting the SDK touches two repositories, and the reference and the guides are on two sites.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-013: Types](013-types.md)
