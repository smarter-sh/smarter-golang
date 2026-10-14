---
name: monthly-maintenance
description: Use when asked to do smarter-golang's monthly maintenance, or either half of it - (1) bring the SDK back in sync with the Smarter platform, the Python SDK (the reference implementation) and the smarter cli's configuration by reviewing their commits since smarter-golang's last release and changing the SDK, its models, tests and docs as needed, and (2) update the SDK's Go module dependencies and toolchain, retest, and, if the tests pass, write a "fix" git commit. Covers finding the baseline, reading the delta by area, the mechanical refreshes (SAM structs, contracts), the dependency and Go toolchain update and its tests, and the commit messages.
---

# smarter-golang Monthly Maintenance

The SDK covers a platform that changes every week, and follows a Python SDK
that changes with it, and a cli whose configuration it shares. Once a month, this maintenance brings it back in step,
and keeps its dependencies current. It has two parts, done in this order, each
with its own commit:

1. **Sync with the platform and the Python SDK** (Part 1).
2. **Update the dependencies and toolchain, and retest** (Part 2), so that
   the retest covers the sync too.

Load `smarter-golang-development` first. Its rules apply throughout, in
particular: **never run git write commands** (write the commit command as text,
see `smarter-git-commits`), **never deploy from a test or probe**, and run the
tests yourself.

Work from the smarter-golang repository root, with the platform checked out at
`../smarter` and its local dev server running (`../smarter`: `make run`; the
platform's `smarter-docker-environment` skill), the Python SDK at
`../smarter-python`, and the cli at `../smarter-cli`.

## Part 1: Sync with the platform and the Python SDK

### 1. Find the delta

```console
skills/monthly-maintenance/scripts/smarter-delta.sh
```

It fetches the platform (`git fetch` only updates its remote-tracking
branches), and lists the platform's commits on `origin/main`, its released
branch, since a baseline, grouped by what they mean for the SDK, with a diff
stat per area. Its last sections list the Python SDK's and the smarter cli's commits since
the same date. The baseline is the later of:

- the platform's commit at the date of smarter-golang's **last release tag**,
  and
- the commit named by the newest `Synced with smarter <sha>` line in
  smarter-golang's commit history, which every maintenance sync records (see
  step 5).

Pass an explicit base as the third argument to override it:
`smarter-delta.sh ../smarter origin/main <sha>`.

If the platform's release that users run is not `origin/main`, ask the user
which ref to sync with.

### 2. Review each area

For each area with changes, read the diffs, not only the subjects:

```console
git -C ../smarter diff <base>..origin/main -- <path>
git -C ../smarter show <sha>
git -C ../smarter-python show <sha>
```

| Area in the report                  | What to check, and what to change in the SDK                                                                                               |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Manifest (cli) api                  | New or changed verbs, parameters, responses: `manifests.go`, `client.go` (`WhoAmI`, `Version`, `Kinds`)                                    |
| REST urls and views                 | New endpoints to cover, changed paths, parameters, bodies or permissions: `<family>.go`                                                    |
| Serializers                         | Changed REST response fields: `models.go`, and the fixtures in `testdata/`                                                                 |
| SAM manifest models and kinds       | A new, renamed or removed kind, or a changed spec: `make sam-models SMARTER_REF=origin/main`, then a resource class for a new kind         |
| Chat and journal contracts, OpenAPI | `make contracts SMARTER_REF=origin/main`, then fix `chat.go`, `journal.go` or the resources until `contracts_test.go` passes               |
| Django-React contract               | Copy the change into `react.go`                                                                                                            |
| Authentication                      | Headers, schemes, token formats: `http.go`                                                                                                 |
| Platform settings that gate apis    | e.g. a new `SMARTER_ENABLE_*` switch: the docs, and the live tests' skips                                                                  |
| ADRs                                | A new or changed platform ADR that the SDK follows or adapts: `doc/adr/`                                                                   |
| Skills                              | A change to a platform skill that a skill here adapts: the adapted copy in `skills/`                                                       |
| Lint standard                       | The platform's generic hooks: `.pre-commit-config.yaml`                                                                                    |
| The SDK's own docs pages            | Someone edited them in the platform: keep the edit, and make sure its examples still compile                                               |
| The Python SDK                      | Every feature, fix, model change, ADR or skill change there: port it here, or say why not                                                  |
| The smarter cli                     | A change to its configuration lookup (`cmd/root.go`, `cmd/api.go`): `config.go`. Its Go version or Go hooks: `.pre-commit-config.yaml`, CI |

