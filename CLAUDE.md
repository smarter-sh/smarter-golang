# CLAUDE.md

Guidance for Claude Code (and other coding agents) working in this repository.

smarter-golang is the Smarter platform's official Go SDK: the module
`github.com/smarter-sh/smarter-golang`, package `smarter`. It is to the Smarter
platform ([smarter-sh/smarter](https://github.com/smarter-sh/smarter), usually
checked out beside this repository as `../smarter`) what `openai-go` is to the
OpenAI api: the library that Go programmers import to build software on their
own Smarter installation, without customizing or forking the platform. The
[Python SDK](https://github.com/smarter-sh/smarter-python) (`../smarter-python`)
is the reference implementation of every Smarter SDK: this SDK has its
features, resources, verbs and errors, in idiomatic Go. The
[smarter cli](https://github.com/smarter-sh/smarter-cli) (`../smarter-cli`) is
also Go, shares this SDK's configuration file, and should be able to import it.

Detailed know-how lives in the Agent Skills under `skills/` (symlinked to
`.claude/skills`). **Load `smarter-golang-development` first for any change.**
This file holds only what applies to nearly every task.

## Hard rules

1. **Never run git write commands**, here, in `../smarter`, `../smarter-python`
   or `../smarter-cli`. "Write a git commit" means print the command as text.
   See `smarter-git-commits`.
2. **Run the tests yourself** after any change (`make test`), and the live
   tests against the local platform when you change a struct or an endpoint.
   Never ask the user to re-run them. Report failures honestly.
3. **Never deploy from a test or a probe.** Deploy creates real, billable
   infrastructure. The local platform's containers hold live credentials.
4. **The pre-commit hooks are the lint standard** (gofmt, go vet, goimports,
   golangci-lint, as in the cli). See `smarter-code-quality`.
5. **Coverage:** at least 90%, 95% where it takes moderate effort.
6. **This repository is public.** No api keys, tokens, account numbers,
   private hostnames, names or customer data in code, fixtures, docs or skills.
7. **Check the Python SDK first.** Before you add or change a feature, read how
   `../smarter-python` does it, and keep the two in step.
8. **A library never exits**: no `panic` on input, no `log.Fatal`, no `os.Exit`.
9. **Live prompt calls are fine.** Live tests and probes send billable LLM
   prompts; that is accepted.

## Repository map

| Path                                                            | Contents                                                                |
| --------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `client.go`, `options.go`, `config.go`                          | `Client`, `NewClient`, the functional options, the configuration lookup |
| `http.go`, `errors.go`, `sse.go`, `log.go`, `types.go`          | Transport, errors, SSE, logging, type aliases                           |
| `<family>.go`                                                   | One file per resource family: `manifests.go`, `llmclients.go`, ...      |
| `models.go`, `manifest.go`, `chat.go`, `journal.go`, `react.go` | Response structs, `Manifest`, and the contract mirrors                  |
| `sam/`                                                          | Generated SAM structs. Never edit by hand                               |
| `internal/cmd/samgen/`                                          | The SAM generator                                                       |
| `testdata/`                                                     | Fixtures (shared with the Python SDK), contracts, SAM schemas           |
| `doc/adr/`                                                      | The SDK's architecture decision records                                 |
| `skills/`                                                       | Agent Skills                                                            |

The api reference is pkg.go.dev. The SDK's guides live in the platform's
repository: `../smarter/docs/source/smarter-framework/smarter-golang*`.

## Common commands

```bash
make test                                   # go test -race ./...
make coverage                               # with coverage, fails below 90%
go vet ./... && golangci-lint run
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=... make test-live
make contracts                              # copy the platform's contracts, SAM schemas and openapi.json from ../smarter
make sam-models                             # make contracts, then regenerate sam/
source .venv/bin/activate && git diff --name-only | xargs pre-commit run --files
```

Monthly, sync the SDK with the platform, the Python SDK and the cli, and update
its toolchain: load the `monthly-maintenance` skill.

## Definition of done

- The change follows an existing pattern, the ADRs in `doc/adr/`, and the Python SDK's behavior.
- Tests cover it, you ran them with `-race`, and the live tests pass or skip only for named platform bugs.
- `go vet`, golangci-lint and the pre-commit hooks pass on the changed files.
- Exported identifiers have doc comments; user-facing features are documented in `../smarter/docs/source/`.
- Your summary says what changed, what you verified, what you skipped, any
  platform bugs found, and any difference from the Python SDK.
