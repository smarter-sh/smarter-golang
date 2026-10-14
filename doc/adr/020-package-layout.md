# ADR-020: One Package, Functional Options

## Status

Accepted

## Context

A Go SDK can split its api across many packages (`client`, `option`, `resources/...`), or keep it in one. Each extra package is another import for callers, and Go forbids import cycles, which pushes shared types into yet more packages. The Python SDK's constructor takes keyword arguments, which Go doesn't have.

## Decision

- The whole public api is the package `smarter`, at the module's root: the client, the resources (`client.LLMClients`, ...), the models, the errors and the options. The generated SAM structs are the package `sam`, because they are many (ADR-005) and their names would collide with the REST models (`sam.Guardrail` is a manifest, `smarter.Guardrail` a REST resource). The generator is in `internal/cmd/samgen`.
- `NewClient` takes **functional options**: `smarter.NewClient(smarter.WithAPIKey(key), smarter.WithTimeout(30*time.Second))`. A resource method's own optional arguments are a params struct, e.g. `smarter.PromptParams{SessionKey: "..."}`, as in other Go SDKs.

## Alternatives Considered

- A `Config` struct passed to `NewClient`. Zero values can't be told from "not set" (e.g. `MaxRetries: 0`), and every new field is visible to every caller.
- One package per resource, as `openai-go` does with `option`. More imports, for an api of this size.

## Consequences

- **Positive:**
  - One import (`smarter "github.com/smarter-sh/smarter-golang"`), and options that can grow without breaking callers.
- **Negative:**
  - The package is large; its doc overview (`doc.go`) must guide readers through it.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-002: Configuration](002-configuration.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
