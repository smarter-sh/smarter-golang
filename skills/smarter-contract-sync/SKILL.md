---
name: smarter-contract-sync
description: Use when the Smarter platform's api contract changed and this repository's copy of it must follow - a platform commit that changed openapi.json, a SAM kind's schema or example, or the chat or journal contract (smarter/smarter/apps/api/v1/contract/ in ../smarter), e.g. a new field in a REST response - outside the monthly maintenance, or when a contract test fails after a sync. Gives the exact steps - pick the platform's release tag, copy with SMARTER_REF, regenerate, read the diff, fix the recorded responses, test, lint the changed files, and write the commit - and lists the sibling repositories that hold the same copy and must be synced with it.
---

# Syncing the Platform's API Contract

This repository holds a copy of the Smarter platform's api contract:
`testdata/contracts/` (`openapi.json`, the chat and journal contracts), `testdata/sam/` (each kind's schema and example), and `sam/`, which `internal/cmd/samgen` generates from the schemas. The platform generates the contract from its Pydantic models
(`../smarter`, its `smarter-api-contract` skill), so the copy is never edited
by hand. It is copied, at a platform ref, and checked by this repository's
tests.

`monthly-maintenance` does this as one step of the monthly sync. Use this
skill for a sync between maintenances, when a single platform change (e.g. a
field added to an LLMClient's REST response) has to reach this repository now.

Load `smarter-golang-development` first. Its rules apply, in particular: never run git write
commands (write the commit as text, `smarter-git-commits`), and run the tests
yourself.

## A contract change reaches every sibling repository

The workspace has five repositories with a copy. A platform change to the
contract means syncing all of them, at the same ref, with one commit each:

| Repository            | Its copy                                                                        | Sync command                                                    |
| --------------------- | ------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| `smarter-cli`         | `testdata/contracts/openapi.json`, and the ref in `testdata/contracts/SOURCE`   | `make contracts SMARTER_REF=<tag>`                              |
| `smarter-python`      | `tests/data/contracts/`, `tests/data/sam/`, generated `smarter/models/sam/`     | `make sam-models SMARTER_REF=<tag>`                             |
| `smarter-nodejs`      | `test/data/contracts/`, `test/data/sam/`, generated `src/models/sam/`           | `make sam-models SMARTER_REF=<tag>`                             |
| `smarter-golang`      | `testdata/contracts/`, `testdata/sam/`, generated `sam/`                        | `make sam-models SMARTER_REF=<tag>`                             |
| `smarter-vscode-yaml` | `contract/openapi.json`, `contract/sam/`, generated `src/contract/openapi.d.ts` | `make contracts SMARTER_REF=<tag>` (also regenerates the types) |

## 1. Pick the platform ref

Sync with a released platform ref, never with `../smarter`'s working tree:

```console
git -C ../smarter fetch -q origin
git -C ../smarter describe --tags --abbrev=0 origin/main   # the latest stable release tag, e.g. v0.25.1
git -C ../smarter branch -r --contains <sha of the change>  # the change is pushed, and on origin/main
git -C ../smarter rev-parse <tag>^{commit}                # the full sha, for the commit body
```

- Use the latest release tag on `origin/main` that contains the change. A tag
  reads better in `SOURCE` and in commit messages than a sha.
- If the change is only on `alpha`, or not pushed yet, ask the user which ref
  to sync with.
- Check what else the tag brings:
  `git -C ../smarter diff --stat <last synced ref> <tag> -- smarter/smarter/apps/api/v1/contract`.

## 2. Copy, and regenerate

```console
make sam-models SMARTER_REF=v0.25.1
```

`sam-models` runs `contracts` first, then `go run ./internal/cmd/samgen` and `go test ./sam/`. **Pass `SMARTER_REF` to `sam-models` too**: without it, its `contracts` step copies `../smarter`'s working tree, which may hold unreleased work, over the files that you just copied at the tag.

Use the tag from step 1 in place of `v0.25.1`. When a later step reruns the
copy, pass the same `SMARTER_REF` again.

## 3. Read the diff

```console
git status --short
git diff --stat
git diff -- <each changed contract file>
```

**Expect more than the change you came for.** The copy brings every contract
change since the last sync. The sync of 2026-10-14, for three new LLMClient
fields, also brought enums on five Guardrail fields, schema descriptions that
the plugin-to-llmtool rename changed, and a reordered LLMClient example.
Classify each change, as the platform's `smarter-api-contract` skill does:

| Backward compatible                                 | Needs a change here                                                    |
| --------------------------------------------------- | ---------------------------------------------------------------------- |
| A new optional field, schema or kind                | A removed or renamed field, kind or operation                          |
| A changed description, a reordered example          | A new **required** response field (recorded responses lack it)         |
| A new enum on a field whose description listed them | A new operation (the operation-coverage test fails until it's handled) |

A new required field in a response is compatible for live use (every model
here accepts unknown fields), but breaks the recorded responses that the tests
validate. See step 4.

## 4. Test, and fix what fails

```console
make test
```

`contracts_test.go` fails when the document has an operation that the SDK doesn't call, or the SDK calls one that isn't in it. **It doesn't validate the recorded responses in `testdata/`**: a field that the platform made required passes here, and fails in smarter-python and smarter-nodejs.

The recorded responses (`llmclient.json`, `llmclients.json`, ...) are the same
files, byte for byte, in smarter-python (`tests/data/`), smarter-nodejs
(`test/data/`) and smarter-golang (`testdata/`). Python and Node validate them
against `openapi.json`. Go doesn't, so a stale fixture passes there and fails
in the other two. Change all three copies together, and check that they're
still identical:

```console
cd .. && md5 -q smarter-python/tests/data/<name>.json smarter-nodejs/test/data/<name>.json smarter-golang/testdata/<name>.json
```

When a recorded response fails with "must have required property" (Node) or
"is a required property" (Python), the platform added a field to that
response. Add it to the fixture with the value that the current platform
returns: probe the real endpoint (`smarter-api-testing`), or derive it from
the platform's serializer and contract model when the value follows from the
fixture's other fields (e.g. an LLMClient's `tools` from its `functions` and
`plugins`). Only add keys. Keep the platform's key order and the file's
two-space indentation, so that the diff shows only the new fields.

Typing a new field in this repository's own models is SDK work, not part of
the sync. Leave it for a separate `feat` commit, and say so in your summary.

## 5. Lint the changed files

```console
git diff --name-only | xargs pre-commit run --files
```

If a hook reformats a copied contract file, the hook's excludes are wrong: fix the exclude, and don't commit the reformatted copy.

## 6. Commit

Write the commit command (never run it), following `smarter-git-commits`.
`fix(contracts): sync with smarter <tag>` when generated code or fixtures changed. End the body with `Synced with smarter <full sha>`, which `monthly-maintenance`'s `smarter-delta.sh` reads as its next baseline.

The body says which platform commit or release the sync follows, lists what
the copy brought (the change you came for, and everything else from step 3),
the fixtures that changed and why, and the test results.

## Don'ts

- Don't run `make contracts` or `make sam-models` without `SMARTER_REF`.
- Don't edit the copied files or the generated code by hand. If a copied file
  is wrong, the fix is in the platform: regenerate it there
  (`make api-contract`), commit it, and sync again.
- Don't sync one repository and leave its siblings behind: the SDKs, the cli
  and the extension should describe the same platform.
