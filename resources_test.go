package smarter_test

import (
	"context"
	"errors"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

func first(t *testing.T, name string) any {
	t.Helper()
	return loadJSON(t, name).([]any)[0]
}

func TestAccounts(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "accounts/", reply{fixture: "accounts"})
	p.route("GET", "accounts/1/", reply{json: first(t, "accounts")})
	p.route("GET", "accounts/users/", reply{fixture: "users"})
	p.route("GET", "accounts/users/1/", reply{json: first(t, "users")})
	p.route("GET", "accounts/contacts/", reply{fixture: "contacts"})
	p.route("GET", "accounts/contacts/1/", reply{json: first(t, "contacts")})
	p.route("GET", "accounts/user-profiles/", reply{fixture: "user_profiles"})
	p.route("GET", "accounts/user-profiles/1/", reply{json: first(t, "user_profiles")})
	accounts, err := p.client.Accounts.List(ctx)
	check(t, err)
	equal(t, accounts[0].AccountNumber, "1234-5678-9012")
	account, err := p.client.Accounts.Retrieve(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, account.CompanyName, "Example Company")
	users, err := p.client.Accounts.Users.List(ctx)
	check(t, err)
	equal(t, smarter.Deref(users[0].IsSuperuser), true)
	user, err := p.client.Accounts.Users.Retrieve(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, user.Username, "admin")
	contacts, err := p.client.Accounts.Contacts.List(ctx)
	check(t, err)
	equal(t, smarter.Deref(contacts[0].IsPrimary), true)
	contact, err := p.client.Accounts.Contacts.Retrieve(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, contact.Account.AccountNumber, "1234-5678-9012")
	profiles, err := p.client.Accounts.UserProfiles.List(ctx)
	check(t, err)
	equal(t, profiles[0].User.Username, "admin")
	profile, err := p.client.Accounts.UserProfiles.Retrieve(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, profile.Account.AccountNumber, "1234-5678-9012")
	equal(t, smarter.Deref(profile.IsTest), false)
	if _, err := p.client.Accounts.Users.Retrieve(ctx, smarter.ID("a/b")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "accounts/users/", reply{status: 403, json: map[string]any{}})
	if _, err := p.client.Accounts.Users.List(ctx); !errors.Is(err, smarter.ErrPermissionDenied) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.Accounts.Users.Retrieve(ctx, smarter.IntID(2)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestCatalog(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "secrets/", reply{fixture: "secrets"})
	p.route("GET", "secrets/1/", reply{json: first(t, "secrets")})
	secrets, err := p.client.Secrets.List(ctx)
	check(t, err)
	equal(t, secrets[0].Name, "anthropic_api_key")
	secret, err := p.client.Secrets.Retrieve(ctx, smarter.IntID(1))
	check(t, err)
	if secret.LastAccessed == "" {
		t.Error("no lastAccessed")
	}
	p.route("GET", "orchestrators/", reply{json: []any{}})
	p.route("GET", "vectorsearches/", reply{json: []any{map[string]any{"id": 1, "name": "v"}}})
	orchestrators, err := p.client.Orchestrators.List(ctx)
	check(t, err)
	equal(t, len(orchestrators), 0)
	vectorsearches, err := p.client.Vectorsearches.List(ctx)
	check(t, err)
	equal(t, vectorsearches[0].Name, "v")
}

func TestPlugins(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	plugins := loadJSON(t, "plugins").([]any)
	plugin := object(t, plugins[0])
	plugin["kind"] = "SqlPlugin"
	name := plugin["name"].(string)
	id := int(plugin["id"].(float64))
	p.route("GET", "plugins/", reply{json: plugins})
	p.route("GET", "plugins/"+smarter.IntID(id).String()+"/", reply{json: map[string]any{"apiVersion": "smarter.sh/v1", "kind": "SqlPlugin", "metadata": map[string]any{"name": name}, "spec": map[string]any{}}})
	p.route("POST", "cli/describe/SqlPlugin/", reply{json: map[string]any{"data": map[string]any{"kind": "SqlPlugin", "metadata": map[string]any{"name": name}}}})
	p.route("POST", "cli/delete/SqlPlugin/", reply{json: map[string]any{"message": "deleted"}})
	kind, err := p.client.Plugins.KindOf(ctx, name)
	check(t, err)
	equal(t, kind, "SqlPlugin")
	described, err := p.client.Plugins.Describe(ctx, name)
	check(t, err)
	equal(t, described.Kind, "SqlPlugin")
	deleted, err := p.client.Plugins.Delete(ctx, name)
	check(t, err)
	equal(t, deleted.Message, "deleted")
	manifest, err := p.client.Plugins.RetrieveManifest(ctx, smarter.IntID(id))
	check(t, err)
	equal(t, manifest.Metadata.Name, name)
	retrieved, err := p.client.Plugins.Retrieve(ctx, smarter.IntID(id))
	check(t, err)
	equal(t, retrieved.Name, name)
	if _, err := p.client.Plugins.Retrieve(ctx, smarter.IntID(-1)); !errors.Is(err, smarter.ErrResourceNotFound) {
		t.Errorf("got %v", err)
	}
	for _, call := range []func() error{
		func() error { _, err := p.client.Plugins.Describe(ctx, "nope"); return err },
		func() error { _, err := p.client.Plugins.Delete(ctx, "nope"); return err },
		func() error { _, err := p.client.Plugins.KindOf(ctx, "nope"); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrResourceNotFound) {
			t.Errorf("got %v", err)
		}
	}
	if _, err := p.client.Plugins.RetrieveManifest(ctx, smarter.ID("")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	plugin["kind"] = ""
	p.route("GET", "plugins/", reply{json: plugins})
	kind, err = p.client.Plugins.KindOf(ctx, name)
	check(t, err)
	equal(t, kind, "Plugin")
	p.route("GET", "plugins/", reply{status: 500, json: map[string]any{}})
	if _, err := p.client.Plugins.Retrieve(ctx, smarter.IntID(id)); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}

	p.route("POST", "cli/apply/", reply{json: map[string]any{"message": "applied"}})
	apiPlugin, err := smarter.ManifestFromObject(smarter.JSONObject{"kind": "ApiPlugin", "metadata": map[string]any{"name": "p"}, "spec": map[string]any{}})
	check(t, err)
	applied, err := p.client.Plugins.Apply(ctx, apiPlugin)
	check(t, err)
	equal(t, applied.Message, "applied")
	apiPlugin.Kind = "Secret"
	var value *smarter.ValueError
	if _, err := p.client.Plugins.Apply(ctx, apiPlugin); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
}

func TestGuardrails(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "guardrails/", reply{fixture: "guardrails"})
	guardrails, err := p.client.Guardrails.List(ctx)
	check(t, err)
	equal(t, guardrails[0].Stage, "output")
	jsonEqual(t, guardrails[0].Config, map[string]any{"detectors": []string{"credit_card", "us_ssn", "email", "phone_number", "iban"}})

	p.route("POST", "guardrails/17/evaluate/", reply{fixture: "guardrail_eval"})
	result, err := p.client.Guardrails.Evaluate(ctx, smarter.IntID(17), "my email is jane@example.com", &smarter.GuardrailEvaluateParams{Stage: smarter.StageOutput})
	check(t, err)
	equal(t, result.Triggered, true)
	equal(t, result.Findings[0].Matches[0].Label, "email")
	jsonEqual(t, p.lastJSON(), map[string]any{"stage": "output", "text": "my email is jane@example.com"})
	_, err = p.client.Guardrails.Evaluate(ctx, smarter.IntID(17), "hi", nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"stage": "input", "text": "hi"})

	p.route("POST", "guardrails/18/evaluate/", reply{status: 400, json: map[string]any{"error": "text is required."}})
	_, err = p.client.Guardrails.Evaluate(ctx, smarter.IntID(18), "", nil)
	var statusError *smarter.APIStatusError
	if !errors.As(err, &statusError) || statusError.Message != "text is required." {
		t.Errorf("got %v", err)
	}

	p.route("GET", "guardrails/17/events/", reply{fixture: "guardrail_events"})
	events, err := p.client.Guardrails.Events(ctx, smarter.IntID(17), &smarter.GuardrailEventsParams{Reviewed: smarter.Ptr(false), Disposition: "blocked"})
	check(t, err)
	equal(t, events.Guardrail, "pii_leak_output")
	equal(t, p.last().URL.RawQuery, "disposition=blocked&reviewed=false")
	_, err = p.client.Guardrails.Events(ctx, smarter.IntID(17), nil)
	check(t, err)
	equal(t, p.last().URL.RawQuery, "")
	for _, call := range []func() error{
		func() error { _, err := p.client.Guardrails.Evaluate(ctx, smarter.ID(""), "x", nil); return err },
		func() error { _, err := p.client.Guardrails.Events(ctx, smarter.ID(""), nil); return err },
		func() error { _, err := p.client.Guardrails.Events(ctx, smarter.IntID(19), nil); return err },
		func() error { _, err := p.client.Guardrails.Evaluate(ctx, smarter.IntID(19), "x", nil); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("got %v", err)
		}
	}

	example := object(t, object(t, loadJSON(t, "example_guardrail"))["data"])
	p.route("POST", "cli/describe/Guardrail/", reply{json: map[string]any{"data": example}})
	p.route("POST", "cli/delete/Guardrail/", reply{json: map[string]any{"message": "deleted"}})
	p.route("POST", "cli/example-manifest/Guardrail/", reply{fixture: "example_guardrail"})
	p.route("POST", "cli/json-schema/Guardrail/", reply{fixture: "schema_secret"})
	p.route("POST", "cli/apply/", reply{json: map[string]any{"message": "applied"}})
	described, err := p.client.Guardrails.Describe(ctx, "x")
	check(t, err)
	equal(t, described.Kind, "Guardrail")
	deleted, err := p.client.Guardrails.Delete(ctx, "x")
	check(t, err)
	equal(t, deleted.Message, "deleted")
	manifest, err := p.client.Guardrails.ExampleManifest(ctx)
	check(t, err)
	applied, err := p.client.Guardrails.Apply(ctx, manifest)
	check(t, err)
	equal(t, applied.Message, "applied")
	schema, err := p.client.Guardrails.JSONSchema(ctx)
	check(t, err)
	if len(schema) == 0 {
		t.Error("no schema")
	}
}

