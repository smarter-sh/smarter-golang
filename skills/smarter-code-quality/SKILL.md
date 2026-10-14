---
name: smarter-code-quality
description: Use before finishing any change to the smarter-golang SDK, to format, vet and lint it with the repository's pre-commit hooks (go-fmt, go-vet, go-imports, golangci-lint, go-unit-tests, codespell, Prettier for Markdown and YAML, the generic hooks, commitlint), which match the smarter cli's. Explains which tools are authoritative, how to run them on changed files under zsh, and the traps (generated sam code, testdata).
---

# Smarter Go SDK Code Quality

`.pre-commit-config.yaml` at the repository root is the authoritative
standard. Its Go hooks are the smarter cli's (`../smarter-cli/.pre-commit-config.yaml`),
and its generic hooks the platform's. If a hook passes, the code meets the
standard. If it fails, fix the code. Never bypass a hook (`--no-verify`,
`SKIP=`, a bare `//nolint`).

## What the hooks enforce

| Hook          | Configuration               | Notes                                                         |
| ------------- | --------------------------- | ------------------------------------------------------------- |
| go-fmt        |                             | `gofmt`. Let it reformat.                                     |
| go-imports    |                             | Import grouping: standard library, then others.               |
| go-vet        |                             | `go vet ./...`.                                               |
| golangci-lint | `.golangci.yml`             | The standard linters plus revive, gosec, errorlint, misspell. |
| go-unit-tests |                             | `go test ./...`: a commit can't break the tests.              |
| codespell     | `codespell.txt` ignore list | Skips `testdata/`. Add real project words to `codespell.txt`. |
| prettier      | `.prettierrc.json`          | Markdown, YAML and JSON. Skips `testdata/`.                   |

Also: trailing whitespace, end-of-file newline, YAML/JSON syntax, merge
conflict markers, private keys, AWS credentials, and commitlint on commit
messages.

## How to run

pre-commit lives in a Python virtual environment, `.venv/`, which `make init`
(or `make python-init`) creates from `requirements/local.txt`, and which also
installs the pre-commit and commit-msg hooks. If `.venv/` is missing, or
`.git/hooks/pre-commit` points at a Python that no longer exists, run
`make python-init`. The Go hooks also need `goimports` and `golangci-lint` on
`PATH`. From the repository root:

```console
gofmt -l . && go vet ./... && golangci-lint run
source .venv/bin/activate
git diff --name-only | xargs pre-commit run --files
git ls-files --others --exclude-standard | xargs pre-commit run --files   # new files
```

- Use `xargs`. Under zsh, `pre-commit run --files $FILES` passes one argument,
  and pre-commit checks nothing.
- Hooks that modify files fail the first run, then pass. Run them until they
  pass, and read the diff of what they changed.
- `sam/` is generated, and gofmt'd by the generator. Don't hand-edit it to
  satisfy a linter: change `internal/cmd/samgen` instead. `.golangci.yml`
  excludes it from the style linters.
- `.golangci.yml` relaxes gosec, errorlint and usestdlibvars in `_test.go`
  files only: tests read their own `testdata/`, and status codes read better as
  numbers there.
- CI checks that `sam/` is exactly what `go run ./internal/cmd/samgen`
  generates: regenerate, never patch.
- `make lint` runs every hook on every file.

## Principles

- Match the surrounding code: its naming, comment density, logging (through the
  client's `*slog.Logger`, never the api key), and the errors of `errors.go`.
- Wrap errors with `%w`, compare them with `errors.Is` and `errors.As`
  (errorlint enforces it).
- Fix the cause of a finding. Suppress one only with a narrow directive that
  names the linter and says why, on the line it concerns
  (`//nolint:gosec // G404: jitter, not security`).
- Don't reformat code you didn't otherwise change.
