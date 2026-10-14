# ADR-006: Error Handling

## Status

Accepted

Follows the Python SDK's [ADR-006](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/006-error-handling.md), which adapts the platform's [ADR-012: Error Handling](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/012-error-handling.rst).

## Context

Callers need to tell a missing resource from a bad manifest, an expired key, an unreachable server, or a platform bug, and the platform returns errors in several shapes: the journal's `error` object, an `error` string, Django REST Framework's `detail`, and Django's html error page. Go has no exception hierarchy: errors are values, inspected with `errors.Is` and `errors.As`.

## Decision

The Python SDK's hierarchy becomes sentinel errors and error types:

| Python SDK                                               | Go SDK                                                         |
| -------------------------------------------------------- | -------------------------------------------------------------- |
| `SmarterError`                                           | `errors.Is(err, smarter.ErrSmarter)`, true for every SDK error |
| `SmarterConfigurationError`                              | `*ConfigurationError`                                          |
| `SmarterValueError`                                      | `*ValueError`                                                  |
| `ResourceNotFoundError`                                  | `errors.Is(err, smarter.ErrResourceNotFound)`                  |
| `APIConnectionError`, `APITimeoutError`                  | `*APIConnectionError` (`Timeout()` reports a timeout)          |
| `APIResponseValidationError`                             | `*APIResponseValidationError`                                  |
| `APIStatusError`                                         | `*APIStatusError`                                              |
| `NotFoundError`, `BadRequestError`, ... (one per status) | `errors.Is(err, smarter.ErrNotFound)`, `ErrBadRequest`, ...    |
| `SmarterPromptError`                                     | `*PromptError`                                                 |

An `*APIStatusError` carries `StatusCode`, a readable `Message` from whichever shape the platform used, the raw `Body`, the response's `Header`, and the journal's `JournalError` with `ErrorClass()` when there is one. Its `Is` method matches the sentinel of its status, so `errors.Is(err, smarter.ErrNotFound)` works through any wrapping, and `ErrNotFound` also matches `ErrResourceNotFound`, which `Find` returns, so one check handles both, as in Python. Errors wrap their cause (`Unwrap`), e.g. `context.DeadlineExceeded`.

## Alternatives Considered

- One error type per status, as Python has. Ten types that callers would test with `errors.As`, for what a sentinel says.
- Return `*http.Response` errors as is.

## Consequences

- **Positive:**
  - Idiomatic checks, which keep working through `fmt.Errorf("...: %w", err)`.
  - Platform errors keep their server-side class name.
- **Negative:**
  - The mapping of the platform's error shapes must follow the platform.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
- [ADR-008: HTTP Transport, Retries and Timeouts](008-http-transport.md)