func TestHistory(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "prompts/history/prompt/", reply{fixture: "history"})
	p.route("GET", "prompts/history/prompt/43/", reply{json: first(t, "history")})
	p.route("GET", "prompts/history/tool-calls/", reply{fixture: "toolcalls"})
	p.route("GET", "prompts/history/tool-calls/3/", reply{json: first(t, "toolcalls")})
	p.route("GET", "prompts/history/plugin-usage/", reply{fixture: "plugin_usage"})
	p.route("GET", "prompts/history/plugin-usage/4/", reply{json: first(t, "plugin_usage")})
	sessions, err := p.client.History.Sessions(ctx)
	check(t, err)
	equal(t, len(sessions[0].SessionKey), 64)
	session, err := p.client.History.Session(ctx, smarter.IntID(43))
	check(t, err)
	equal(t, smarter.Deref(session.LLMClient), 5)
	toolCalls, err := p.client.History.ToolCalls(ctx)
	check(t, err)
	equal(t, toolCalls[0].FunctionName, "calculator")
	equal(t, smarter.Deref(toolCalls[0].Prompt.Session.LLMClient), 5)
	toolCall, err := p.client.History.ToolCall(ctx, smarter.IntID(3))
	check(t, err)
	equal(t, toolCall.ID, 3)
	usage, err := p.client.History.PluginUsage(ctx)
	check(t, err)
	equal(t, smarter.Deref(usage[0].Prompt.Session.LLMClient), 5)
	equal(t, usage[0].Plugin.Kind, "WebsearchPlugin")
	use, err := p.client.History.PluginUse(ctx, smarter.IntID(4))
	check(t, err)
	equal(t, use.InputText, "What is 17 * 23?")
	if _, err := p.client.History.Session(ctx, smarter.ID("")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.History.Session(ctx, smarter.IntID(1)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "prompts/history/prompt/", reply{status: 500, json: map[string]any{}})
	if _, err := p.client.History.Sessions(ctx); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}
	// a session reference that is only an id, as the platform sends some rows.
	p.route("GET", "prompts/history/tool-calls/", reply{json: []any{map[string]any{"id": 1, "prompt": 43}}})
	toolCalls, err = p.client.History.ToolCalls(ctx)
	check(t, err)
	equal(t, toolCalls[0].Prompt.ID, 43)
	jsonEqual(t, toolCalls[0].Prompt, 43)
	jsonEqual(t, smarter.SessionRef{Session: &smarter.PromptSession{ID: 7}}, map[string]any{"id": 7})
}

