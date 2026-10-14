---
name: smarter-sphinx-docs
description: Use when adding or editing the smarter-golang SDK's guides, which live in the Smarter platform's Sphinx docs (../smarter/docs/source/smarter-framework/smarter-golang*, published to docs.smarter.sh). Covers where each page goes, why the api reference is pkg.go.dev and not a generated page, the evergreen-prose rule, and how to verify without the platform's 20-minute build.
---

# The SDK's Sphinx Documentation

The SDK has no docs folder of its own. Its guides are a section of the
platform's docs, in the platform's repository, beside the other SDKs', so that
docs.smarter.sh is one site. Its api reference is pkg.go.dev (ADR-012):

| Content                                        | Location                                                                                        |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| The SDK's landing page: what it is, SDK vs CLI | `../smarter/docs/source/smarter-framework/smarter-golang.rst`, in `apis-and-sdks.rst`'s toctree |
| Guides: configuration, manifests, chat, ...    | `../smarter/docs/source/smarter-framework/smarter-golang/<topic>.rst`                           |
| The api reference                              | pkg.go.dev, from the doc comments (`smarter-godoc`). Link it; don't copy it                     |
| The SDK's own architecture decisions           | `doc/adr/` in this repository                                                                   |

The guides mirror the Python SDK's guides
(`smarter-framework/smarter-python/<topic>.rst`): the same topics, in the same
order, with the same examples in Go. When you change one, check whether the
others need the same change.

Load the platform's own `smarter-sphinx-docs` skill (`../smarter/skills/`) for
its general rules: headings (`=`, `-`, `~`, `^`), toctrees, code blocks, and
writing for a reader new to Smarter.

## Writing rules

- **Evergreen prose**, as in the platform: no "New in 0.2" sections. Weave a
  feature into the page where its topic belongs.
- Every example must compile: [scripts/check-examples.sh](scripts/check-examples.sh)
  builds every `go` block of the SDK's pages against this checkout (a page may
  use `client`, `ctx`, `llmclient`, `guardrail`, `mcpclient`, `store`,
  `manifest` and `chat` without declaring them; a block is the body of a
  function that returns an `error`). Prefer copying an `Example` function from
  `example_test.go`, which `go test` checks. Check SAM field paths against
  the generated `sam` package (e.g. an LLMClient's provider is
  `Spec.Config.Prompt.Provider`).
- Examples use `.. code-block:: go`, and handle every error.
- Link an identifier's reference as
  `` `Client.Manifests <https://pkg.go.dev/github.com/smarter-sh/smarter-golang#Client>`__ ``.
- Link the platform's resource pages with `:doc:`, e.g.
  `:doc:`Guardrails <../../smarter-resources/smarter-guardrail>``.
- Say when an operation is **billable** (prompts) or creates **real
  infrastructure** (deploy).

## Verify (never the full build)

1. Links, with the platform's checker, from `../smarter`:

   ```console
   cd ../smarter && skills/smarter-sphinx-docs/scripts/check-docs.sh \
     docs/source/smarter-framework/smarter-golang.rst docs/source/smarter-framework/smarter-golang/*.rst
   ```

2. The reST itself, with a build of **only the SDK's pages**, which takes
   seconds: [scripts/check-sdk-pages.sh](scripts/check-sdk-pages.sh). It reports
   every warning except links to pages outside the SDK's section. It must report
   none.

3. The examples: [scripts/check-examples.sh](scripts/check-examples.sh). It
   must say `ok`.

The user builds the full docs.