Follow the pattern in `smarter-golang-development` ("Covering a platform
endpoint") for every change: probe the real endpoint (`smarter-api-testing`),
model it, add a sanitized fixture and a unit test, and a live test when it is
safe.

Notes:

- **Everything comes from the platform's committed contract**, in
  `../smarter/smarter/smarter/apps/api/v1/contract/` (the SAM schemas and
  examples, and `openapi.json`), plus the chat and journal contracts. Pass the
  ref you sync with, e.g. `SMARTER_REF=origin/main`, so that the files are read
  with `git show` at that ref, not from `../smarter`'s working tree.
- **`contracts_test.go` fails when the platform adds an operation** that the
  SDK doesn't call. Cover it, or add it to `notCalled` with a reason, the same
  reason as the Python SDK's.
- **Removals and renames are breaking changes** for the SDK's users. Keep the
  old name with a `// Deprecated:` comment for a release when that is cheap,
  and say so in the commit body.
- **Not every platform change concerns the SDK.** Internal refactors, Django
  views of the web console, React apps, migrations, Celery tasks and
  infrastructure are out of scope unless they change the api.

### 3. Refresh what is generated

```console
make sam-models SMARTER_REF=origin/main    # copies every contract, then regenerates sam/
skills/smarter-sphinx-docs/scripts/check-sdk-pages.sh
```

The api reference is pkg.go.dev, which updates itself from the next release
tag. Update the SDK's guide pages in
`../smarter/docs/source/smarter-framework/smarter-golang/` for any user-visible
change (`smarter-sphinx-docs`).

### 4. Test

```console
make test
go vet ./... && golangci-lint run
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=<throwaway key> make test-live
git diff --name-only | xargs pre-commit run --files
git ls-files --others --exclude-standard | xargs pre-commit run --files
```

Read the live tests' skip reasons. A new skip is either a platform bug to report
or an SDK gap to fix.

### 5. Commit

Write the commit command (never run it), following `smarter-git-commits`. Its
type follows the change: `feat` when the SDK covers something new, `fix` when it
corrects drift. The body lists the platform and Python SDK commits that each
change follows, and ends with this line, which the next run's baseline reads:

```text
Synced with smarter <full sha of the platform ref you synced with>
```

Get the sha with `git -C ../smarter rev-parse origin/main`. Changes in
`../smarter` (the SDK's docs) get their own commit command, for that repository.

If nothing in the delta concerns the SDK, there is no commit. Report that, with
the range you reviewed.

## Part 2: Update the dependencies and toolchain, and retest

### What to update

| What                            | Where                                      | How it is updated                                                                          |
| ------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Module dependencies             | `go.mod`, `go.sum`                         | Raise one only for a security fix, a bug fix the SDK needs, or a feature it uses (ADR-011) |
| The minimum Go (`go` directive) | `go.mod`                                   | Raise it when a Go release reaches end-of-life (ADR-019), as a breaking change             |
| The Go versions that CI tests   | `.github/workflows/test.yml`               | The two supported Go releases                                                              |
| golangci-lint and the Go hooks  | `.pre-commit-config.yaml`, the CI workflow | The cli's versions, or newer                                                               |
| semantic-release and commitlint | `package.json`, `package-lock.json`        | Pinned; each to its latest version                                                         |

Go's minimal version selection makes a library's requirements its callers'
minimums, so the SDK doesn't chase the newest module versions: it raises a
requirement only when it has a reason, and says which.

### 1. See what is outdated

```console
go list -m -u all | grep '\['            # modules with a newer version
govulncheck ./...                        # known vulnerabilities (go install golang.org/x/vuln/cmd/govulncheck@latest)
npm outdated                             # the release tooling
```

Check the supported Go releases at https://go.dev/doc/devel/release, and the
cli's Go version and hooks in `../smarter-cli`.

### 2. Update

- A module with a vulnerability that `govulncheck` reports as reachable, or a
  fix that the SDK needs: `go get <module>@<version>`, then `go mod tidy`.
- A Go release that reached end-of-life: raise `go.mod`'s `go` directive to the
  oldest supported release (`go mod edit -go=1.27.0`), and the CI matrix.
- golangci-lint and the hooks: the versions in `.pre-commit-config.yaml` and
  the workflow, at least the cli's. The hooks' own versions are bumped weekly
  by the `precommitVersionBumps` workflow.
- The release tooling in `package.json`: each pin to its latest version, then
  `npm install`.

### 3. Retest

```console
make test                                                  # the installed Go
skills/monthly-maintenance/scripts/test-minimum-go.sh      # the lowest Go that go.mod allows
go vet ./... && golangci-lint run
govulncheck ./...
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=<throwaway key> make test-live
```

If a test fails:

- fix the SDK when the fix is small and clearly right (e.g. a deprecation),
- otherwise keep the previous version, retest, and report the failure, with
  its output.

### 4. Commit

Only when every test passes, write the commit command (never run it), following
`smarter-git-commits`:

```console
git commit --author="Claude <noreply@anthropic.com>" \
  -m "fix(deps): update the go toolchain and dependencies" \
  -m "Monthly maintenance: the SDK is tested on the supported Go releases, its module requirements have no known vulnerabilities, and its tooling is current." \
  -m "go.mod:" \
  -m "- go.yaml.in/yaml/v3: no change, no newer version is needed
- ..." \
  -m "Tooling:" \
  -m "- golangci-lint v2.14.0 -> v2.15.0
- ..." \
  -m "Tests: make test (n passed), test-minimum-go.sh on Go 1.26.0 (n passed), make test-live (n passed, n skipped for platform bugs), go vet, golangci-lint and govulncheck." \
  -m "Co-authored-by: Lawrence McDaniel <lpm0073@gmail.com>"
```

`fix` makes semantic-release tag a patch release. List every version change,
and the real test counts. If nothing was outdated, there is no commit: report
that.

## Report

End with a summary for the user:

- the platform, Python SDK and cli ranges that were reviewed, and what changed in
  the SDK because of them,
- the changes that were deliberately not followed, and why,
- platform bugs found (new live test skips), and differences from the Python SDK,
- the dependency and toolchain updates, and any that were held back, with the failing output,
- the test results of every suite,
- the commit commands, one per repository and part.
