// Package smarter is the official Go SDK of the Smarter platform's api: the library that Go
// programs import to build software on their own Smarter installation, running locally or on
// Kubernetes, without customizing or forking the platform. It has the features of the Python SDK,
// smarter-python, which is the reference implementation of every Smarter SDK.
//
// # Getting started
//
//	client, err := smarter.NewClient() // SMARTER_API_KEY, or the smarter cli's ~/.smarter/config.yaml
//	if err != nil {
//		log.Fatal(err)
//	}
//	ctx := context.Background()
//	llmclient, err := client.LLMClients.Find(ctx, "customer_support")
//	if err != nil {
//		log.Fatal(err)
//	}
//	response, err := client.LLMClients.Prompt(ctx, llmclient.Ref(), smarter.Text("Where is my order #1234?"), nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(response.Text())
//
// # Configuration
//
// [NewClient] takes each value from the first of: its options ([WithAPIKey], [WithBaseURL],
// [WithEnvironment], [WithTimeout], [WithMaxRetries], ...), the SMARTER_* environment variables, the
// smarter cli's configuration file, ~/.smarter/config.yaml, and the defaults, so a developer who has
// run smarter configure needs no other setup.
//
// # Resources
//
// The client's fields group the api's operations. [Client.Manifests] applies, validates, gets,
// describes, deletes, deploys and undeploys manifests of any kind, exactly as the cli does: it is
// the only way that the SDK writes resources. Each resource family, e.g. [Client.Guardrails], reads
// its resources from the REST api (List, Retrieve, Find), applies, describes and deletes them through
// the manifest api, and has its own actions, e.g. [GuardrailsService.Evaluate].
//
// A [Manifest] is a resource's YAML document, whose Spec is a JSON object. [ToSAM] decodes one into
// its kind's generated struct, in the [github.com/smarter-sh/smarter-golang/sam] package.
//
// # Prompts, chats and streams
//
// [LLMClientsService.Prompt] prompts an LLMClient, [LLMClientsService.Chat] holds a multi-turn
// conversation that keeps its session, and [LLMClientsService.Stream] streams a prompt's progress
// (its LLM requests, tool calls and MCP calls) while it runs: see [PromptStream]. Prompts make
// billable requests to the LLMClient's provider.
//
// # Errors
//
// Every error that the SDK returns matches [ErrSmarter] with errors.Is. An api error is an
// [*APIStatusError], which matches the sentinel of its status, e.g. [ErrNotFound]:
//
//	_, err := client.Manifests.Describe(ctx, "LLMClient", "no_such_client")
//	var status *smarter.APIStatusError
//	if errors.As(err, &status) && errors.Is(err, smarter.ErrNotFound) {
//		fmt.Println(status.StatusCode, status.Message, status.ErrorClass())
//	}
//
// # Infrastructure
//
// Deploy methods create real, possibly billable infrastructure on the platform's cloud: DNS
// records, TLS certificates, Kubernetes ingresses and GPU nodes.
//
// The guides are at https://docs.smarter.sh/en/latest/smarter-framework/smarter-golang.html, and the
// design decisions in the repository's doc/adr/.
package smarter
