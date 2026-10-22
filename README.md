# The Smarter Go SDK

![Test Status](https://github.com/smarter-sh/smarter-golang/actions/workflows/test.yml/badge.svg?branch=main)
[![Go coverage](https://img.shields.io/codecov/c/github/smarter-sh/smarter-golang/main?flag=go&label=coverage&logo=codecov)](https://codecov.io/gh/smarter-sh/smarter-golang)
[![Latest Release](https://img.shields.io/github/v/release/smarter-sh/smarter-golang?label=release)](https://github.com/smarter-sh/smarter-golang/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/smarter-sh/smarter-golang.svg)](https://pkg.go.dev/github.com/smarter-sh/smarter-golang)
[![Docs](https://img.shields.io/badge/Read%20the%20Docs-smarter.sh-blue?logo=readthedocs)](https://docs.smarter.sh/en/latest/smarter-framework/smarter-golang.html)
[![Website](https://img.shields.io/badge/official%20web%20site-smarter.sh-blue?logo=google-chrome)](https://smarter.sh)
[![License: GNU AGPL v3](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)<br>

`github.com/smarter-sh/smarter-golang` is the official Go SDK of the
[Smarter platform](https://github.com/smarter-sh/smarter). It is to Smarter
what `openai-go` is to the OpenAI api: one `go get` gives your Go software
typed, complete access to your own Smarter installation, whether it runs on
your laptop or on Kubernetes.

Use it to build custom software that is tightly integrated with your Smarter
resources and functionality, without customizing or forking the Smarter
backend: a service that chats with your LLMClients, a Kubernetes operator or
CI job that applies your manifests, a pipeline that loads documents into a
vectorstore, or a tool that calls your MCP servers' tools.

```go
package main

import (
	"context"
	"fmt"
	"log"

	smarter "github.com/smarter-sh/smarter-golang"
)

func main() {
	client, err := smarter.NewClient() // SMARTER_API_KEY, or the smarter cli's ~/.smarter/config.yaml
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	llmclient, err := client.LLMClients.Find(ctx, "customer_support")
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.LLMClients.Prompt(ctx, llmclient.Ref(), smarter.Text("Where is my order #1234?"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Text())
}
```

It has the same features as the [Smarter Python SDK](https://github.com/smarter-sh/smarter-python),
which is the reference implementation of every Smarter SDK, with the same
resources, verbs and errors, in idiomatic Go.

## Smarter Go SDK or Smarter CLI?

Both manage the same resources through the same api, and both are written in
Go. The difference is how you use them:

| [smarter-golang](https://github.com/smarter-sh/smarter-golang) (this module) | [smarter-cli](https://github.com/smarter-sh/smarter-cli) |
| ---------------------------------------------------------------------------- | -------------------------------------------------------- |
| A library that you **import** into your own software.                        | A program that you **install** on your computer.         |
| `client.Manifests.Apply(ctx, manifest)`                                      | `smarter apply -f guardrail.yaml`                        |
| Returns typed structs, and errors for `errors.Is` and `errors.As`.           | Prints JSON or YAML in a terminal window.                |
| For services, operators, tests and automation.                               | For people at a keyboard, and shell scripts.             |

The SDK reads the cli's configuration file exactly as the cli does, so if you
have run `smarter configure`, the SDK needs no other setup.

## At a Glance

- **Every resource kind**, through the same manifest api as the cli:
  `Apply`, `Validate`, `Get`, `Describe`, `Delete`, `Deploy`, `Undeploy` and
  `Logs`, for LLMClients, Plugins, Guardrails, MCPClients, Vectorstores,
  LLMHosts, Secrets, Custom Domains, Accounts, Users, and every other kind.
- **Prompts and conversations**: prompt an LLMClient, hold a multi-turn chat
  that keeps its session, and stream a prompt's progress (its LLM requests,
  tool calls and MCP calls) as it runs.
- **Resource actions**: evaluate a Guardrail, call an MCP server's tools, add
  documents to and search a vectorstore, plan and deploy an LLMHost, discover
  open-weight models, and call an LLM provider through a Proxy.
- **The platform's own models**: a Go struct of every
  [SAM manifest](https://docs.smarter.sh/en/latest/smarter-framework/smarter-manifests.html)
  kind (e.g. `sam.LLMClient`), generated from the platform's JSON Schemas, and
  the Smarter Chat api contract (e.g. `smarter.PromptChatAPIConfig`), kept in
  step with the platform by tests.
- **Production behavior**: `context.Context` on every call, retries with
  backoff, timeouts, typed errors, `log/slog` logging that never prints your
  api key, and one dependency besides the standard library.

## Quickstart

```console
go get github.com/smarter-sh/smarter-golang
```

Point the SDK at your platform with environment variables, or pass them as
options to `smarter.NewClient(...)`:

```console
export SMARTER_API_KEY="your 64-character Smarter api key"
export SMARTER_BASE_URL="https://platform.example.com"   # or http://localhost:9357
```

### Manifests

```go
manifest, err := smarter.ManifestFromFile("pii-redactor.yaml")
if err != nil {
	return err
}
if _, err := client.Manifests.Apply(ctx, manifest); err != nil { // create or update
	return err
}
described, err := client.Guardrails.Describe(ctx, "pii_redactor") // read it back
if err != nil {
	return err
}
guardrail, err := smarter.ToSAM[sam.Guardrail](described) // as a typed sam.Guardrail
if err != nil {
	return err
}
fmt.Println(guardrail.Spec.Config.Action)
```

### Chat

```go
chat, err := client.LLMClients.Chat(ctx, llmclient.Ref(), nil)
if err != nil {
	return err
}
reply, err := chat.Send(ctx, "What is 2 to the 10th power?")
if err != nil {
	return err
}
fmt.Println(reply.Text())

stream := chat.Stream(ctx, "Now divide it by 4.")
defer stream.Close()
for stream.Next() {
	fmt.Println("...", stream.Current().Message) // e.g. "Calling tool calculator"
}
response, err := stream.Response()
```

`client.LLMClients.Config` returns an LLMClient's chat configuration and a
session. It has what a chat needs; pass
`&smarter.PromptParams{Verbose: true}` for the complete configuration, with the
LLMClient's plugin and extension lists and the session's tool calls.

### Errors

```go
_, err := client.Manifests.Describe(ctx, "LLMClient", "no_such_client")
var status *smarter.APIStatusError
switch {
case errors.Is(err, smarter.ErrNotFound) && errors.As(err, &status):
	fmt.Println(status.StatusCode, status.Message, status.ErrorClass())
case errors.Is(err, smarter.ErrSmarter):
	fmt.Println("something else went wrong:", err)
}
```

## Documentation

The api reference is on [pkg.go.dev](https://pkg.go.dev/github.com/smarter-sh/smarter-golang),
and the guides on [docs.smarter.sh](https://docs.smarter.sh/en/latest/smarter-framework/smarter-golang.html).
The design decisions are recorded in [doc/adr/](./doc/adr/).

## Requirements

A supported Go release (1.26 or later), and a Smarter platform to connect to.
See the [Quick Start Guide](https://docs.smarter.sh/en/latest/smarter-platform/installation/quick-start.html)
to run one on your laptop.

## Development

```console
make init        # the release tooling, and the pre-commit hooks
make test        # the unit tests, which never reach a platform
make coverage    # the unit tests, with coverage
make vet         # go vet and golangci-lint
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=... make test-live
```

`make init` creates a Python virtual environment in `.venv/` with
[pre-commit](https://pre-commit.com/) (Python 3.13), and installs the
pre-commit and commit-msg hooks, which lint each commit and check its message.
`make python-init` does only that part. The Go hooks also need
[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) and
[golangci-lint](https://golangci-lint.run/) on your `PATH`.

The agent skills in [skills/](./skills/) describe how this repository is
developed, tested and documented, for people and for AI coding agents.

## Support

Please report bugs to the [GitHub Issues Page](https://github.com/smarter-sh/smarter-golang/issues)
for this project.

## Contributing

Please see [CONTRIBUTING](./.github/CONTRIBUTING.md).
