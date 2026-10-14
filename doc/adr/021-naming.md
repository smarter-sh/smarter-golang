# ADR-021: Names: Go Identifiers, the Wire's JSON Tags

## Status

Accepted

## Context

The platform's [APIs and SDKs](https://docs.smarter.sh/en/latest/smarter-framework/apis-and-sdks.html) design goals ask every SDK to be idiomatic in its language, with the platform's vocabulary. Go's names are exported in PascalCase, and initialisms keep one case (`ID`, `URL`, `API`, `LLM`, `MCP`). The api's JSON mixes camelCase (REST resources) and snake_case (the chat contract).

## Decision

- **Identifiers are Go's**: `client.LLMClients.Prompt`, `client.Manifests.ExampleManifest`, `LLMClient.HashedID`, `PromptChatAPIMessage`.
- **The vocabulary is the Python SDK's**, translated mechanically: `llm_clients` is `LLMClients`, `add_text` is `AddText`, `PromptChatApiMessage` is `PromptChatAPIMessage`.
- **The package is the namespace**: a Python name that starts with `Smarter`
  drops it, because `smarter.SmarterJournalMessage` stutters (revive flags it):
  `SmarterJournalMessage` is `smarter.JournalMessage`, `SmarterJournaledResponse`
  is `smarter.JournaledResponse[T]`, and the REST base model `SmarterResource` is
  `smarter.Resource`.
- **Every field's JSON tag is the api's own name**, exactly as it is on the wire (`json:"hashedId,omitempty"`, `json:"session_key,omitempty"`), so a struct encodes back to the platform's JSON.

## Alternatives Considered

- Keep the Python names (`PromptChatApiMessage`). golint and revive would flag every one.

## Consequences

- **Positive:**
  - Code reads like other Go SDKs, and every struct round-trips to the platform's JSON.
- **Negative:**
  - The contract test needs a table from the platform's model names to the Go types (ADR-005).

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-001: Scope and Purpose of the SDK](001-scope.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
