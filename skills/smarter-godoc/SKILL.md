---
name: smarter-godoc
description: Use when writing or editing doc comments or runnable examples in the smarter-golang SDK. Doc comments are the SDK's api reference on pkg.go.dev, follow Go's doc comment syntax, and say what the Python SDK's docstring of the same object says. Covers the form of each kind of comment, doc.go, Example functions, links, deprecations, and how to check them.
---

# Smarter Go SDK Doc Comments

The SDK's api reference is pkg.go.dev, generated from its doc comments
(ADR-012), and editors show them on hover. Write for a Go programmer who is new
to Smarter, in [Go's doc comment syntax](https://go.dev/doc/comment).

**Start from the Python SDK's docstring** of the same object
(`../smarter-python/smarter/...`), and translate it: same summary, same
parameters, same warnings, with this SDK's names. The SDKs' docs should read as
one set.

## Form

```go
// Evaluate dry-runs a Guardrail on a text, without saving an event.
//
// The text is evaluated as the params' Stage, "input" (a prompt) by default or
// "output" (an LLM's reply); the Guardrail must run on that stage. It returns
// whether the Guardrail triggered, its disposition, and the text after its
// action.
//
// It returns an error that matches [ErrBadRequest] if the text is empty or too
// long, or the Guardrail does not run on the stage.
func (r *GuardrailsService) Evaluate(ctx context.Context, id ID, text string, params *GuardrailEvaluateParams) (*GuardrailEvaluation, error)
```

- **The first sentence starts with the identifier's name** and is a complete
  sentence: `// Evaluate dry-runs ...`, `// Guardrail is a ...`,
  `// ErrNotFound matches a 404 response.`. revive enforces it.
- Then a blank `//` line, then the body: parameters, the return value, the
  errors a caller should check (by sentinel or type), and side effects that
  matter to a caller: **billable** LLM requests, and **real infrastructure**
  that a deploy creates.
- Go has no `@param` tags: name parameters in prose.
- **Struct fields** get a comment each when their meaning isn't obvious from
  the name. Values that the platform may add to are listed:
  `// Transport is http, sse or stdio.`
- **Generated code** (`sam/`) takes its comments from the platform's schema.
  Fix those in the platform's models, then regenerate.
- Say why, not only what, when the reason isn't obvious. Be accurate: when you
  change behavior, update the comment in the same edit.

## Syntax

- Links: `[Client]`, `[Client.Manifests]`, `[ErrNotFound]`, `[sam.Guardrail]`,
  `[context.Context]`. They become links on pkg.go.dev.
- Code blocks are indented by a tab after `//`, with a blank `//` line before.
- Lists: `//   - item`.
- Headings in `doc.go`: `// # Heading`.
- `// Deprecated: Use X instead.` as its own paragraph, for a renamed api kept
  for a release.

## doc.go and examples

- `doc.go` holds the package comment: what the SDK is, how to create a client,
  the configuration lookup, the resources, errors, streaming, and links to the
  guides on docs.smarter.sh. It is the first thing pkg.go.dev shows.
- `example_test.go` holds runnable examples, which pkg.go.dev shows next to the
  identifier they name: `ExampleNewClient`, `ExampleLLMClientsService_Prompt`,
  `ExampleManifestsService_Apply`. They run against a mock platform, with an
  `// Output:` comment, so `go test` checks them.

## Test names

A test's name states what it covers (`TestGuardrailsEvaluate`), and a subtest
the behavior it proves (`t.Run("sends the text and stage", ...)`).

## Verify

```console
go doc -all . | less              # the package as pkg.go.dev will render it
go test -run Example ./...        # the examples
golangci-lint run                 # revive checks the comments' form
```
