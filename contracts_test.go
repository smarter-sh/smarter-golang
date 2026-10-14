package smarter_test

// Test that the SDK's mirrors of the platform's api contracts are current. testdata/contracts/ holds
// the platform's published JSON Schemas and examples, and the api's OpenAPI document (see
// scripts/sync-contracts.sh). These are the Python SDK's tests/test_contracts.py.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

// mirrors maps each platform contract model to the Go type that mirrors it.
var mirrors = map[string]any{
	"PromptChatApiTextPart":          smarter.PromptChatAPIContentPart{},
	"PromptChatApiImagePart":         smarter.PromptChatAPIContentPart{},
	"PromptChatApiAudioPart":         smarter.PromptChatAPIContentPart{},
	"PromptChatApiFilePart":          smarter.PromptChatAPIContentPart{},
	"PromptChatApiRefusalPart":       smarter.PromptChatAPIContentPart{},
	"PromptChatApiImageUrl":          smarter.PromptChatAPIImageURL{},
	"PromptChatApiInputAudio":        smarter.PromptChatAPIInputAudio{},
	"PromptChatApiFileRef":           smarter.PromptChatAPIFileRef{},
	"PromptChatApiFunctionCall":      smarter.PromptChatAPIFunctionCall{},
	"PromptChatApiToolCall":          smarter.PromptChatAPIToolCall{},
	"PromptChatApiMessage":           smarter.PromptChatAPIMessage{},
	"PromptChatApiJournalMetadata":   smarter.PromptChatAPIJournalMetadata{},
	"PromptChatApiErrorBody":         smarter.PromptChatAPIErrorBody{},
	"PromptChatApiErrorResponse":     smarter.PromptChatAPIErrorResponse{},
	"PromptChatApiStyleSection":      smarter.PromptChatAPIStyleSection{},
	"PromptChatApiStyles":            smarter.PromptChatAPIStyles{},
	"PromptChatApiLLMClient":         smarter.PromptChatAPILLMClient{},
	"PromptChatApiPluginMeta":        smarter.PromptChatAPIPluginMeta{},
	"PromptChatApiLLMClientPlugin":   smarter.PromptChatAPILLMClientPlugin{},
	"PromptChatApiLLMClientFunction": smarter.PromptChatAPILLMClientFunction{},
	"PromptChatApiPromptRequest":     smarter.PromptChatAPIPromptRequest{},
	"PromptChatApiLLMClientRequest":  smarter.PromptChatAPILLMClientRequest{},
	"PromptChatApiPromptToolCall":    smarter.PromptChatAPIPromptToolCall{},
	"PromptChatApiPromptPluginUsage": smarter.PromptChatAPIPromptPluginUsage{},
	"PromptChatApiPluginSelection":   smarter.PromptChatAPIPluginSelection{},
	"PromptChatApiHistory":           smarter.PromptChatAPIHistory{},
	"PromptChatApiMetaData":          smarter.PromptChatAPIMetaData{},
	"PromptChatApiListMetaData":      smarter.PromptChatAPIListMetaData{},
	"PromptChatApiPluginList":        smarter.PromptChatAPIPluginList{},
	"PromptChatApiFunctionList":      smarter.PromptChatAPIFunctionList{},
	"PromptChatApiConfig":            smarter.PromptChatAPIConfig{},
	"PromptChatApiConfigRequest":     smarter.PromptChatAPIConfigRequest{},
	"PromptChatApiConfigResponse":    smarter.PromptChatAPIConfigResponse{},
	"PromptChatApiChoice":            smarter.PromptChatAPIChoice{},
	"PromptChatApiTokenDetails":      smarter.PromptChatAPITokenDetails{},
	"PromptChatApiUsage":             smarter.PromptChatAPIUsage{},
	"PromptChatApiCompletion":        smarter.PromptChatAPICompletion{},
	"PromptChatApiPromptResult":      smarter.PromptChatAPIPromptResult{},
	"PromptChatApiPromptResponse":    smarter.PromptChatAPIPromptResponse{},
	"PromptChatApiProgressEvent":     smarter.PromptChatAPIProgressEvent{},
	"PromptChatApiProgressResult":    smarter.PromptChatAPIProgressResult{},
	"SAMJournalApiMetadata":          smarter.SAMJournalAPIMetadata{},
	"SAMJournalApiErrorBody":         smarter.SAMJournalAPIErrorBody{},
	"SAMJournalApiEnvelope":          smarter.SAMJournalAPIEnvelope{},
	"SAMJournalApiErrorResponse":     smarter.SAMJournalAPIErrorResponse{},
}

