# ADR-001: Scope and Purpose of the SDK

## Status

Accepted

Follows the Python SDK's [ADR-001](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/001-scope.md).

## Context

The Smarter platform is managed through its REST api. The smarter cli serves people at a terminal and is itself written in Go, but its http code lives in its `cmd` package, untyped and tied to its commands, so other Go programs can't use it. Go programmers who build services, operators and infrastructure tooling on Smarter had to hand-write HTTP calls, or customize the platform. The platform's [APIs and SDKs](https://docs.smarter.sh/en/latest/smarter-framework/apis-and-sdks.html) page lists a Go SDK as planned, and sets the requirements that every SDK meets.

## Decision

smarter-golang is the platform's official, complete Go SDK: the module `github.com/smarter-sh/smarter-golang`, package `smarter`. It is to Smarter what `openai-go` is to the OpenAI api:

- it covers **all** of the platform's public api: every resource kind through the manifest api, the chat and prompt apis, and every resource's REST reads and actions,
- it exposes the platform's own models: a Go struct of every SAM kind, and the api contracts (ADR-005),
- it has the **same features and vocabulary as the Python SDK**, which is the reference implementation, in idiomatic Go (ADR-021),
- it is a library that the **smarter cli can adopt** for its own requests: it reads the cli's configuration file exactly as the cli does (ADR-002), and has no dependency on cobra or viper.

The platform is never imported. Everything goes through its http api, so the SDK works against a platform on a laptop or on Kubernetes, of any version that serves `smarter.sh/v1`.

## Alternatives Considered

- Generate a client from the platform's `openapi.json` (e.g. `oapi-codegen`). It would type the REST routes, but not the manifest api's broker semantics, the configuration lookup, the chat session and streaming helpers, or the error hierarchy, and generated Go clients read poorly.
- Move the cli's `cmd/api.go` into an importable package. It covers only the cli api, untyped.

## Consequences

- **Positive:**
  - Go software can integrate with Smarter without forking it.
  - The cli can share one tested http client instead of maintaining its own.
- **Negative:**
  - The SDK must track the platform's api, and the Python SDK's features, as they change (ADR-005, ADR-015).

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-002: Configuration](002-configuration.md)
- [ADR-003: Writes Go Through the Manifest API](003-writes-through-manifests.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
