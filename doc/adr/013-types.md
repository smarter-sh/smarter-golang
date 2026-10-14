# ADR-013: Types

## Status

Accepted

Follows the Python SDK's [ADR-013: Type Annotations](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/013-type-annotations.md), and the platform's [ADR-027: Python Annotations](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/027-annotations.rst).

## Context

Go is statically typed, but `any` (`interface{}`) switches the compiler's checks off for the values that hold it, in the SDK and in its callers' code.

## Decision

`any` is used only for JSON whose shape really is free, through the aliases of `types.go`, which mirror the Python SDK's `smarter.core.typing`:

| Alias        | Means                                 | Python SDK              |
| ------------ | ------------------------------------- | ----------------------- |
| `JSONValue`  | `any`, a decoded JSON value           | `SmarterJsonType`       |
| `JSONObject` | `map[string]any`                      | `SmarterJsonObjectType` |
| `ID`         | a resource's id or hashed id, as text | `SmarterIdType`         |

An `ID` is built with `smarter.IntID(5)` or `smarter.ID("rMTAwMDAwNQx")`, because a Go method can't take "an `int` or a `string`".

## Alternatives Considered

- `any` for every id parameter. It would accept anything, and fail at run time.

## Consequences

- **Positive:**
  - The compiler checks every call and every response field.
- **Negative:**
  - Types are part of the public api, and change with it (ADR-015).

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
