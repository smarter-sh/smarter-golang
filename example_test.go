package smarter_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	smarter "github.com/smarter-sh/smarter-golang"
	"github.com/smarter-sh/smarter-golang/sam"
)

// examplePlatform is a stand-in for a Smarter platform, for the examples: it answers with the
// recorded responses of testdata/.
func examplePlatform() *httptest.Server {
	fixtures := map[string]string{
		"POST /api/v1/cli/whoami/":                  "whoami.json",
		"GET /api/v1/llm-clients/":                  "llmclients.json",
		"POST /api/v1/llm-clients/5/prompt/":        "prompt.json",
		"POST /api/v1/llm-clients/6/prompt/":        "prompt_sse.txt",
		"POST /api/v1/cli/apply/":                   "delete.json",
		"POST /api/v1/cli/describe/Guardrail/":      "example_guardrail.json",
		"POST /api/v1/cli/describe/LLMClient/":      "describe_missing.json",
		"POST /api/v1/guardrails/17/evaluate/":      "guardrail_eval.json",
		"POST /api/v1/cli/example-manifest/Secret/": "example_guardrail.json",
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := fixtures[r.Method+" "+r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.Contains(name, "missing") {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_, _ = w.Write(data)
	}))
}

func exampleClient(server *httptest.Server) *smarter.Client {
	client, err := smarter.NewClient(
		smarter.WithAPIKey("an example api key, 64 characters long, as the platform issues"),
		smarter.WithBaseURL(server.URL),
		smarter.WithMaxRetries(0),
		smarter.WithLookupEnv(testEnv(nil)),
	)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func ExampleNewClient() {
	server := examplePlatform()
	defer server.Close()

	// smarter.NewClient() alone reads SMARTER_API_KEY, or the smarter cli's ~/.smarter/config.yaml.
	client, err := smarter.NewClient(
		smarter.WithAPIKey("an example api key, 64 characters long, as the platform issues"),
		smarter.WithBaseURL(server.URL),
		smarter.WithLookupEnv(testEnv(nil)),
	)
	if err != nil {
		log.Fatal(err)
	}
	me, err := client.WhoAmI(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(me.User.Username, me.Account.AccountNumber)
	// Output: admin 1234-5678-9012
}

func ExampleLLMClientsService_Prompt() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	response, err := client.LLMClients.Prompt(context.Background(), smarter.IntID(5), smarter.Text("What is 17 times 23?"), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(strings.Contains(response.Text(), "391"))
	// Output: true
}

func ExamplePromptStream() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	stream := client.LLMClients.Stream(context.Background(), smarter.IntID(6), smarter.Text("What is 17 times 23?"), nil)
	defer func() { _ = stream.Close() }()
	for stream.Next() {
		if stream.Current().Type == "llm_request" {
			fmt.Println("an LLM request")
			break
		}
	}
	// Output: an LLM request
}

func ExampleManifestsService_Apply() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	manifest, err := smarter.ManifestFromYAML(`
apiVersion: smarter.sh/v1
kind: Secret
metadata:
  name: openai_api_key
  description: The OpenAI api key.
spec:
  config:
    value: sk-...
`)
	if err != nil {
		log.Fatal(err)
	}
	response, err := client.Manifests.Apply(context.Background(), manifest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(response.Message != "")
	// Output: true
}

func ExampleToSAM() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	manifest, err := client.Guardrails.Describe(context.Background(), "pii_redactor")
	if err != nil {
		log.Fatal(err)
	}
	guardrail, err := smarter.ToSAM[sam.Guardrail](manifest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(guardrail.Kind, guardrail.Spec.Config.Strategy)
	// Output: Guardrail detector
}

func ExampleGuardrailsService_Evaluate() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	result, err := client.Guardrails.Evaluate(context.Background(), smarter.IntID(17), "my email is jane@example.com",
		&smarter.GuardrailEvaluateParams{Stage: smarter.StageOutput})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Triggered, result.Findings[0].Matches[0].Label)
	// Output: true email
}

func ExampleAPIStatusError() {
	server := examplePlatform()
	defer server.Close()
	client := exampleClient(server)

	_, err := client.Manifests.Describe(context.Background(), "LLMClient", "no_such_client")
	var status *smarter.APIStatusError
	if errors.As(err, &status) {
		fmt.Println(status.StatusCode, status.ErrorClass())
	}
	fmt.Println(errors.Is(err, smarter.ErrInternalServer), errors.Is(err, smarter.ErrSmarter))
	// Output:
	// 503 SAMBrokerErrorNotReady
	// true true
}
