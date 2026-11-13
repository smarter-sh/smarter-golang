package smarter_test

// Test that the SDK's mirrors of the platform's api contracts are current. testdata/contracts/ holds
// the platform's published JSON Schemas and examples, and the api's OpenAPI document (see
// scripts/sync-contracts.sh). These are the Python SDK's tests/test_contracts.py.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

// mirrors maps each platform contract model to the Go type that mirrors it.
var mirrors = map[string]any{
	"PromptChatApiTextPart":                smarter.PromptChatAPIContentPart{},
	"PromptChatApiImagePart":               smarter.PromptChatAPIContentPart{},
	"PromptChatApiAudioPart":               smarter.PromptChatAPIContentPart{},
	"PromptChatApiFilePart":                smarter.PromptChatAPIContentPart{},
	"PromptChatApiRefusalPart":             smarter.PromptChatAPIContentPart{},
	"PromptChatApiImageUrl":                smarter.PromptChatAPIImageURL{},
	"PromptChatApiInputAudio":              smarter.PromptChatAPIInputAudio{},
	"PromptChatApiFileRef":                 smarter.PromptChatAPIFileRef{},
	"PromptChatApiFunctionCall":            smarter.PromptChatAPIFunctionCall{},
	"PromptChatApiToolCall":                smarter.PromptChatAPIToolCall{},
	"PromptChatApiMessage":                 smarter.PromptChatAPIMessage{},
	"PromptChatApiJournalMetadata":         smarter.PromptChatAPIJournalMetadata{},
	"PromptChatApiErrorBody":               smarter.PromptChatAPIErrorBody{},
	"PromptChatApiErrorResponse":           smarter.PromptChatAPIErrorResponse{},
	"PromptChatApiStyleSection":            smarter.PromptChatAPIStyleSection{},
	"PromptChatApiStyles":                  smarter.PromptChatAPIStyles{},
	"PromptChatApiLLMClient":               smarter.PromptChatAPILLMClient{},
	"PromptChatApiPluginMeta":              smarter.PromptChatAPIPluginMeta{},
	"PromptChatApiLLMClientPlugin":         smarter.PromptChatAPILLMClientPlugin{},
	"PromptChatApiLLMClientExtension":      smarter.PromptChatAPILLMClientExtension{},
	"PromptChatApiPromptRequest":           smarter.PromptChatAPIPromptRequest{},
	"PromptChatApiLLMClientRequest":        smarter.PromptChatAPILLMClientRequest{},
	"PromptChatApiPromptToolCall":          smarter.PromptChatAPIPromptToolCall{},
	"PromptChatApiPromptPluginUsage":       smarter.PromptChatAPIPromptPluginUsage{},
	"PromptChatApiPluginSelection":         smarter.PromptChatAPIPluginSelection{},
	"PromptChatApiHistory":                 smarter.PromptChatAPIHistory{},
	"PromptChatApiMetaData":                smarter.PromptChatAPIMetaData{},
	"PromptChatApiListMetaData":            smarter.PromptChatAPIListMetaData{},
	"PromptChatApiPluginList":              smarter.PromptChatAPIPluginList{},
	"PromptChatApiExtensionList":           smarter.PromptChatAPIExtensionList{},
	"PromptChatApiConfig":                  smarter.PromptChatAPIConfig{},
	"PromptChatApiConfigRequest":           smarter.PromptChatAPIConfigRequest{},
	"PromptChatApiConfigResponse":          smarter.PromptChatAPIConfigResponse{},
	"PromptChatApiProviderRequest":         smarter.PromptChatAPIProviderRequest{},
	"PromptChatApiProviderRequestResponse": smarter.PromptChatAPIProviderRequestResponse{},
	"PromptChatApiChoice":                  smarter.PromptChatAPIChoice{},
	"PromptChatApiTokenDetails":            smarter.PromptChatAPITokenDetails{},
	"PromptChatApiUsage":                   smarter.PromptChatAPIUsage{},
	"PromptChatApiCompletion":              smarter.PromptChatAPICompletion{},
	"PromptChatApiPromptResult":            smarter.PromptChatAPIPromptResult{},
	"PromptChatApiPromptResponse":          smarter.PromptChatAPIPromptResponse{},
	"PromptChatApiProgressEvent":           smarter.PromptChatAPIProgressEvent{},
	"PromptChatApiProgressResult":          smarter.PromptChatAPIProgressResult{},
	"SAMJournalApiMetadata":                smarter.SAMJournalAPIMetadata{},
	"SAMJournalApiErrorBody":               smarter.SAMJournalAPIErrorBody{},
	"SAMJournalApiEnvelope":                smarter.SAMJournalAPIEnvelope{},
	"SAMJournalApiErrorResponse":           smarter.SAMJournalAPIErrorResponse{},
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
	"llmclient_extension_retrieve",
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
		func() error { _, err := c.Manifests.Get(ctx, "Guardrail", "", nil); return err },
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
		func() error { _, err := c.LLMClients.Extensions(ctx, smarter.IntID(12)); return err },
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
		func() error { _, err := c.Accounts.Usage(ctx, smarter.UsagePeriodicity7Days); return err },
		func() error { _, err := c.Accounts.DataExports.Overview(ctx); return err },
		func() error { _, err := c.Accounts.DataExports.Start(ctx); return err },
		func() error { _, err := c.Accounts.DataExports.Retrieve(ctx, 7); return err },
		func() error { _, err := c.Vectorsearches.Run(ctx, hashed, "refunds", nil); return err },
		func() error { _, err := c.Extensions.List(ctx); return err },
		func() error { _, err := c.Extensions.Registry(ctx, true); return err },
		func() error { _, err := c.Extensions.Retrieve(ctx, "weather"); return err },
		func() error { _, err := c.Extensions.CheckUpdates(ctx); return err },
		func() error { _, err := c.Extensions.Install(ctx, "weather", ""); return err },
		func() error { _, err := c.Extensions.Upload(ctx, strings.NewReader("PK"), "weather.zip"); return err },
		func() error { _, err := c.Extensions.Enable(ctx, "weather"); return err },
		func() error { _, err := c.Extensions.Disable(ctx, "weather"); return err },
		func() error { _, err := c.Extensions.Update(ctx, "weather"); return err },
		func() error { _, err := c.Extensions.Rollback(ctx, "weather", 3); return err },
		func() error { _, err := c.Extensions.Configure(ctx, "weather", nil); return err },
		func() error { _, err := c.Extensions.SetAvailability(ctx, "weather", true, nil); return err },
		func() error { _, err := c.Extensions.Uninstall(ctx, "weather"); return err },
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
		func() error { _, err := c.LLMClients.ProviderRequest(ctx, smarter.IntID(12), nil); return err },
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

// recordedResponse is the operation of the platform's OpenAPI document whose response a fixture
// records: its path and method, and the response's status, or "" for its 200 (or 202) response.
type recordedResponse struct {
	path, method, status string
}

// recordedResponses maps each fixture of testdata/ to the operation and status of the response that it
// records. A fixture that drifted from what
// the platform publishes would let the tests pass against a response that the platform never sends.
// The Python SDK's TestRecordedResponses.FIXTURES, with the extensions, usage, data exports and
// Vectorsearch run, which it does not cover yet, and the documented refusals (402 and 429).
var recordedResponses = map[string]recordedResponse{
	"llmclients":           {"/llm-clients/", "get", ""},
	"llmclient":            {"/llm-clients/{id}/", "get", ""},
	"llmclient_plugins":    {"/llm-clients/{id}/plugins/", "get", ""},
	"llmclient_extensions": {"/llm-clients/{id}/extensions", "get", ""},
	"config":               {"/llm-clients/{id}/config/", "post", ""},
	"provider_request":     {"/llm-clients/{id}/provider-request/", "post", ""},
	"prompt":               {"/llm-clients/{id}/prompt/", "post", ""},
	"guardrails":           {"/guardrails/", "get", ""},
	"guardrail_eval":       {"/guardrails/{id}/evaluate/", "post", ""},
	"guardrail_events":     {"/guardrails/{id}/events/", "get", ""},
	"secrets":              {"/secrets/", "get", ""},
	"mcpclients":           {"/mcpclients/", "get", ""},
	"mcp_tools":            {"/mcpclients/{id}/tools/", "get", ""},
	"vectorstores":         {"/vectorstores/", "get", ""},
	"llmhosts":             {"/llmhosts/", "get", ""},
	"llmhost_status":       {"/llmhosts/{id}/status/", "get", ""},
	"llmhost_plan":         {"/llmhosts/{id}/plan/", "get", ""},
	"llmhost_report":       {"/llmhosts/report/", "get", ""},
	"llmhost_discover":     {"/llmhosts/discover/", "get", ""},
	"plugins":              {"/plugins/", "get", ""},
	"proxy":                {"/proxy/", "get", ""},
	"accounts":             {"/accounts/", "get", ""},
	"users":                {"/accounts/users/", "get", ""},
	"contacts":             {"/accounts/contacts/", "get", ""},
	"history":              {"/prompts/history/prompt/", "get", ""},
	"toolcalls":            {"/prompts/history/tool-calls/", "get", ""},
	"journal":              {"/journal/", "get", ""},
	"plugin_usage":         {"/prompts/history/plugin-usage/", "get", ""},
	"user_profiles":        {"/accounts/user-profiles/", "get", ""},
	"extensions":           {"/extensions/", "get", ""},
	"extension_registry":   {"/extensions/registry/", "get", ""},
	"extension_detail":     {"/extensions/{slug}/", "get", ""},
	"extension_action":     {"/extensions/{slug}/enable/", "post", ""},
	"token_usage":          {"/accounts/usage/", "get", ""},
	"data_export_overview": {"/accounts/data-exports/", "get", ""},
	"data_export_status":   {"/accounts/data-exports/{export_id}/", "get", ""},
	"vectorsearch_run":     {"/vectorsearches/{id}/vectorsearch/", "post", ""},
	"vectorsearch_budget":  {"/vectorsearches/{id}/vectorsearch/", "post", "402"},
	"data_export_refused":  {"/accounts/data-exports/", "post", "429"},
}

// schemaErrors returns where a JSON value breaks a JSON Schema of the OpenAPI document. It checks the
// keywords that the platform's document uses for its responses' shapes: $ref, type, properties,
// required, additionalProperties, items, anyOf, oneOf, allOf, enum and const.
func schemaErrors(document, schema map[string]any, value any, at string) []string {
	if ref, ok := schema["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, "#/components/schemas/")
		target, _ := document["components"].(map[string]any)["schemas"].(map[string]any)[name].(map[string]any)
		if target == nil {
			return []string{at + ": unknown " + ref}
		}
		return schemaErrors(document, target, value, at)
	}
	var problems []string
	for _, keyword := range []string{"anyOf", "oneOf"} {
		if options, ok := schema[keyword].([]any); ok {
			matched := false
			for _, option := range options {
				if len(schemaErrors(document, option.(map[string]any), value, at)) == 0 {
					matched = true
					break
				}
			}
			if !matched {
				problems = append(problems, at+": matches no option of "+keyword)
			}
		}
	}
	if all, ok := schema["allOf"].([]any); ok {
		for _, option := range all {
			problems = append(problems, schemaErrors(document, option.(map[string]any), value, at)...)
		}
	}
	if enum, ok := schema["enum"].([]any); ok && !slices.ContainsFunc(enum, func(v any) bool { return reflect.DeepEqual(v, value) }) {
		problems = append(problems, fmt.Sprintf("%s: %v is not one of %v", at, value, enum))
	}
	if constant, ok := schema["const"]; ok && !reflect.DeepEqual(constant, value) {
		problems = append(problems, fmt.Sprintf("%s: %v is not %v", at, value, constant))
	}
	if declared, ok := schema["type"]; ok {
		types, ok := declared.([]any)
		if !ok {
			types = []any{declared}
		}
		if !slices.ContainsFunc(types, func(t any) bool { return jsonTypeIs(value, t.(string)) }) {
			return append(problems, fmt.Sprintf("%s: %v is not of type %v", at, value, declared))
		}
	}
	switch v := value.(type) {
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		for _, name := range schemaRequired(schema) {
			if _, ok := v[name]; !ok {
				problems = append(problems, at+": "+name+" is required")
			}
		}
		for name, item := range v {
			if property, ok := properties[name].(map[string]any); ok {
				problems = append(problems, schemaErrors(document, property, item, at+"."+name)...)
			} else if extra, ok := schema["additionalProperties"].(map[string]any); ok {
				problems = append(problems, schemaErrors(document, extra, item, at+"."+name)...)
			} else if schema["additionalProperties"] == false {
				problems = append(problems, at+": "+name+" is not a property")
			}
		}
	case []any:
		if items, ok := schema["items"].(map[string]any); ok {
			for i, item := range v {
				problems = append(problems, schemaErrors(document, items, item, fmt.Sprintf("%s[%d]", at, i))...)
			}
		}
	}
	return problems
}

func schemaRequired(schema map[string]any) []string {
	var names []string
	required, _ := schema["required"].([]any)
	for _, name := range required {
		names = append(names, name.(string))
	}
	return names
}

// jsonTypeIs reports whether a decoded JSON value is of a JSON Schema type.
func jsonTypeIs(value any, kind string) bool {
	switch v := value.(type) {
	case nil:
		return kind == "null"
	case bool:
		return kind == "boolean"
	case string:
		return kind == "string"
	case float64:
		return kind == "number" || (kind == "integer" && v == math.Trunc(v))
	case []any:
		return kind == "array"
	case map[string]any:
		return kind == "object"
	}
	return false
}

func TestRecordedResponsesAreValid(t *testing.T) {
	document := loadContract(t, "openapi")
	paths := object(t, document["paths"])
	for name, operation := range recordedResponses {
		t.Run(name, func(t *testing.T) {
			responses := object(t, object(t, object(t, paths[operation.path])[operation.method])["responses"])
			status := operation.status
			if status == "" {
				status = "200"
				if _, ok := responses[status]; !ok {
					status = "202"
				}
			}
			response, ok := responses[status]
			if !ok {
				t.Fatalf("%s %s documents no %s response", operation.method, operation.path, status)
			}
			schema := object(t, object(t, object(t, object(t, response)["content"])["application/json"])["schema"])
			for _, problem := range schemaErrors(document, schema, loadJSON(t, name), "$") {
				t.Error(problem)
			}
		})
	}
}

func TestSchemaErrors(t *testing.T) {
	document := map[string]any{"components": map[string]any{"schemas": map[string]any{
		"Point": map[string]any{
			"type": "object", "required": []any{"x"}, "additionalProperties": false,
			"properties": map[string]any{
				"x":    map[string]any{"type": "integer"},
				"tags": map[string]any{"type": "array", "items": map[string]any{"enum": []any{"a", "b"}}},
				"kind": map[string]any{"const": "point"},
				"note": map[string]any{"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "null"}}},
				"meta": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "boolean"}},
				"both": map[string]any{"allOf": []any{map[string]any{"type": []any{"number", "string"}}}},
			},
		},
	}}}
	point := map[string]any{"$ref": "#/components/schemas/Point"}
	valid := map[string]any{"x": 1.0, "tags": []any{"a"}, "kind": "point", "note": nil, "meta": map[string]any{"ok": true}, "both": 1.5}
	if problems := schemaErrors(document, point, valid, "$"); len(problems) != 0 {
		t.Errorf("got %v", problems)
	}
	invalid := map[string]any{"tags": []any{"c"}, "kind": "line", "note": 1.0, "meta": map[string]any{"ok": "yes"}, "both": false, "extra": true}
	equal(t, len(schemaErrors(document, point, invalid, "$")), 7)
	equal(t, len(schemaErrors(document, point, map[string]any{"x": 1.5}, "$")), 1)
	equal(t, len(schemaErrors(document, map[string]any{"$ref": "#/components/schemas/Missing"}, nil, "$")), 1)
}
