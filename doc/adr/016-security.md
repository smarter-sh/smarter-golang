# ADR-016: Security

## Status

Accepted

Follows the Python SDK's [ADR-016](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/016-security.md), which adapts the platform's [ADR-016: Security](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/016-security.rst).

## Context

The SDK holds api keys that can change production resources, and some of its operations create billable infrastructure. Its repository is public, and its tests use data from real platforms. `fmt.Printf("%+v", client)` prints every field of a struct.

## Decision

- The api key is an unexported field, sent only in the `Authorization` header. The client's `String`, `GoString` and `LogValue` methods mask it, and so do the SDK's logs.
- `Deploy` methods say in their doc comments that they create real infrastructure, and no test calls them against a platform.
- Test fixtures from real responses are sanitized: no names, emails, addresses, account numbers, keys or private hostnames.
- The SDK never executes code from a response, and never reads a `.env` file on its own.
- gosec runs in golangci-lint (ADR-014).

## Alternatives Considered

- Leave key handling to the caller.

## Consequences

- **Positive:**
  - Keys don't leak through logs or printed structs.
- **Negative:**
  - Sanitizing fixtures is manual, and must be reviewed.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-002: Configuration](002-configuration.md)
- [ADR-007: Logging](007-logging.md)
- [ADR-009: Testing](009-testing.md)
