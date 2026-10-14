---
name: smarter-golang-development
description: Start here for any change to the smarter-golang repository, the Smarter platform's Go SDK (module github.com/smarter-sh/smarter-golang, package smarter). Gives the non-negotiable rules, the SDK's architecture (client, transport, resources, structs, the generated sam package and the mirrored api contracts), its relationship to the Python SDK (the reference implementation) and to the smarter cli (also Go), how to cover a new platform endpoint, and an index of the narrower skills to load for each kind of work.
---

# Smarter Go SDK Development

`github.com/smarter-sh/smarter-golang` (package `smarter`) is to the Smarter
platform what `openai-go` is to the OpenAI api: the library that Go
programmers import to build custom software on their own Smarter installation,
running locally or on Kubernetes. The platform itself is
[smarter-sh/smarter](https://github.com/smarter-sh/smarter), usually checked out
beside this repository as `../smarter`. Its sibling SDKs are
[smarter-python](https://github.com/smarter-sh/smarter-python) (`../smarter-python`)
and [smarter-nodejs](https://github.com/smarter-sh/smarter-nodejs)
(`../smarter-nodejs`). The [smarter cli](https://github.com/smarter-sh/smarter-cli)
(`../smarter-cli`) covers the same api for people at a terminal, and is also
written in Go.

The SDK's goal is to cover **all** of the platform's api, so that nobody needs
to customize or fork the backend to integrate with it.

## The Python SDK is the reference implementation

Every Smarter SDK has the same features, resources, verbs and errors
(`doc/adr/001-scope.md`). **Before you add or change a feature, read how
`../smarter-python` does it**, and keep the two in step: the same endpoint,
query parameters, request body, defaults, error and doc comment, in idiomatic
Go (`doc/adr/021-naming.md`):

| Python SDK                                       | This SDK                                                                   |
| ------------------------------------------------ | -------------------------------------------------------------------------- |
| `client.llm_clients.prompt(5, "hi")`             | `client.LLMClients.Prompt(ctx, smarter.IntID(5), smarter.Text("hi"), nil)` |
| `client.manifests.example_manifest("Guardrail")` | `client.Manifests.ExampleManifest(ctx, "Guardrail")`                       |
| keyword arguments (`session_key=...`)            | a params struct (`&smarter.PromptParams{SessionKey: "..."}`)               |
| Pydantic model, snake_case attributes            | struct, Go field names, **the wire's JSON tags**                           |
| `with client.llm_clients.stream(...) as s:`      | `s := ...Stream(...); defer s.Close(); for s.Next() {...}`                 |
| `except NotFoundError`                           | `errors.Is(err, smarter.ErrNotFound)`                                      |
| `smarter/resources/guardrails.py`                | `guardrails.go`                                                            |
| `tests/test_guardrails.py`                       | `guardrails_test.go`                                                       |

When the Python SDK has a feature that this SDK lacks, that is a bug here. When
you find a platform behavior that the Python SDK handles wrongly, tell the user:
both SDKs should be fixed.

## The smarter cli

The cli (`../smarter-cli`) reads `~/.smarter/config.yaml` with viper, and sends
its own requests from `cmd/api.go`. This SDK reads the same file the same way
(`config.go`, ADR-002) and is meant to be importable by the cli, so:

- keep the configuration lookup identical to the cli's (`cmd/root.go`,
  `cmd/api.go`): when the cli changes it, change it here,
- never add a dependency that the cli would not want (no cobra, no viper, no
  global state, no `os.Exit`, no `log.Fatal`, no `panic` on bad input),
- the cli's Go version and lint hooks are this SDK's too (ADR-014, ADR-019).

## Repository map

| Path                                 | What lives there                                                              |
| ------------------------------------ | ----------------------------------------------------------------------------- |
| `doc.go`                             | The package overview that pkg.go.dev shows first                              |
| `client.go`                          | `Client`, `NewClient`: the resources, `WhoAmI`/`Version`/`Status`/`Kinds`     |
| `options.go`                         | The functional options (`WithAPIKey`, ...)                                    |
| `config.go`                          | The configuration lookup (options, `SMARTER_*` env, the cli's config file)    |
| `http.go`                            | The transport: auth, retries, error mapping, decoding, SSE                    |
| `errors.go`                          | The sentinels and error types (ADR-006)                                       |
| `sse.go`, `log.go`, `types.go`       | The SSE parser, logging, `JSONValue`/`JSONObject`/`ID`                        |
| `manifests.go`, `llmclients.go`, ... | One file per resource family                                                  |
| `models.go`, `manifest.go`           | Structs of the REST api's responses; `Manifest`, `ToSAM`                      |
| `chat.go`, `journal.go`, `react.go`  | The contract mirrors                                                          |
| `resource.go`                        | `KindService[T]` and `ReadService[T]`, which every resource service builds on |
| `sam/`                               | **Generated** SAM structs of every manifest kind. Never edit by hand          |
| `internal/cmd/samgen/`               | The SAM generator                                                             |
| `scripts/`                           | `sync-contracts.sh`                                                           |
| `testdata/`                          | Fixtures (shared with the Python SDK), `contracts/`, `sam/`                   |
| `*_test.go`                          | Unit tests (package `smarter_test`), `live_test.go`, `example_test.go`        |
| `doc/adr/`                           | The SDK's architecture decision records                                       |
| `skills/`                            | These skills                                                                  |

The SDK's guides live in the **platform's** repository:
`../smarter/docs/source/smarter-framework/smarter-golang*`, and its api
reference is pkg.go.dev. See `smarter-sphinx-docs` and `smarter-godoc`.

## Non-negotiable rules

1. **Never execute git write commands**, in this repository, in `../smarter`
   or in `../smarter-cli`. "Write a git commit" means output the command text.
   See `smarter-git-commits`.
2. **Run the tests yourself**, after every change: `make test` (`go test -race
./...`). Never ask the user to re-run them. Report failures honestly,
   including pre-existing ones.
3. **Validate against a real platform.** A struct that has only met a
   hand-written fixture is a guess. Probe the endpoint (`smarter-api-testing`),
   build sanitized fixtures from real responses (or reuse the Python SDK's
   fixture of the same response), and run `make test-live` against the local
   platform before you call an endpoint covered.
4. **Never deploy from a test.** `Deploy` creates real, billable cloud
   infrastructure (DNS, certificates, Kubernetes ingresses, GPU nodes). The live
   tests never call it. The local platform's containers hold live credentials:
   see `smarter-infrastructure-safety` in `../smarter/skills/`.
5. **The pre-commit hooks are the lint standard.** See `smarter-code-quality`.
6. **Coverage:** at least 90% of the `smarter` package, 95% where it takes
   moderate effort. CI fails below 90%.
7. **This repository is public.** No api keys, tokens, account numbers,
   private hostnames, names, addresses or customer data in code, fixtures,
   docs or skills. Sanitize every fixture taken from a real response, and grep
   it before you finish.
8. **No `any` outside the JSON aliases.** Every response is a struct (ADR-004).
   See `smarter-golang-types`.
9. **Every exported identifier has a Go doc comment** that starts with its
   name, and says what the Python SDK's docstring says. See `smarter-godoc`.
10. **A library never exits.** No `panic` on a caller's input, no `log.Fatal`,
    no `os.Exit`: return an error.

## Architecture, and why

Each decision below is recorded in `doc/adr/`, with the Python SDK ADR and the
platform ADR that it follows or adapts. Read the relevant ADR before you change
one of them, and add an ADR when you make a new decision of that size.

- **Writes go through the manifest api** (`/api/v1/cli/apply/`, `delete/`,
  ...), exactly as the cli sends them. The REST api is used only to **read**
  resources and for **actions** that the manifest api has no verb for (ADR-003).
- **Structs never fail on unknown fields**, values that the platform may add to
  are `string`s, optional numbers and booleans are pointers, timestamps are
  strings, and a manifest's `spec` is a `JSONObject` (ADR-004).
- **Identifiers are Go's, JSON tags are the wire's** names (ADR-021).
- **One package, functional options, params structs** (ADR-020); the generated
  structs are the `sam` package.
- **The platform's models can't be imported**, so:
  - the **SAM structs** are generated from the JSON Schema of each kind that the
    platform commits in `smarter/smarter/apps/api/v1/contract/sam/`
    (`make sam-models`, `internal/cmd/samgen`),
  - the **chat and journal contracts** are hand-mirrored in `chat.go` and
    `journal.go`, and `contracts_test.go` checks their JSON tags against the
    platform's committed schemas and examples (`make contracts`),
  - the **api** is published as `openapi.json`, and `contracts_test.go` checks
    that every request of the SDK is one of its operations, and that the SDK
    calls every operation,
  - the **React contract** is copied in `react.go`.
- **Configuration follows ADR-002**, and the cli: options, `SMARTER_*`
  variables, `~/.smarter/config.yaml`, then defaults. Only `config.go` reads the
  environment.
- **Logging** goes to the caller's `*slog.Logger`, and is off unless
  `SMARTER_LOG` or `WithLogger` asks for it. Api keys are masked with
  `MaskSecret()` (ADR-007, ADR-016).
- **The chat contract is always version 2**: every request sends
  `X-Smarter-Capabilities: chat-contract-v2`.
- **Every request method takes a `context.Context` first, and blocks**
  (ADR-018).
- **Errors** are sentinels for `errors.Is` and types for `errors.As` (ADR-006).

## Covering a platform endpoint (the pattern)

1. Find the view and its url in `../smarter/smarter/smarter/apps/<app>/api/v1/`,
   and the Python SDK's method for it, if any. Read both.
2. Probe it on the local platform with a throwaway api key
   (`smarter-api-testing`) and save the real response in your scratchpad.
3. Add or extend the struct in `models.go`: the fields that callers need, with
   `omitempty` JSON tags of the wire's names, following ADR-004.
4. Add the method to `<family>.go`, copying its neighbors:
   `s.c.http.do(ctx, &request{method: http.MethodGet, path: path, query: url.Values(q)}, &out)`,
   the path from `s.path(id, "suffix/")` (which validates the `ID`), optional
   arguments in a `<Name>Params` struct that may be nil, and a doc comment.
5. Add a sanitized fixture to `testdata/` (the Python SDK's, if it has one), a
   unit test in `<family>_test.go`, the call in `contracts_test.go`, and a
   check in `live_test.go` if it is safe (no deploy, no destructive change to
   existing data).
6. Run `make test`, `make test-live`, and the hooks.
7. Update the guide in `../smarter/docs/source/smarter-framework/smarter-golang/`
   (`smarter-sphinx-docs`).
8. If the Python SDK lacks the endpoint, say so in your summary.

A **new resource kind** on the platform usually needs: `make sam-models`, a
struct that embeds `Resource` in `models.go`, a service that embeds
`*KindService[T]`, a field on `Client` set in `NewClient`, and its tests and docs.

Two deliberate differences from the Python SDK: a name that starts with
`Smarter` drops it (`smarter.JournalMessage`, ADR-021), and
`Plugins.Retrieve` reads the plugin from the list, while
`Plugins.RetrieveManifest` is the Python SDK's `plugins.retrieve()`, because the
platform's `plugins/<id>/` returns a manifest.

## Platform bugs

The SDK sees the platform from outside, so it finds the platform's bugs. When
an endpoint misbehaves, don't paper over it silently: work around it in the SDK
only when the workaround is harmless (e.g. `Manifests.Get` filters by name
client-side, as the Python SDK does), say so in a comment, make the live test
skip with the platform's message rather than fail, and report the bug to the
user.

## Which skill to load

| Task                                                   | Skill                    |
| ------------------------------------------------------ | ------------------------ |
| Unit tests, live tests, fixtures, coverage             | `smarter-golang-testing` |
| Probing the platform's api by hand                     | `smarter-api-testing`    |
| Go doc comments and examples                           | `smarter-godoc`          |
| Types, `any`, pointers, the `types.go` aliases         | `smarter-golang-types`   |
| The SDK's guides in the platform's Sphinx docs         | `smarter-sphinx-docs`    |
| gofmt, go vet, golangci-lint, pre-commit               | `smarter-code-quality`   |
| Writing a commit message                               | `smarter-git-commits`    |
| Monthly sync with the platform, and dependency updates | `monthly-maintenance`    |

The platform's own skills (`../smarter/skills/`) apply whenever you change the
platform itself, and the Python SDK's (`../smarter-python/skills/`) when you
change the Python SDK, e.g. to keep it in step with a fix made here.

## Definition of done

- The change follows an existing pattern in this repository, its ADRs, and the
  Python SDK's behavior.
- Unit tests cover it, `make test` passes with the race detector, and coverage
  stays at 90% or more.
- `go vet`, golangci-lint and the pre-commit hooks pass on the changed files.
- The live tests pass against the local platform, or their skips name platform bugs.
- Exported identifiers have doc comments, and the guides in `../smarter/docs/source/`
  describe the feature.
- Your summary says what changed, what you verified, what you skipped, any
  platform bugs you found, and any difference from the Python SDK.
