package smarter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

func sse(events ...[2]any) string {
	var b strings.Builder
	for _, event := range events {
		data, _ := json.Marshal(event[1])
		fmt.Fprintf(&b, "event: %s\ndata: %s\n\n", event[0], data)
	}
	return b.String()
}

func TestLLMClientsReads(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "llm-clients/", reply{fixture: "llmclients"})
	p.route("GET", "llm-clients/5/", reply{fixture: "llmclient"})
	p.route("GET", "llm-clients/rMTAwMDAwNQx/", reply{fixture: "llmclient"})
	llmclients, err := p.client.LLMClients.List(ctx)
	check(t, err)
	equal(t, len(llmclients), 2)
	byID, err := p.client.LLMClients.Retrieve(ctx, smarter.IntID(5))
	check(t, err)
	equal(t, byID.Name, "calculator")
	byHash, err := p.client.LLMClients.Retrieve(ctx, smarter.ID("rMTAwMDAwNQx"))
	check(t, err)
	equal(t, byHash.Name, "calculator")
	found, err := p.client.LLMClients.Find(ctx, llmclients[1].Name)
	check(t, err)
	equal(t, found.ID, llmclients[1].ID)
	equal(t, found.Ref(), smarter.ID(llmclients[1].HashedID))
	equal(t, smarter.Resource{ID: 3}.Ref(), smarter.IntID(3))
	if _, err := p.client.LLMClients.Retrieve(ctx, smarter.ID("a/b")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "llm-clients/", reply{status: 500, json: map[string]any{}})
	if _, err := p.client.LLMClients.Find(ctx, "x"); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}
	p.route("GET", "llm-clients/6/", reply{status: 404, json: map[string]any{}})
	if _, err := p.client.LLMClients.Retrieve(ctx, smarter.IntID(6)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestLLMClientsTools(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "llm-clients/", reply{fixture: "llmclients"})
	p.route("GET", "llm-clients/5/", reply{fixture: "llmclient"})
	llmclient, err := p.client.LLMClients.Retrieve(ctx, smarter.IntID(5))
	check(t, err)
	equal(t, len(llmclient.MCPClients), 0)
	equal(t, len(llmclient.Guardrails), 0)
	equal(t, len(llmclient.Tools), 1)
	equal(t, llmclient.Tools[0], smarter.LLMClientTool{ID: 1, Name: "calculator", Kind: "Extension"})
	llmclients, err := p.client.LLMClients.List(ctx)
	check(t, err)
	for _, item := range llmclients {
		equal(t, len(item.Tools), 1)
		equal(t, item.Tools[0].Kind, "Plugin")
	}
}

func TestLLMClientsChildren(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "llm-clients/1/plugins/", reply{fixture: "llmclient_plugins"})
	p.route("GET", "llm-clients/5/extensions", reply{fixture: "llmclient_extensions"})
	apiKeyID := "0b8e1c55-2f3a-4c1d-9a51-7d2f4e6b8a90"
	p.route("GET", "llm-clients/5/apikeys/", reply{json: []any{map[string]any{"id": 1, "llmclient": 5, "apiKey": apiKeyID}}})
	p.route("GET", "llm-clients/5/customdomains", reply{json: []any{map[string]any{"id": 3, "name": "example_com", "domainName": "example.com", "verificationStatus": "Failed"}}})
	plugins, err := p.client.LLMClients.Plugins(ctx, smarter.IntID(1))
	check(t, err)
	equal(t, plugins[0].PluginMeta.PluginClass, "sql")
	extensions, err := p.client.LLMClients.Extensions(ctx, smarter.IntID(5))
	check(t, err)
	equal(t, extensions[0].Name, "calculator")
	equal(t, *extensions[0].Extension, 1)
	keys, err := p.client.LLMClients.APIKeys(ctx, smarter.IntID(5))
	check(t, err)
	equal(t, keys[0].APIKey, apiKeyID)
	domains, err := p.client.LLMClients.CustomDomains(ctx, smarter.IntID(5))
	check(t, err)
	equal(t, domains[0].DomainName+" "+domains[0].VerificationStatus, "example.com Failed")
	if _, err := p.client.LLMClients.Plugins(ctx, smarter.ID("")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	if _, err := p.client.LLMClients.Plugins(ctx, smarter.IntID(9)); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestLLMClientsConfig(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/config/", reply{fixture: "config"})
	config, err := p.client.LLMClients.Config(context.Background(), smarter.IntID(5), &smarter.PromptParams{SessionKey: "abc"})
	check(t, err)
	equal(t, config.LLMClient.Name, "calculator")
	equal(t, *config.Extensions.MetaData.TotalExtensions, 1)
	equal(t, config.Extensions.Extensions[0].Name+" "+config.Extensions.Extensions[0].Version, "calculator 1.0.0")
	equal(t, config.Extensions.Extensions[0].Enabled, true)
	jsonEqual(t, p.lastJSON(), map[string]any{"session_key": "abc"})
	_, err = p.client.LLMClients.Config(context.Background(), smarter.IntID(5), nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"session_key": nil})
	if _, err := p.client.LLMClients.Config(context.Background(), smarter.ID("a/b"), nil); err == nil {
		t.Error("no error")
	}
}

