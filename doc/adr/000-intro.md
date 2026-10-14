# ADR Introduction

Architectural Design Records (ADRs) are short documents that capture a single architectural or design decision, its context, and its consequences. Less is more: keep them brief.

These are the ADRs of the Smarter Go SDK. They follow the Smarter Python SDK's [ADRs](https://github.com/smarter-sh/smarter-python/tree/main/doc/adr), which is the reference implementation of every Smarter SDK, and through them the Smarter platform's own [ADRs](https://github.com/smarter-sh/smarter/tree/main/docs/source/adr). ADRs 001 to 019 keep the Python SDK's numbers and titles, so that the same decision has the same number in every SDK, and each one says how it adapts the Python decision to Go. ADRs 020 and 021 are decisions that only a Go SDK has to make. A decision that every SDK and the smarter cli share, made later, takes the next number free in every repository: ADR-022 is the first.

The [smarter cli](https://github.com/smarter-sh/smarter-cli) is also written in Go. Where the cli has already made a decision that a Go library shares (the configuration file, the Go toolchain, the lint hooks), these ADRs follow it.

The platform's decisions about its own runtime (Django, Docker, Kubernetes, the database, Celery, React, templates, feature flags, internationalization) don't apply to an SDK and are not repeated here.

See: [Michael Nygard's ADRs](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html)

## Guidelines for writing ADRs

- **One decision per ADR.**
- **Be concise.** Focus on clarity and brevity.
- **State the context:** the forces that led to the decision.
- **List the alternatives**, and why they weren't chosen.
- **Explain the decision**, and **describe its consequences**, positive and negative.
- **Use simple language**, for future contributors without today's context.
- **Use the same format** as the others, and **reference related ADRs**.
- **Keep the numbering aligned** with the Python SDK: a decision that every SDK shares takes the Python SDK's number.

## Index

- [ADR-001: Scope and Purpose of the SDK](001-scope.md)
- [ADR-002: Configuration](002-configuration.md)
- [ADR-003: Writes Go Through the Manifest API](003-writes-through-manifests.md)
- [ADR-004: Go Structs, Forward Compatible](004-go-models.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
- [ADR-006: Error Handling](006-error-handling.md)
- [ADR-007: Logging](007-logging.md)
- [ADR-008: HTTP Transport, Retries and Timeouts](008-http-transport.md)
- [ADR-009: Testing](009-testing.md)
- [ADR-010: CI/CD](010-ci-cd.md)
- [ADR-011: Dependency Management](011-dependencies.md)
- [ADR-012: Doc Comments and Documentation](012-documentation.md)
- [ADR-013: Types](013-types.md)
- [ADR-014: Code Quality](014-code-quality.md)
- [ADR-015: Semantic Versioning and API Versions](015-versioning.md)
- [ADR-016: Security](016-security.md)
- [ADR-017: No Response Caching](017-caching.md)
- [ADR-018: Blocking Calls With a Context](018-blocking-calls.md)
- [ADR-019: Supported Go Versions](019-go-versions.md)
- [ADR-020: One Package, Functional Options](020-package-layout.md)
- [ADR-021: Names: Go Identifiers, the Wire's JSON Tags](021-naming.md)
- [ADR-022: A Pinned Copy of the Platform's OpenAPI Document](022-pinned-openapi.md)