func TestJournal(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "journal/", reply{fixture: "journal"})
	page, err := p.client.Journal.List(ctx, &smarter.JournalListParams{Page: 2, PageSize: 10, Search: "apply", Ordering: "-createdAt"})
	check(t, err)
	equal(t, page.Pagination.PageSize, 25)
	query, _ := url.ParseQuery(p.last().URL.RawQuery)
	jsonEqual(t, query, url.Values{"page": {"2"}, "page_size": {"10"}, "search": {"apply"}, "ordering": {"-createdAt"}})
	_, err = p.client.Journal.List(ctx, nil)
	check(t, err)
	equal(t, p.last().URL.RawQuery, "page=1")
	key := strings.Repeat("a", 64)
	p.route("GET", "journal/"+key+"/", reply{json: map[string]any{"key": key, "thing": "Guardrail", "command": "apply", "statusCode": 200, "user": "admin", "request": map[string]any{}}})
	entry, err := p.client.Journal.Retrieve(ctx, key)
	check(t, err)
	equal(t, entry.StatusCode, 200)
	if _, err := p.client.Journal.Retrieve(ctx, ""); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.Journal.Retrieve(ctx, "b"); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "journal/", reply{status: 403, json: map[string]any{}})
	if _, err := p.client.Journal.List(ctx, nil); !errors.Is(err, smarter.ErrPermissionDenied) {
		t.Errorf("got %v", err)
	}
}

