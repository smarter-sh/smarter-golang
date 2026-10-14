# ADR-009: Testing

## Status

Accepted

Follows the Python SDK's [ADR-009](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/009-testing.md), which adapts the platform's [ADR-013: Testing](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/013-testing.rst).

## Context

The SDK's correctness depends on a platform that it can't run in CI, and the local platform's containers hold credentials for real cloud infrastructure.

## Decision

`go test` runs two suites:

- **Unit tests** (`<file>_test.go`, one per source file, in the external package `smarter_test`, so that they use only the public api) build their client with `newMockPlatform(t)` (`helpers_test.go`), the SDK's counterpart of the Python SDK's `SmarterTestBase`. A `RoundTripper` answers every request from the routes that the test registers, with sanitized copies of real platform responses (`testdata/`, the same fixtures as the Python SDK's). They run in CI on every supported Go version (ADR-019), with the race detector, and at least 90% coverage.
- **Live tests** (`live_test.go`, skipped unless `SMARTER_TEST_LIVE=true`) run against a real platform, and validate every struct against real responses. They may send billable prompts, and create and delete throwaway resources. **They never deploy anything.** A platform error skips a subtest with the platform's message, and an SDK error fails it.

## Alternatives Considered

- `httptest.Server`. A real socket per test, slower, and its url changes per test; a `RoundTripper` exercises the same code above the transport.
- Only mocked tests, or only live tests.

## Consequences

- **Positive:**
  - Fast, hermetic CI, and structs checked against reality before release.
  - Fixtures are shared with the Python SDK.
- **Negative:**
  - Live tests must be run by hand before a release.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
- [ADR-010: CI/CD](010-ci-cd.md)
- [ADR-016: Security](016-security.md)
