# ADR-004: Go Structs, Forward Compatible

## Status

Accepted

Adapts the Python SDK's [ADR-004: Pydantic Models, Forward Compatible](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/004-pydantic-models.md), and through it the platform's [ADR-031: Pydantic Models Over Raw JSON](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/031-pydantic-over-json.rst).

## Context

Every response is JSON. The Python SDK validates every response into a model that keeps the fields it doesn't declare, because the SDK runs against platforms newer than itself, which add fields and enum values in minor releases. Go decodes JSON into structs with `encoding/json`, which ignores unknown fields by default, and has no optional type.

## Decision

Every response is decoded into an exported struct, never returned as `map[string]any`. As in the Python SDK:

- **an unknown field never fails a decode**: structs are decoded without `DisallowUnknownFields`,
- a value that the platform may add to (a plugin's class, a progress event's type) is a `string` whose known values are listed in its doc comment, not a named type with constants,
- a field whose shape depends on a kind (a manifest's `spec`) is a `JSONObject`, so nothing in it is lost (ADR-013),
- a response that can't be decoded into its struct returns an `*APIResponseValidationError`.

The Go idioms:

- An optional string, slice, map or struct is a plain field, whose zero value means absent. An optional number or boolean is a pointer (`*int`, `*bool`), because `0` and `false` are real values.
- Timestamps are `string`s in ISO 8601, not `time.Time`: some platform responses carry naive timestamps, which `time.Time` can't decode, and a decode failure would fail the whole response.
- Unlike the Python models, a typed struct doesn't keep fields that it doesn't declare. A resource's full manifest is a `*Manifest`, whose `spec` and `status` are `JSONObject`s, so describe, change and apply loses nothing.

## Alternatives Considered

- Pointers for every optional field, as go-github does. Every read then needs a nil check.
- Keep unknown fields in an `Extra map[string]json.RawMessage` on every struct, with a custom `UnmarshalJSON` per struct. Much generated code for little use; the `Manifest` covers the case that needs it.
- `map[string]any` responses.

## Consequences

- **Positive:**
  - Callers get typed structs, with editor completion.
  - An SDK release keeps working when the platform is upgraded.
- **Negative:**
  - A field that the platform removes or renames is silently zero, not an error. The live and contract tests catch it (ADR-005, ADR-009).
  - A typed struct can't be used to round-trip fields that it doesn't declare.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
- [ADR-013: Types](013-types.md)
- [ADR-021: Names: Go Identifiers, the Wire's JSON Tags](021-naming.md)
