# smarter-golang Agent Skills

This folder holds the [Agent Skills](https://agentskills.io/specification) for
working on the Smarter Go SDK with an AI coding agent such as Claude Code.
This README is written mostly for people.

A skill is a folder of written know-how that an agent loads only when a task
needs it. Each one contains a `SKILL.md` file, and sometimes `references/` and
`scripts/` folders. These skills cover how this repository is built, tested,
documented and committed. They are developer tooling, like the `Makefile` and
`.pre-commit-config.yaml`, and no SDK code loads them.

They are adapted from the Smarter Python SDK's skills, in
[smarter-sh/smarter-python/skills](https://github.com/smarter-sh/smarter-python/tree/main/skills),
which are in turn adapted from the platform's own, in
[smarter-sh/smarter/skills](https://github.com/smarter-sh/smarter/tree/main/skills),
so that the platform and its SDKs are developed the same way. The Python SDK is
the reference implementation of every Smarter SDK, so these skills tell an
agent to check it before changing a feature here. The smarter cli, also written
in Go, shares this SDK's configuration file and lint standard. Load the platform's skills
from `../smarter/skills/` when an SDK task leads you into the platform itself,
e.g. to fix a bug that the SDK found.

## What's here

Start with `smarter-golang-development`. It sets out the rules that apply to
every change, the SDK's architecture, and which narrower skill to load for each
kind of task.

| Skill                        | Use it when                                                                         |
| ---------------------------- | ----------------------------------------------------------------------------------- |
| `smarter-golang-development` | You're making any change. Start here.                                               |
| `smarter-golang-testing`     | You're writing or running the unit or live tests, or raising coverage.              |
| `smarter-api-testing`        | You're probing the platform's api by hand, e.g. to build a fixture.                 |
| `smarter-godoc`              | You're writing doc comments or examples, which are the reference on pkg.go.dev.     |
| `smarter-golang-types`       | You're writing types, especially with `any`, pointers or assertions.                |
| `smarter-sphinx-docs`        | You're editing the SDK's pages in the platform's `docs/source`.                     |
| `smarter-code-quality`       | You're about to finish a change and need to run the hooks.                          |
| `smarter-git-commits`        | You've been asked to write a commit message.                                        |
| `monthly-maintenance`        | It's time to sync the SDK with the platform and the Python SDK, and update deps.    |
| `smarter-contract-sync`      | The platform's api contract changed, and this repository's copy must follow it now. |

The bundled scripts are ordinary shell scripts that you can run yourself from
the repository root:

- `smarter-sphinx-docs/scripts/check-sdk-pages.sh` builds only the SDK's pages
  of the platform's docs, in seconds, and reports their warnings.
- `smarter-sphinx-docs/scripts/check-examples.sh` compiles every Go example of
  those pages against this checkout.
- `monthly-maintenance/scripts/smarter-delta.sh` lists the platform's commits
  since the SDK's last release, grouped by what they mean for the SDK, and the
  Python SDK's commits since then.
- `monthly-maintenance/scripts/test-minimum-go.sh` runs the unit tests with
  the lowest Go version that `go.mod` allows.

## Claude Code

Claude Code looks for project skills in `.claude/skills/`. This repository
connects the two folders with a committed symlink, `.claude/skills -> ../skills`,
as the platform's and the Python SDK's repositories do:

```bash
mkdir -p .claude
ln -s ../skills .claude/skills
```

Other agents that read `SKILL.md` folders can be pointed at this folder, or told
to read `skills/smarter-golang-development/SKILL.md` at the start of a session.

## When to add or change a skill

Add to a skill when you've explained the same thing to an agent twice, when a
mistake is costly (e.g. a test that deploys real infrastructure), when a
pattern isn't obvious from the code, or when checking a change takes a fragile
series of steps that a script could run. Don't add one for what the code, the
linters or CI already enforce, for one-off facts, or for personal preferences.
Prefer editing an existing skill to adding one, and keep each `SKILL.md` short.

When the platform or the Python SDK changes a convention that a skill here
copies, update the skill here too.

## Security considerations

A skill is a set of instructions that an agent follows, often without asking.
**Review a change to this folder as carefully as a change to CI.**

- **The local platform is live.** The SDK's live tests and api probes run
  against the platform in `../smarter`, whose containers hold credentials for
  real cloud infrastructure. The skills forbid deploying from tests and probes.
  Don't weaken that.
- **Some instructions forbid things on purpose**, e.g. never running
  `git commit`. Don't remove a rule like that unless you mean to give agents
  that power.
- **Skills can carry prompt injection.** Look for instructions unrelated to a
  skill's purpose, new scripts with network calls, and text that tells an agent
  to skip confirmations, hooks or safety rules.
- **Keep secrets out.** This repository is public: no api keys, tokens, account
  numbers, private hostnames or customer data in a skill, an example or a script.
