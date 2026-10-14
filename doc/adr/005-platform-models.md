# ADR-005: Mirrored Contracts and Generated SAM Structs

## Status

Accepted

Follows the Python SDK's [ADR-005](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/005-platform-models.md).

## Context

The platform's Pydantic models are the source of truth of its api: the SAM models of every manifest kind, and the published contracts (the Smarter Chat api, the Smarter Journal, the api's OpenAPI document). The platform commits them as JSON in `smarter/smarter/apps/api/v1/contract/` and beside each contract's models. The SDK can't import them, and must not drift from them.

## Decision

- The contract files are copied from a platform checkout, at a git ref, by `scripts/sync-contracts.sh` (`make contracts`), into `testdata/contracts/` and `testdata/sam/`, exactly as the Python SDK copies them.
- **SAM structs** are generated from each kind's JSON Schema into the `sam` package (`sam/<kind>.go`) by the SDK's own generator, `internal/cmd/samgen` (`make sam-models`). It reads the subset of JSON Schema that Pydantic writes (`$defs`, `$ref`, `anyOf` with `null`, `enum`, `items`, `additionalProperties`), and writes one struct per definition, with the platform's descriptions as doc comments. Each kind gets a top-level struct, e.g. `sam.Guardrail`, and `sam.Kinds` maps each kind to its struct. Generated code is never edited by hand.
- A test decodes every kind's example manifest into its struct with `DisallowUnknownFields`, so a field that the platform adds and the generator missed fails it.
- **Contracts** are mirrored by hand in `chat.go` and `journal.go`. A test fails when the platform's schema declares a property that the mirror's JSON tags don't, or when a platform example doesn't decode.
- The SDK reads **only version 2** of the chat contract, and asks for it in every request (`X-Smarter-Capabilities: chat-contract-v2`).
- A test checks that every request that the SDK sends is an operation of the platform's `openapi.json`, and that the SDK calls every operation, apart from a named list.

## Alternatives Considered

- Hand-write the SAM structs. 25 kinds, which drift as soon as they are written.
- A third-party generator (`go-jsonschema`, `oapi-codegen`). Each makes choices about optional fields, unions and names that differ from ADR-004 and ADR-021, and Pydantic's schemas use a small, regular subset of JSON Schema.

## Consequences

- **Positive:**
  - Callers get the platform's own models, as Go structs.
  - Drift is caught by tests, not by users.
- **Negative:**
  - The generator is code to maintain, and must learn any new JSON Schema construct that the platform's models start to use. It fails loudly on one it doesn't know.
  - Refreshing is a manual step (`make contracts`, `make sam-models`), monthly at least (the `monthly-maintenance` skill).
  - Generated structs check structure, not the platform's business rules: `Manifests.Validate` does.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
- [ADR-009: Testing](009-testing.md)