func TestLLMHosts(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "llmhosts/", reply{fixture: "llmhosts"})
	p.route("GET", "llmhosts/1/status/", reply{fixture: "llmhost_status"})
	p.route("GET", "llmhosts/1/logs/", reply{json: map[string]any{"llmhost": "m", "logs": "line"}})
	p.route("GET", "llmhosts/1/plan/", reply{fixture: "llmhost_plan"})
	p.route("GET", "llmhosts/report/", reply{fixture: "llmhost_report"})
	hosts, err := p.client.LLMHosts.List(ctx)
	check(t, err)
	equal(t, hosts[0].InferenceEngine, "tei")
	status, err := p.client.LLMHosts.Status(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, status.Status, "inactive")
	logs, err := p.client.LLMHosts.Logs(ctx, smarter.IntID(1), &smarter.LogsParams{Tail: 5})
	check(t, err)
	equal(t, logs.Logs, any("line"))
	equal(t, p.last().URL.Query().Get("tail"), "5")
	_, err = p.client.LLMHosts.Logs(ctx, smarter.IntID(1), nil)
	check(t, err)
	equal(t, p.last().URL.Query().Get("tail"), "200")
	plan, err := p.client.LLMHosts.Plan(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, plan.Resources[0]["kind"], any("PersistentVolumeClaim"))
	report, err := p.client.LLMHosts.Report(ctx)
	check(t, err)
	equal(t, report.Totals["llmhosts"], any(float64(24)))

	state := map[string]any{"llmhost": "m", "status": "pending"}
	p.route("POST", "llmhosts/1/deploy/", reply{json: state, status: 202})
	p.route("POST", "llmhosts/1/undeploy/", reply{json: state})
	deployed, err := p.client.LLMHosts.Deploy(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, deployed.Status, "pending")
	_, err = p.client.LLMHosts.Undeploy(ctx, smarter.IntID(1), nil)
	check(t, err)
	equal(t, p.last().URL.Query().Has("purge"), false)
	_, err = p.client.LLMHosts.Undeploy(ctx, smarter.IntID(1), &smarter.UndeployParams{Purge: true})
	check(t, err)
	equal(t, p.last().URL.Query().Get("purge"), "true")

	p.route("GET", "llmhosts/discover/", reply{fixture: "llmhost_discover"})
	models, err := p.client.LLMHosts.Discover(ctx, &smarter.DiscoverParams{Query: "minilm", Limit: 2})
	check(t, err)
	equal(t, models[0].Repository, "sentence-transformers/all-MiniLM-L6-v2")
	query, _ := url.ParseQuery(p.last().URL.RawQuery)
	jsonEqual(t, query, url.Values{"q": {"minilm"}, "catalog": {"huggingface"}, "limit": {"2"}})
	_, err = p.client.LLMHosts.Discover(ctx, nil)
	check(t, err)
	p.route("GET", "llmhosts/discover/manifest/", reply{json: map[string]any{"manifest": loadJSON(t, "sam/LLMHost.example")}})
	drafted, err := p.client.LLMHosts.DiscoverManifest(ctx, "sentence-transformers/all-MiniLM-L6-v2", &smarter.DiscoverManifestParams{Engine: "tei"})
	check(t, err)
	equal(t, drafted.Kind, "LLMHost")
	equal(t, p.last().URL.Query().Get("engine"), "tei")
	_, err = p.client.LLMHosts.DiscoverManifest(ctx, "x/y", nil)
	check(t, err)
	if _, err := p.client.LLMHosts.Status(ctx, smarter.ID("")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.LLMHosts.Status(ctx, smarter.IntID(2)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "llmhosts/report/", reply{status: 500, json: map[string]any{}})
	p.route("GET", "llmhosts/discover/", reply{status: 500, json: map[string]any{}})
	p.route("GET", "llmhosts/discover/manifest/", reply{status: 500, json: map[string]any{}})
	for _, call := range []func() error{
		func() error { _, err := p.client.LLMHosts.Report(ctx); return err },
		func() error { _, err := p.client.LLMHosts.Discover(ctx, nil); return err },
		func() error { _, err := p.client.LLMHosts.DiscoverManifest(ctx, "x", nil); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrInternalServer) {
			t.Errorf("got %v", err)
		}
	}
}

