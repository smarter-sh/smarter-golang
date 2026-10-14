# ADR-003: Writes Go Through the Manifest API

## Status

Accepted

Follows the Python SDK's [ADR-003](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/003-writes-through-manifests.md), and the platform's [ADR-005: Manifests](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/005-manifests.rst), [ADR-006: Rest API](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/006-rest-api.rst) and [ADR-024: Broker Model](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/024-broker-model.rst).

## Context

The platform has two ways to change a resource: the brokered manifest api (`/api/v1/cli/apply/`, `delete/`, ...), which validates a manifest with the kind's Pydantic models and applies its business rules, and the REST views' own POST, PATCH and DELETE handlers, which are inconsistent: some create rows from the raw request, set arbitrary attributes, or answer with redirects.

## Decision

As in the Python SDK, and as the cli does, the SDK creates, changes and deletes resources **only** through the manifest api. It uses the REST api to **read** resources (`List`, `Retrieve`, `Find`), and for **actions** that the manifest api has no verb for, such as evaluating a Guardrail, calling an MCP tool, or searching a vectorstore. Each resource family (`client.Guardrails`, ...) binds the manifest operations to its kind.

A manifest is a `*Manifest`; `ManifestFromYAML`, `ManifestFromFile` and `ManifestFromSAM` build one from YAML or JSON text, a file, or a typed SAM struct.

## Alternatives Considered

- Wrap every REST handler, including its writes.
- Offer both, and let the caller choose.

## Consequences

- **Positive:**
  - Every write is validated and applied by the platform's broker, as the cli's are.
  - A manifest that works with the cli works with the SDK, and the other way round.
- **Negative:**
  - A write takes a whole manifest, not a single field. Read, change and apply the manifest (`Describe`, edit, `Apply`).

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-001: Scope and Purpose of the SDK](001-scope.md)
- [ADR-005: Mirrored Contracts and Generated SAM Structs](005-platform-models.md)
