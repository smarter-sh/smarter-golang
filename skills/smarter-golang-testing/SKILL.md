---
name: smarter-golang-testing
description: Use when writing, fixing or running the smarter-golang SDK's tests, or raising its coverage. Covers the go test unit tests (newMockPlatform, routes answered by a RoundTripper, sanitized fixtures in testdata shared with the Python SDK), the opt-in live tests against a running Smarter platform, the contract and SAM tests, runnable examples, coverage, and what never to do in a test.
---

# Smarter Go SDK Testing

## Two kinds of tests

| Kind | Where            | Reaches a platform? | Runs in CI | Command                                     |
| ---- | ---------------- | ------------------- | ---------- | ------------------------------------------- |
| Unit | `<file>_test.go` | Never               | Yes        | `make test` (`go test -race ./...`)         |
| Live | `live_test.go`   | Yes                 | No         | `make test-live` (`SMARTER_TEST_LIVE=true`) |

Run them on the host, from the repository root:

```console
go test -race ./...                         # every unit test, and the examples
go test -run TestGuardrails ./...           # tests whose name matches
go test -run 'TestGuardrails/evaluate' .    # one subtest
make coverage                               # with coverage, fails below 90%
```

Run the tests yourself, after every change, and report the counts
(`go test -json ./... | ...`, or `-v` and count `--- PASS`/`--- FAIL`/`--- SKIP`).

## Unit tests

- One test file per source file, mostly: `llmclients.go` is tested by
  `llmclients_test.go`, and the small resource services share
  `resources_test.go`. Test files are in the external package `smarter_test`, so
  that they use only the public api, as callers do. Unexported helpers whose
  behavior the public api can't reach easily (the error message shapes, the
  retry delay, the SSE parser) are tested in `http_internal_test.go`, in package
  `smarter`.
- Build the client with `newMockPlatform(t)` from `helpers_test.go`, the
  counterpart of the Python SDK's `SmarterTestBase`. Its `client` is a
  `*smarter.Client` whose `*http.Client` has a `RoundTripper` double: nothing
  leaves the process. It also hides the developer's `SMARTER_*` variables and
  `~/.smarter/config.yaml`: its client reads the environment through
  `smarter.WithLookupEnv(testEnv(...))`, which sees only the variables that the
  test gives it, and a configuration file that doesn't exist. A local
  configuration can't change a result.
- Register answers with
  `p.route("GET", "guardrails/", reply{fixture: "guardrails"})`; a `reply` also
  takes `json`, `status`, `text`, `header` and `handler`. `path` is relative to
  `/api/v1/`. An unrouted request gets a 404.
- Assert what was sent with `p.last()` (the `*http.Request`, with its `body`),
  `p.lastJSON()`, `p.last().URL.Query()` and `p.count()`. The helpers
  `check(t, err)`, `equal(t, got, want)` and `jsonEqual(t, got, want)` keep
  assertions short.
- `p.makeClient(t, smarter.WithMaxRetries(2))` builds another client on the
  same routes. A test that retries answers with a `Retry-After: 0` header, so
  that it doesn't wait.
- Assert errors with `errors.Is(err, smarter.ErrNotFound)` and `errors.As`.
  The standard `testing` package only: no testify (ADR-011).
- Table-driven tests with `t.Run` for variations of one behavior.

## Examples

`example_test.go` holds runnable `Example...` functions, which pkg.go.dev
shows and `go test` runs. They can't take a `*testing.T`, so they use
`examplePlatform()`, an `httptest.Server` that answers with the files of
`testdata/`, and their `// Output:` comments are checked. Add one for every
common task (ADR-012).

## Fixtures

- Fixtures are **real platform responses**, trimmed and sanitized, in
  `testdata/<name>.json` (and `prompt_sse.txt` for a stream). They are the same
  files as the Python SDK's `tests/data/`: copy a new one from there, or add it
  to both. Don't invent a response shape: probe the endpoint first
  (`smarter-api-testing`).
- **Sanitize before saving.** Replace names, emails, addresses, phone numbers,
  account numbers, api keys (even masked ones), tool call ids and private
  hostnames with placeholders (`Example Company`, `user@example.com`,
  `1234-5678-9012`). Then grep `testdata/` for what you replaced. The
  repository is public.
- `testdata/contracts/` and `testdata/sam/` are copies of what the platform
  publishes. Refresh them with `make contracts`, never by hand.

## Contract and SAM tests

- `contracts_test.go` fails when the platform's chat or journal contract has a
  property that the mirror's JSON tags don't declare (it maps the platform's
  model names to the Go types in a table: add a new mirror there), when a
  platform example doesn't decode, when the SDK sends a request that isn't an
  operation of `openapi.json`, or when `openapi.json` has an operation that the
  SDK doesn't call and that isn't in `notCalled` (with its reason).
- `sam/sam_test.go` fails when a kind's example manifest doesn't decode into
  its struct with `DisallowUnknownFields`, or a kind has no struct. Regenerate
  (`make sam-models`), don't edit the generated code.

## Live tests

```console
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=<throwaway key> make test-live
```

- They decode the platform's real responses into every struct, including
  `Describe` of a real resource of every kind decoded strictly as its SAM struct.
- They prompt an LLMClient (`SMARTER_TEST_LLMCLIENT`, default `calculator`),
  which makes **billable LLM requests**. That is accepted in this project.
- They apply and delete a throwaway Guardrail named `sdk_live_test_<hash>`,
  with `t.Cleanup`. Anything a live test creates, it deletes.
- **They never deploy, undeploy, snapshot, restore, or delete existing
  data.** Deploy creates real, billable infrastructure.
- A platform error (an `*APIStatusError`) skips the subtest with the platform's
  message (`t.Skipf`). An SDK error (e.g. `*APIResponseValidationError`)
  fails it. Read the skip reasons: they are platform bugs to report.

Create and delete the throwaway api key as `smarter-api-testing` describes.

## Coverage

At least 90% of the `smarter` package (CI enforces it), and 95% where that
takes moderate effort. The generated `sam` package and `internal/cmd/samgen`
are tested by their own tests, and not counted.

## Reporting

Report passed, failed and skipped counts honestly, for both suites. Name the
live tests' skips, and separate platform bugs from SDK failures.