func TestMCPClients(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "mcpclients/", reply{fixture: "mcpclients"})
	mcpclients, err := p.client.MCPClients.List(ctx)
	check(t, err)
	equal(t, mcpclients[0].Status, "connected")
	if len(mcpclients[0].AllowedTools) == 0 {
		t.Error("no allowed tools")
	}
	p.route("GET", "mcpclients/1/tools/", reply{fixture: "mcp_tools"})
	tools, err := p.client.MCPClients.Tools(ctx, smarter.IntID(1), nil)
	check(t, err)
	equal(t, tools.Tools[0].Name, "aws___read_documentation")
	equal(t, p.last().URL.Query().Has("refresh"), false)
	_, err = p.client.MCPClients.Tools(ctx, smarter.IntID(1), &smarter.MCPToolsParams{Refresh: true})
	check(t, err)
	equal(t, p.last().URL.Query().Get("refresh"), "true")
	p.route("POST", "mcpclients/1/refresh/", reply{json: map[string]any{"name": "aws_knowledge", "status": "connected"}})
	refreshed, err := p.client.MCPClients.Refresh(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, refreshed.Status, "connected")
	result := map[string]any{"name": "aws_knowledge", "tool": "aws___list_regions", "isError": false, "content": "us-east-1"}
	p.route("POST", "mcpclients/1/tools/aws___list_regions/", reply{json: result})
	called, err := p.client.MCPClients.CallTool(ctx, smarter.IntID(1), "aws___list_regions", smarter.JSONObject{"limit": 1})
	check(t, err)
	equal(t, called.IsError, false)
	equal(t, called.Content, "us-east-1")
	jsonEqual(t, p.lastJSON(), map[string]any{"limit": 1})
	_, err = p.client.MCPClients.CallTool(ctx, smarter.IntID(1), "aws___list_regions", nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{})
	for _, call := range []func() error{
		func() error { _, err := p.client.MCPClients.Tools(ctx, smarter.ID(""), nil); return err },
		func() error { _, err := p.client.MCPClients.Refresh(ctx, smarter.ID("")); return err },
		func() error { _, err := p.client.MCPClients.CallTool(ctx, smarter.ID(""), "t", nil); return err },
		func() error { _, err := p.client.MCPClients.CallTool(ctx, smarter.IntID(1), "a/b", nil); return err },
		func() error { _, err := p.client.MCPClients.Tools(ctx, smarter.IntID(2), nil); return err },
		func() error { _, err := p.client.MCPClients.Refresh(ctx, smarter.IntID(2)); return err },
		func() error { _, err := p.client.MCPClients.CallTool(ctx, smarter.IntID(2), "t", nil); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("got %v", err)
		}
	}
}

