# ADR-018: Blocking Calls With a Context

## Status

Accepted

Replaces, for Go, the Python SDK's [ADR-018: A Synchronous Client First](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/018-synchronous-client.md).

## Context

Python had to choose between a synchronous and an `asyncio` client. Go doesn't: calls block, and callers run them concurrently in goroutines. A Go library's calls take a `context.Context` for cancellation and deadlines.

## Decision

Every method that sends a request blocks until it has its response, takes a `context.Context` as its first argument, and is safe to call from many goroutines on one `*Client`. A streamed prompt is a `*PromptStream` that is read with `Next()`, `Current()` and `Err()`, like `bufio.Scanner`, then gives its `Response()`, and must be `Close()`d.

## Alternatives Considered

- Channels of events. They leak a goroutine when the caller stops reading.
- `iter.Seq2` (range over function). Readable, but the stream's response and its close still need methods, so it adds a second way to read a stream.

## Consequences

- **Positive:**
  - Idiomatic Go, and concurrency is the caller's choice.
- **Negative:**
  - Every call takes a context, also where the caller has none to give (`context.Background()`).

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-008: HTTP Transport, Retries and Timeouts](008-http-transport.md)