// jsonFields returns the JSON names of a struct's fields, including its embedded structs'.
func jsonFields(t reflect.Type) map[string]bool {
	fields := map[string]bool{}
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Anonymous {
			for name := range jsonFields(field.Type) {
				fields[name] = true
			}
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			fields[name] = true
		}
	}
	return fields
}

func loadContract(t *testing.T, name string) map[string]any {
	t.Helper()
	return object(t, loadJSON(t, "contracts/"+name))
}

func declaresSchemaFields(t *testing.T, schema map[string]any, prefix string) int {
	t.Helper()
	checked := 0
	for name, definition := range object(t, schema["$defs"]) {
		if !strings.HasPrefix(name, prefix) || strings.HasSuffix(name, "V1") {
			continue
		}
		mirror, ok := mirrors[name]
		if !ok {
			continue
		}
		fields := jsonFields(reflect.TypeOf(mirror))
		properties, _ := object(t, definition)["properties"].(map[string]any)
		var missing []string
		for property := range properties {
			if !fields[property] {
				missing = append(missing, property)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s does not declare the platform's fields %v", name, missing)
		}
		checked++
	}
	return checked
}

func TestContractMirrorsDeclareEveryField(t *testing.T) {
	if checked := declaresSchemaFields(t, loadContract(t, "chat-contract.schema"), "PromptChatApi"); checked <= 30 {
		t.Errorf("only %d chat models were checked", checked)
	}
	if checked := declaresSchemaFields(t, loadContract(t, "journal-contract.schema"), "SAMJournalApi"); checked < 4 {
		t.Errorf("only %d journal models were checked", checked)
	}
}

func TestContractExamplesDecode(t *testing.T) {
	decoded := 0
	for name, example := range loadContract(t, "chat-contract.example") {
		if strings.HasSuffix(name, "V1") {
			continue
		}
		mirror, ok := mirrors[name]
		if !ok {
			t.Errorf("%s has no mirror", name)
			continue
		}
		data, _ := json.Marshal(example)
		target := reflect.New(reflect.TypeOf(mirror)).Interface()
		if err := json.Unmarshal(data, target); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		decoded++
	}
	if decoded < 8 {
		t.Errorf("only %d examples were decoded", decoded)
	}
}

// notCalled are the operations that the SDK doesn't call: the cli's prompt commands, which it
// replaces with the chat api, and the reads of one related row of an LLMClient, or of one model of a
// Provider, whose lists it reads instead. The same as the Python SDK's.
var notCalled = []string{
	"llmclient_apikey_retrieve",
	"llmclient_customdomain_retrieve",
	"llmclient_function_retrieve",
	"llmclient_plugin_retrieve",
	"prompt",
	"prompt_config",
	"provider_model",
}

func TestEveryRequestIsPublished(t *testing.T) {
	document := loadContract(t, "openapi")
	paths := object(t, document["paths"])
	// literal paths first: /llmhosts/report/ is also matched by /llmhosts/{id}/.
	templates := make([]string, 0, len(paths))
	for template := range paths {
		templates = append(templates, template)
	}
	sort.Slice(templates, func(i, j int) bool {
		pi, pj := strings.Count(templates[i], "{"), strings.Count(templates[j], "{")
		if pi != pj {
			return pi < pj
		}
		return templates[i] < templates[j]
	})
	operation := func(method, path string) string {
		for _, template := range templates {
			item := paths[template]
			pattern := "^/api/v1" + regexp.MustCompile(`\\\{[^/]+?\\\}`).ReplaceAllString(regexp.QuoteMeta(template), "[^/]+") + "$"
			entry, ok := object(t, item)[strings.ToLower(method)]
			if ok && regexp.MustCompile(pattern).MatchString(path) {
				return object(t, entry)["operationId"].(string)
			}
		}
		t.Errorf("%s %s is not in the platform's OpenAPI document", method, path)
		return ""
	}

	p := newMockPlatform(t)
	c := p.client
	ctx := context.Background()
	guardrail, err := smarter.ManifestFromObject(object(t, loadJSON(t, "sam/Guardrail.example")))
	check(t, err)
	upload := filepath.Join(t.TempDir(), "notes.txt")
	check(t, os.WriteFile(upload, []byte("Some notes."), 0o600))
	id, hashed := smarter.IntID(1), smarter.ID("rMTAwMDAwMQx")
	calls := []func() error{
		func() error { _, err := c.Manifests.Apply(ctx, guardrail); return err },
		func() error { _, err := c.Manifests.Validate(ctx, guardrail); return err },
		func() error { _, err := c.Manifests.Get(ctx, "Guardrail", ""); return err },
		func() error { _, err := c.Manifests.Describe(ctx, "Guardrail", "example"); return err },
		func() error { _, err := c.Manifests.Delete(ctx, "Guardrail", "example"); return err },
		func() error { _, err := c.Manifests.Deploy(ctx, "LLMClient", "example"); return err },
		func() error { _, err := c.Manifests.Undeploy(ctx, "LLMClient", "example"); return err },
		func() error { _, err := c.Manifests.Logs(ctx, "LLMClient", "example"); return err },
		func() error { _, err := c.Manifests.ExampleManifest(ctx, "Guardrail"); return err },
		func() error { _, err := c.Manifests.JSONSchema(ctx, "Guardrail"); return err },
		func() error { _, err := c.WhoAmI(ctx); return err },
		func() error { _, err := c.Version(ctx); return err },
		func() error { _, err := c.Status(ctx); return err },
		func() error { _, err := c.Kinds(ctx); return err },
		func() error { _, err := c.LLMClients.List(ctx); return err },
		func() error { _, err := c.LLMClients.Retrieve(ctx, hashed); return err },
		func() error { _, err := c.LLMClients.Plugins(ctx, smarter.IntID(12)); return err },
		func() error { _, err := c.LLMClients.Functions(ctx, smarter.IntID(12)); return err },
		func() error { _, err := c.LLMClients.APIKeys(ctx, smarter.IntID(12)); return err },
		func() error { _, err := c.LLMClients.CustomDomains(ctx, smarter.IntID(12)); return err },
		func() error { _, err := c.Guardrails.List(ctx); return err },
		func() error { _, err := c.Guardrails.Retrieve(ctx, hashed); return err },
		func() error { _, err := c.Guardrails.Evaluate(ctx, id, "hello", nil); return err },
		func() error {
			_, err := c.Guardrails.Events(ctx, id, &smarter.GuardrailEventsParams{Reviewed: smarter.Ptr(false)})
			return err
		},
		func() error { _, err := c.Secrets.List(ctx); return err },
		func() error { _, err := c.Secrets.Retrieve(ctx, id); return err },
		func() error { _, err := c.MCPClients.List(ctx); return err },
		func() error { _, err := c.MCPClients.Retrieve(ctx, hashed); return err },
		func() error {
			_, err := c.MCPClients.Tools(ctx, id, &smarter.MCPToolsParams{Refresh: true})
			return err
		},
		func() error { _, err := c.MCPClients.Refresh(ctx, id); return err },
		func() error {
			_, err := c.MCPClients.CallTool(ctx, id, "add_numbers", smarter.JSONObject{"a": 1, "b": 2})
			return err
		},
		func() error { _, err := c.Vectorstores.List(ctx); return err },
		func() error { _, err := c.Vectorstores.Retrieve(ctx, hashed); return err },
		func() error { _, err := c.Vectorstores.Status(ctx, id); return err },
		func() error { _, err := c.Vectorstores.Deploy(ctx, id); return err },
		func() error { _, err := c.Vectorstores.Undeploy(ctx, id); return err },
		func() error { _, err := c.Vectorstores.Documents(ctx, id); return err },
		func() error {
			_, err := c.Vectorstores.AddText(ctx, id, "Some text.", &smarter.AddTextParams{Name: "faq"})
			return err
		},
		func() error {
			_, err := c.Vectorstores.AddURL(ctx, id, "https://example.com/page.html", nil)
			return err
		},
		func() error { _, err := c.Vectorstores.AddFiles(ctx, id, []string{upload}); return err },
		func() error { _, err := c.Vectorstores.DeleteDocument(ctx, id, 2); return err },
		func() error { _, err := c.Vectorstores.Search(ctx, id, "query", nil); return err },
		func() error { _, err := c.Vectorstores.Snapshots(ctx, id); return err },
		func() error { _, err := c.Vectorstores.CreateSnapshot(ctx, id); return err },
		func() error { _, err := c.Vectorstores.Restore(ctx, id, 3); return err },
		func() error { _, err := c.LLMHosts.List(ctx); return err },
		func() error { _, err := c.LLMHosts.Retrieve(ctx, hashed); return err },
		func() error { _, err := c.LLMHosts.Status(ctx, id); return err },
		func() error { _, err := c.LLMHosts.Logs(ctx, id, &smarter.LogsParams{Tail: 20}); return err },
		func() error { _, err := c.LLMHosts.Plan(ctx, id); return err },
		func() error { _, err := c.LLMHosts.Deploy(ctx, id); return err },
		func() error { _, err := c.LLMHosts.Undeploy(ctx, id, &smarter.UndeployParams{Purge: true}); return err },
		func() error { _, err := c.LLMHosts.Report(ctx); return err },
		func() error { _, err := c.LLMHosts.Discover(ctx, &smarter.DiscoverParams{Query: "llama"}); return err },
		func() error { _, err := c.LLMHosts.DiscoverManifest(ctx, "BAAI/bge-m3", nil); return err },
		func() error { _, err := c.Plugins.List(ctx); return err },
		func() error { _, err := c.Plugins.RetrieveManifest(ctx, smarter.IntID(355)); return err },
		func() error { _, err := c.Providers.List(ctx); return err },
		func() error { _, err := c.Providers.Retrieve(ctx, "openai"); return err },
		func() error { _, err := c.Providers.Models(ctx, "openai"); return err },
		func() error { _, err := c.Proxies.List(ctx); return err },
		func() error { _, err := c.Orchestrators.List(ctx); return err },
		func() error { _, err := c.Orchestrators.Retrieve(ctx, hashed); return err },
		func() error { _, err := c.Vectorsearches.List(ctx); return err },
		func() error { _, err := c.Vectorsearches.Retrieve(ctx, id); return err },
		func() error { _, err := c.Accounts.List(ctx); return err },
		func() error { _, err := c.Accounts.Retrieve(ctx, id); return err },
		func() error { _, err := c.Accounts.Users.List(ctx); return err },
		func() error { _, err := c.Accounts.Users.Retrieve(ctx, id); return err },
		func() error { _, err := c.Accounts.UserProfiles.List(ctx); return err },
		func() error { _, err := c.Accounts.UserProfiles.Retrieve(ctx, id); return err },
		func() error { _, err := c.Accounts.Contacts.List(ctx); return err },
		func() error { _, err := c.Accounts.Contacts.Retrieve(ctx, id); return err },
		func() error { _, err := c.History.Sessions(ctx); return err },
		func() error { _, err := c.History.Session(ctx, id); return err },
		func() error { _, err := c.History.ToolCalls(ctx); return err },
		func() error { _, err := c.History.ToolCall(ctx, id); return err },
		func() error { _, err := c.History.PluginUsage(ctx); return err },
		func() error { _, err := c.History.PluginUse(ctx, id); return err },
		func() error {
			_, err := c.Journal.List(ctx, &smarter.JournalListParams{Page: 2, Search: "apply"})
			return err
		},
		func() error { _, err := c.Journal.Retrieve(ctx, strings.Repeat("0", 64)); return err },
		func() error { _, err := c.LLMClients.Config(ctx, smarter.IntID(12), nil); return err },
		func() error {
			_, err := c.LLMClients.Prompt(ctx, smarter.ID("rMTAwMDAxMgx"), smarter.Text("Hello"), nil)
			return err
		},
		func() error {
			_, err := c.LLMClients.Stream(ctx, smarter.IntID(12), smarter.Text("Hello"), nil).UntilDone()
			return err
		},
	}
	for _, call := range calls {
		// the requests are answered with 404s: only their paths matter.
		if err := call(); err != nil && !errors.Is(err, smarter.ErrSmarter) {
			t.Fatalf("not an SDK error: %v", err)
		}
	}
	equal(t, p.count(), len(calls))
	called := map[string]bool{}
	for _, request := range p.requests {
		called[operation(request.Method, request.URL.Path)] = true
	}
	var uncalled []string
	for _, item := range paths {
		for _, method := range []string{"get", "post", "delete"} {
			if entry, ok := object(t, item)[method]; ok {
				if id := object(t, entry)["operationId"].(string); !called[id] {
					uncalled = append(uncalled, id)
				}
			}
		}
	}
	sort.Strings(uncalled)
	jsonEqual(t, uncalled, notCalled)
}
