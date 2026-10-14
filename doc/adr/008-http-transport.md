# ADR-008: HTTP Transport, Retries and Timeouts

## Status

Accepted

Follows the Python SDK's [ADR-008](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/008-http-transport.md).

## Context

Requests fail for transient reasons: timeouts, rate limits, gateways. The platform also answers 503 for a resource that isn't ready (`SAMBrokerErrorNotReady`), which no retry will fix. A prompt that calls several tools can take tens of seconds. Go's `net/http` is complete, and `context` carries cancellation and deadlines.

## Decision

All requests go through one transport, `httpClient` in `http.go`, on `net/http`. Its retry rules are the Python SDK's: it retries a request that got no response, or a 408, 429, 502 or 504, and a 503 only when the platform didn't send it (its body is not a journaled response), with exponential backoff and jitter, honoring `Retry-After` up to 60 seconds, and never after the caller's context is done. The default is 2 retries and a 60 second timeout per attempt. The caller's `context.Context` cancels a request at any time.

Callers may pass their own `*http.Client` (`WithHTTPClient`, e.g. with a proxy or a test `RoundTripper`). Prompt progress is read as Server-Sent Events by the SDK's own parser, a port of the Python SDK's.

## Alternatives Considered

- A retrying client library (`hashicorp/go-retryablehttp`). A dependency for a loop, with different retry rules.
- No retries.

## Consequences

- **Positive:**
  - No HTTP dependency, and the same retry behavior as every other Smarter SDK.
- **Negative:**
  - A retried prompt may be billed twice if the first attempt reached the LLM. `WithMaxRetries(0)` disables retries where that matters.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-006: Error Handling](006-error-handling.md)
- [ADR-018: Blocking Calls With a Context](018-blocking-calls.md)