func TestLLMClientsPrompt(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/prompt/", reply{fixture: "prompt"})
	response, err := p.client.LLMClients.Prompt(context.Background(), smarter.IntID(5), smarter.Text("What is 17 times 23?"), &smarter.PromptParams{SessionKey: "k"})
	check(t, err)
	if response.Text() == "" {
		t.Error("no text")
	}
	jsonEqual(t, p.lastJSON(), map[string]any{"session_key": "k", "messages": []any{map[string]any{"role": "user", "content": "What is 17 times 23?"}}})
	if _, err := p.client.LLMClients.Prompt(context.Background(), smarter.ID(""), smarter.Text("x"), nil); err == nil {
		t.Error("no error")
	}
}

func TestLLMClientsPromptFailure(t *testing.T) {
	p := newMockPlatform(t)
	body := object(t, loadJSON(t, "prompt"))
	body["error"] = map[string]any{"errorClass": "LLMProviderError", "description": "bad request", "status": 400}
	p.route("POST", "llm-clients/5/prompt/", reply{status: 400, json: body})
	_, err := p.client.LLMClients.Prompt(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil)
	var statusError *smarter.APIStatusError
	if !errors.As(err, &statusError) || !errors.Is(err, smarter.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
	equal(t, statusError.Message, "bad request")
}

func TestRequestMessages(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/prompt/", reply{fixture: "prompt"})
	thread := []smarter.PromptChatAPIMessage{
		{Role: "system", Content: smarter.TextContent("be brief")},
		{Role: "assistant", ToolCalls: []smarter.PromptChatAPIToolCall{{ID: "c", Type: "function", Function: smarter.PromptChatAPIFunctionCall{Name: "f", Arguments: "{}"}}}},
		{Role: "smarter", Content: smarter.TextContent("charges")},
		{Role: "tool", Content: smarter.TextContent("1"), ToolCallID: "c"},
	}
	_, err := p.client.LLMClients.Prompt(context.Background(), smarter.IntID(5), thread, nil)
	check(t, err)
	sent := object(t, p.lastJSON())["messages"].([]any)
	var roles []string
	for _, message := range sent {
		roles = append(roles, object(t, message)["role"].(string))
	}
	jsonEqual(t, roles, []string{"system", "assistant", "tool"})
	content, ok := object(t, sent[1])["content"]
	if !ok || content != nil {
		t.Errorf("an assistant's tool calls lost their null content: %v", sent[1])
	}
}

func TestLLMClientsStream(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/prompt/", reply{text: string(loadBytes(t, "prompt_sse.txt"))})
	stream := p.client.LLMClients.Stream(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil)
	defer func() { _ = stream.Close() }()
	var events []smarter.PromptChatAPIProgressEvent
	for stream.Next() {
		events = append(events, stream.Current())
	}
	check(t, stream.Err())
	if len(events) == 0 || events[0].Type != "llm_request" || events[0].Fields["type"] != "llm_request" {
		t.Fatalf("got %v", events)
	}
	equal(t, p.last().Header.Get("Accept"), "text/event-stream")
	equal(t, stream.Result().Status, 200)
	response, err := stream.Response()
	check(t, err)
	if response.Text() == "" {
		t.Error("no text")
	}
	if stream.Next() {
		t.Error("a finished stream gave another event")
	}
	check(t, stream.Close())
}

func TestLLMClientsStreamUntilDone(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/prompt/", reply{text: string(loadBytes(t, "prompt_sse.txt"))})
	response, err := p.client.LLMClients.Stream(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil).UntilDone()
	check(t, err)
	if response.Text() == "" {
		t.Error("no text")
	}
}

func TestLLMClientsStreamErrors(t *testing.T) {
	p := newMockPlatform(t)
	body := object(t, loadJSON(t, "prompt"))
	body["error"] = map[string]any{"errorClass": "LLMProviderError", "description": "boom", "status": 400}
	errorOnly := map[string]any{"error": map[string]any{"errorClass": "NotFound", "description": "gone", "status": 404}}
	for _, c := range []struct {
		body    any
		message string
	}{{body, "boom"}, {errorOnly, "gone"}} {
		p.route("POST", "llm-clients/5/prompt/", reply{text: sse([2]any{"result", map[string]any{"status": 400, "response": c.body}})})
		_, err := p.client.LLMClients.Stream(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil).UntilDone()
		var promptError *smarter.PromptError
		if !errors.As(err, &promptError) || promptError.Message != c.message || promptError.Response == nil {
			t.Errorf("got %v", err)
		}
	}
	_, err := p.client.LLMClients.Stream(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil).Response()
	if !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
}

func TestChatSession(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("POST", "llm-clients/5/config/", reply{fixture: "config"})
	p.route("POST", "llm-clients/5/prompt/", reply{fixture: "prompt"})
	config := object(t, object(t, loadJSON(t, "config"))["data"])
	chat, err := p.client.LLMClients.Chat(ctx, smarter.IntID(5), nil)
	check(t, err)
	equal(t, chat.SessionKey(), config["session_key"].(string))
	equal(t, chat.Messages[0].Role, "system")
	before := len(chat.Messages)
	response, err := chat.Send(ctx, "What is 17 times 23?")
	check(t, err)
	sent := object(t, p.lastJSON())
	equal(t, sent["session_key"], any(chat.SessionKey()))
	messages := sent["messages"].([]any)
	jsonEqual(t, messages[len(messages)-1], map[string]any{"role": "user", "content": "What is 17 times 23?"})
	equal(t, len(chat.Messages), before+1+len(response.Data.Messages))
	p.route("POST", "llm-clients/5/prompt/", reply{status: 500, json: map[string]any{}})
	if _, err := chat.Send(ctx, "again"); err == nil {
		t.Error("no error")
	}
	equal(t, len(chat.Messages), before+1+len(response.Data.Messages))
}

func TestChatSessionResumedAndEmpty(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	config := object(t, object(t, loadJSON(t, "config"))["data"])
	object(t, config["history"])["chat_history"] = []any{map[string]any{"role": "user", "content": "earlier"}}
	p.route("POST", "llm-clients/5/config/", reply{json: map[string]any{"api": "smarter.sh/v1", "data": config}})
	chat, err := p.client.LLMClients.Chat(ctx, smarter.IntID(5), &smarter.PromptParams{SessionKey: "k"})
	check(t, err)
	equal(t, chat.Messages[0].Text(), "earlier")
	object(t, config["history"])["chat_history"] = []any{}
	object(t, config["llmclient"])["default_system_role"] = nil
	object(t, config["llmclient"])["app_welcome_message"] = nil
	p.route("POST", "llm-clients/5/config/", reply{json: map[string]any{"api": "smarter.sh/v1", "data": config}})
	chat, err = p.client.LLMClients.Chat(ctx, smarter.IntID(5), nil)
	check(t, err)
	equal(t, len(chat.Messages), 0)
	p.route("POST", "llm-clients/5/config/", reply{status: 404, json: map[string]any{}})
	if _, err := p.client.LLMClients.Chat(ctx, smarter.IntID(5), nil); !errors.Is(err, smarter.ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestChatStream(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("POST", "llm-clients/5/config/", reply{fixture: "config"})
	p.route("POST", "llm-clients/5/prompt/", reply{text: string(loadBytes(t, "prompt_sse.txt"))})
	chat, err := p.client.LLMClients.Chat(ctx, smarter.IntID(5), nil)
	check(t, err)
	before := len(chat.Messages)
	_, err = chat.Stream(ctx, "hi").UntilDone()
	check(t, err)
	if len(chat.Messages) <= before+1 {
		t.Fatalf("the thread was not updated: %d", len(chat.Messages))
	}
	count := len(chat.Messages)
	failed := map[string]any{"error": map[string]any{"errorClass": "NotFound", "description": "gone", "status": 404}}
	p.route("POST", "llm-clients/5/prompt/", reply{handler: func(req *http.Request) (*http.Response, error) {
		return response(req, 200, sse([2]any{"result", map[string]any{"status": 404, "response": failed}}), nil), nil
	}})
	if _, err := chat.StreamMessage(ctx, smarter.UserMessage("again")).UntilDone(); err == nil {
		t.Error("no error")
	}
	equal(t, len(chat.Messages), count)
}

func TestLLMClientsManifestOperations(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("POST", "cli/deploy/LLMClient/", reply{json: map[string]any{"message": "deployed"}})
	p.route("POST", "cli/undeploy/LLMClient/", reply{json: map[string]any{"message": "undeployed"}})
	deployed, err := p.client.LLMClients.Deploy(ctx, "calculator")
	check(t, err)
	equal(t, deployed.Message, "deployed")
	undeployed, err := p.client.LLMClients.Undeploy(ctx, "calculator")
	check(t, err)
	equal(t, undeployed.Message, "undeployed")
	var value *smarter.ValueError
	if _, err := p.client.LLMClients.Apply(ctx, exampleGuardrail(t)); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
}

func TestMessageContent(t *testing.T) {
	var message smarter.PromptChatAPIMessage
	check(t, json.Unmarshal([]byte(`{"role": "user", "content": [{"type": "text", "text": "a"}, {"type": "image_url", "image_url": {"url": "data:,"}}, {"type": "text", "text": "b"}]}`), &message))
	equal(t, message.Text(), "a\n\nb")
	encoded, err := json.Marshal(message)
	check(t, err)
	if !strings.Contains(string(encoded), `"content":[{"type":"text","text":"a"}`) {
		t.Errorf("got %s", encoded)
	}
	empty, _ := json.Marshal(smarter.PromptChatAPIMessage{Role: "user", Content: smarter.PartsContent()})
	equal(t, string(empty), `{"role":"user","content":[]}`)
	equal(t, smarter.PromptChatAPIMessage{Role: "assistant"}.Text(), "")
	equal(t, smarter.PromptChatAPIPromptResult{}.Text(), "")
}