func TestProvidersAndProxies(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "providers/", reply{json: []any{map[string]any{"name": "openai", "baseUrl": "https://api.openai.com/v1/"}}})
	p.route("GET", "providers/openai/", reply{json: map[string]any{"name": "openai", "isActive": true}})
	p.route("GET", "providers/openai/models/", reply{json: []any{map[string]any{"name": "gpt-4o-mini", "supportsTools": true}}})
	providers, err := p.client.Providers.List(ctx)
	check(t, err)
	equal(t, providers[0].BaseURL, "https://api.openai.com/v1/")
	provider, err := p.client.Providers.Retrieve(ctx, "openai")
	check(t, err)
	equal(t, smarter.Deref(provider.IsActive), true)
	models, err := p.client.Providers.Models(ctx, "openai")
	check(t, err)
	equal(t, smarter.Deref(models[0].SupportsTools), true)

	p.route("GET", "proxy/", reply{fixture: "proxy"})
	proxies, err := p.client.Proxies.List(ctx)
	check(t, err)
	if len(proxies[0].AllowedPaths) == 0 {
		t.Error("no allowed paths")
	}
	equal(t, p.client.Proxies.BaseURL("openai"), apiURL+"proxy/openai/")
	p.route("POST", "proxy/openai/v1/chat/completions", reply{json: map[string]any{"id": "chatcmpl-1"}})
	completion, err := p.client.Proxies.Request(ctx, "openai", "/v1/chat/completions", &smarter.ProxyRequestParams{JSON: map[string]any{"model": "m"}})
	check(t, err)
	jsonEqual(t, completion, map[string]any{"id": "chatcmpl-1"})
	jsonEqual(t, p.lastJSON(), map[string]any{"model": "m"})
	p.route("GET", "proxy/openai/v1/models", reply{json: map[string]any{"data": []any{}}})
	listed, err := p.client.Proxies.Request(ctx, "openai", "v1/models", &smarter.ProxyRequestParams{Method: "get", Query: url.Values{"limit": {"1"}}})
	check(t, err)
	jsonEqual(t, listed, map[string]any{"data": []any{}})
	equal(t, p.last().URL.Query().Get("limit"), "1")
	p.route("POST", "proxy/openai/empty", reply{status: 204})
	empty, err := p.client.Proxies.Request(ctx, "openai", "empty", nil)
	check(t, err)
	equal(t, empty, nil)
	p.route("POST", "proxy/openai/broken", reply{text: "not json"})
	var validation *smarter.APIResponseValidationError
	if _, err := p.client.Proxies.Request(ctx, "openai", "broken", nil); !errors.As(err, &validation) {
		t.Errorf("got %v", err)
	}
	for _, call := range []func() error{
		func() error { _, err := p.client.Providers.Retrieve(ctx, ""); return err },
		func() error { _, err := p.client.Providers.Models(ctx, ""); return err },
		func() error { _, err := p.client.Proxies.Request(ctx, "", "x", nil); return err },
		func() error { _, err := p.client.Proxies.Request(ctx, "nope", "x", nil); return err },
		func() error { _, err := p.client.Providers.Retrieve(ctx, "nope"); return err },
		func() error { _, err := p.client.Providers.Models(ctx, "nope"); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("got %v", err)
		}
	}
	p.route("GET", "providers/", reply{status: 500, json: map[string]any{}})
	p.route("GET", "proxy/", reply{status: 500, json: map[string]any{}})
	if _, err := p.client.Providers.List(ctx); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.Proxies.List(ctx); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}
}

