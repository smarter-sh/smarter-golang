# ADR-022: A Pinned Copy of the Platform's OpenAPI Document

## Status

Accepted

Follows the Python SDK's [ADR-022](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/022-pinned-openapi.md). The Node.js SDK and the smarter cli share this decision, under the same number.

## Context

The platform publishes its api as an OpenAPI document, generated from its Pydantic models and committed as `smarter/smarter/apps/api/v1/contract/openapi.json`. It covers the cli api (`/api/v1/cli/`), the chat api, and every family of REST routes, with their request and response schemas.

The SDK's unit tests answer requests with a mock platform, a `RoundTripper` (ADR-009). A mock answers whatever path it is asked, so it can't tell when the SDK calls a route that the platform renamed or removed. The smarter cli, also Go, shipped two broken commands for that reason, with every unit test passing.

## Decision

The SDK keeps a **copy** of the platform's `openapi.json`, pinned to a platform version, in `testdata/contracts/openapi.json`, and tests against it:

- `scripts/sync-contracts.sh` (`make contracts`, or `make contracts SMARTER_REF=<tag>`) copies it from a checkout of the platform, at its working tree or at a git ref, with the other contracts (ADR-005). The copy is committed, and is never edited by hand.
- `TestEveryRequestIsPublished` (`contracts_test.go`) checks that every request the SDK sends matches an operation of the document (method and path template), and that the SDK calls every operation, apart from a named list (`notCalled`) of operations it deliberately doesn't call.
- Refreshing the copy is part of the monthly maintenance; a failing contract test after a refresh is how the SDK learns what changed.

## Alternatives Considered

- Fetch the document from a running platform during the tests. Tests would need the network and a platform, and would change under the SDK without a commit.
- Read it from `../smarter` during the tests. CI has no platform checkout.
- Generate the client from the document with `oapi-codegen` (ADR-001).
- Rely on live tests alone. They are opt-in, and can't cover operations that are unsafe to call (deploy).

## Consequences

- **Positive:**
  - Drift between the SDK and the platform fails a unit test, in CI, rather than reaching users.
  - The diff of the copy, at each refresh, is a precise list of the api's changes.
  - The four Smarter clients (three SDKs and the cli) check themselves against the same document.
- **Negative:**
  - The copy is only as current as its last refresh.
  - The `notCalled` list must be kept honest.
  - Unlike the Python and Node.js SDKs, the Go SDK doesn't yet validate its recorded fixtures in `testdata/` against the document's response schemas, so a fixture that drifts from what the platform sends goes unnoticed.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
- [ADR-009: Testing](009-testing.md)
