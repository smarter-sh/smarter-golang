---
name: smarter-golang-types
description: Use when writing or reviewing types in the smarter-golang SDK, especially any use of any or interface{}, map[string]any, pointers versus values for optional fields, type assertions, or a type that repeats. Explains when any is honest, the aliases in types.go (JSONValue, JSONObject, ID), the forward-compatibility rules for response structs (strings, not enum types; pointers for optional numbers and booleans; timestamps as strings), and the SDK's traps.
---

# Smarter Go SDK Types

A type is a claim about what a value can be. `any` makes no claim, and turns the
compiler's checks off for every value that holds it, in the SDK and in its
callers' code. This is the platform's and the Python SDK's rule (platform
ADR-027, Python SDK `smarter-python-typing`), applied to Go (ADR-013).

## When `any` is acceptable

Only where the value really can be any JSON, through the aliases of `types.go`:

| Alias        | Means                                 | Python SDK              | Use for                             |
| ------------ | ------------------------------------- | ----------------------- | ----------------------------------- |
| `JSONValue`  | `any`: a decoded JSON value           | `SmarterJsonType`       | a tool's result, a proxy's response |
| `JSONObject` | `map[string]any`: a JSON object       | `SmarterJsonObjectType` | a manifest's `spec`, a free object  |
| `ID`         | a resource's id or hashed id, as text | `SmarterIdType`         | every id parameter                  |

Build an `ID` with `smarter.IntID(5)` or `smarter.ID("rMTAwMDAwNQx")`.

## When `any` is wrong

- **A response.** It is a struct (ADR-004). Decode into the struct, not into a
  map.
- **A parameter that takes a few types.** Write a method per type, or a small
  interface or constructor (as `ID` does).
- **Generic code** that works for several structs: use a type parameter
  (`KindService[T]`), not `any`.

## Response structs (ADR-004)

- **No enum types** for values that the platform may add to (a plugin's class,
  an event's type): a `string`, with the known values in its doc comment. Keep
  typed constants for values that the SDK sends (e.g.
  `smarter.StageInput`).
- **Optional strings, slices, maps and structs** are plain fields with
  `omitempty`; their zero value means absent.
- **Optional numbers and booleans** are pointers (`*int`, `*bool`,
  `*float64`), because `0` and `false` are real values. Give callers the
  helpers `smarter.Ptr(v)` and `smarter.Deref(p)`.
- **Timestamps** are `string`s, not `time.Time`: some platform responses carry
  naive timestamps, which `time.Time` can't decode.
- **JSON tags are the wire's names**, exactly (`json:"hashedId,omitempty"`)
  (ADR-021). The contract tests compare them with the platform's schemas.
- A union in the api (a message's `content` is a string or a list of parts) is
  a struct with a custom `UnmarshalJSON` and `MarshalJSON`, tested both ways.

## Assertions

A type assertion (`v.(string)`) on a `JSONValue` must use the two-value form
(`s, ok := v.(string)`). A one-value assertion panics on a platform's
unexpected value, and a library must not panic (`smarter-golang-development`).

## Checking your work

```console
go vet ./...
git diff -- '*.go' ':!sam/' | grep -nE '^\+.*(\bany\b|interface\{\}|\.\([A-Za-z\[\]]+\)$)'
```

For each hit, ask: can I state what this value is? If so, state it.