func TestVectorstores(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "vectorstores/", reply{fixture: "vectorstores"})
	p.route("GET", "vectorstores/2/status/", reply{fixture: "vs_status"})
	vectorstores, err := p.client.Vectorstores.List(ctx)
	check(t, err)
	equal(t, vectorstores[0].Backend, "qdrant")
	status, err := p.client.Vectorstores.Status(ctx, smarter.IntID(2))
	check(t, err)
	equal(t, status.Status, "pending")
	vectorstore := first(t, "vectorstores")
	p.route("POST", "vectorstores/2/deploy/", reply{json: vectorstore})
	p.route("POST", "vectorstores/2/undeploy/", reply{json: vectorstore})
	deployed, err := p.client.Vectorstores.Deploy(ctx, smarter.IntID(2))
	check(t, err)
	equal(t, deployed.Name, object(t, vectorstore)["name"].(string))
	_, err = p.client.Vectorstores.Undeploy(ctx, smarter.IntID(2))
	check(t, err)

	added := map[string]any{"documents": []any{map[string]any{"id": 1, "name": "refunds", "status": "pending"}}, "duplicates": []any{}, "queued": false}
	p.route("GET", "vectorstores/2/documents/", reply{json: map[string]any{"documents": added["documents"]}})
	p.route("POST", "vectorstores/2/documents/", reply{json: added, status: 202})
	p.route("DELETE", "vectorstores/2/documents/1/", reply{json: map[string]any{"document": "refunds", "status": "deleting"}})
	documents, err := p.client.Vectorstores.Documents(ctx, smarter.IntID(2))
	check(t, err)
	equal(t, documents.Documents[0].Name, "refunds")
	_, err = p.client.Vectorstores.AddText(ctx, smarter.IntID(2), "30 days", &smarter.AddTextParams{Name: "refunds", Metadata: smarter.JSONObject{"topic": "refunds"}})
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"metadata": map[string]any{"topic": "refunds"}, "name": "refunds", "text": "30 days"})
	_, err = p.client.Vectorstores.AddText(ctx, smarter.IntID(2), "30 days", nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"text": "30 days"})
	_, err = p.client.Vectorstores.AddURL(ctx, smarter.IntID(2), "https://example.com/faq", &smarter.AddURLParams{Metadata: smarter.JSONObject{"a": 1}})
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"metadata": map[string]any{"a": 1}, "url": "https://example.com/faq"})
	_, err = p.client.Vectorstores.AddURL(ctx, smarter.IntID(2), "https://example.com/faq", nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"url": "https://example.com/faq"})
	path := filepath.Join(t.TempDir(), "handbook.txt")
	check(t, os.WriteFile(path, []byte("handbook"), 0o600))
	uploaded, err := p.client.Vectorstores.AddFiles(ctx, smarter.IntID(2), []string{path})
	check(t, err)
	equal(t, uploaded.Queued, false)
	_, params, err := mime.ParseMediaType(p.last().Header.Get("Content-Type"))
	check(t, err)
	form, err := multipart.NewReader(strings.NewReader(string(p.last().body)), params["boundary"]).ReadForm(1 << 20)
	check(t, err)
	equal(t, form.File["files"][0].Filename, "handbook.txt")
	if _, err := p.client.Vectorstores.AddFiles(ctx, smarter.IntID(2), []string{filepath.Join(t.TempDir(), "missing")}); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.Vectorstores.AddFiles(ctx, smarter.ID(""), nil); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	deleted, err := p.client.Vectorstores.DeleteDocument(ctx, smarter.IntID(2), 1)
	check(t, err)
	equal(t, deleted.Status, "deleting")

	results := map[string]any{"vectorstore": "kb", "results": []any{map[string]any{"id": "a", "text": "30 days", "metadata": map[string]any{}, "score": 0.9}}}
	p.route("POST", "vectorstores/2/search/", reply{json: results})
	found, err := p.client.Vectorstores.Search(ctx, smarter.IntID(2), "refunds", nil)
	check(t, err)
	equal(t, smarter.Deref(found.Results[0].Score), 0.9)
	jsonEqual(t, p.lastJSON(), map[string]any{"k": 4, "query": "refunds", "searchType": "similarity"})
	_, err = p.client.Vectorstores.Search(ctx, smarter.IntID(2), "refunds", &smarter.SearchParams{K: 2, SearchType: "mmr", FetchK: smarter.Ptr(8), LambdaMult: smarter.Ptr(0.5), ScoreThreshold: smarter.Ptr(0.1), Filter: smarter.JSONObject{"a": 1}})
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"fetchK": 8, "filter": map[string]any{"a": 1}, "k": 2, "lambdaMult": 0.5, "query": "refunds", "scoreThreshold": 0.1, "searchType": "mmr"})

	snapshot := map[string]any{"id": 3, "name": "s3", "status": "ready"}
	p.route("GET", "vectorstores/2/snapshots/", reply{json: map[string]any{"snapshots": []any{snapshot}}})
	p.route("POST", "vectorstores/2/snapshots/", reply{json: snapshot, status: 201})
	p.route("POST", "vectorstores/2/snapshots/3/restore/", reply{json: map[string]any{"vectorstore": "kb", "restored": "s3", "vectorCount": 10}})
	snapshots, err := p.client.Vectorstores.Snapshots(ctx, smarter.IntID(2))
	check(t, err)
	equal(t, snapshots.Snapshots[0].ID, 3)
	created, err := p.client.Vectorstores.CreateSnapshot(ctx, smarter.IntID(2))
	check(t, err)
	equal(t, created.Name, "s3")
	restored, err := p.client.Vectorstores.Restore(ctx, smarter.IntID(2), 3)
	check(t, err)
	equal(t, smarter.Deref(restored.VectorCount), 10)
	if _, err := p.client.Vectorstores.Status(ctx, smarter.ID("")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.Vectorstores.Status(ctx, smarter.IntID(9)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}
