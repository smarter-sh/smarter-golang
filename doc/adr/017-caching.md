# ADR-017: No Response Caching

## Status

Accepted

Follows the Python SDK's [ADR-017](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/017-caching.md), which adapts the platform's [ADR-018: Caching](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/018-caching.rst).

## Context

The platform caches function results server-side. A client-side cache of resources would serve stale data after a change made elsewhere, e.g. by the cli.

## Decision

The SDK doesn't cache responses. Every call reads the platform's current state. Callers who need a cache add one with their own invalidation rules.

## Alternatives Considered

- An LRU cache of resources.

## Consequences

- **Positive:**
  - No stale reads, and no cache invalidation in the SDK.
- **Negative:**
  - Repeated reads cost a request each.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-008: HTTP Transport, Retries and Timeouts](008-http-transport.md)
