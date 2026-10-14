# ADR-007: Logging

## Status

Accepted

Follows the Python SDK's [ADR-007](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/007-logging.md), which adapts the platform's [ADR-011: Logging](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/011-logging.rst).

## Context

Go's standard structured logger is `log/slog`. A library must not write to its caller's output unasked, and must never log an api key. The cli logs with the `log` package when `--verbose` is set.

## Decision

The client logs to a `*slog.Logger`, given with `WithLogger`. By default it logs nothing (a discarding handler). `SMARTER_LOG=info` (requests: method, url, status, duration) or `SMARTER_LOG=debug` (and their bodies) logs to stderr with a text handler, when no logger is given. Every record has the attribute `component=smarter` and its source (`http`, `config`). Api keys are masked by `MaskSecret()`, and the client's `LogValue()` masks the key when a client is logged.

## Alternatives Considered

- The `log` package, as the cli uses. It has no levels, and writes to a global logger.
- zap, zerolog or logrus. A dependency that the caller may not use.

## Consequences

- **Positive:**
  - The SDK writes to the application's own slog handler, with levels and attributes.
  - Debugging a request needs only an environment variable, the same one as in the Python SDK.
- **Negative:**
  - Debug logs contain request bodies, e.g. prompts, which the caller must treat as sensitive.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-016: Security](016-security.md)
