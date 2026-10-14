package smarter_test

// Test the SDK against a running Smarter platform. These tests are skipped unless
// SMARTER_TEST_LIVE=true. They read the client's configuration as the SDK always does
// (SMARTER_API_KEY, SMARTER_BASE_URL, ...), e.g.:
//
//	SMARTER_TEST_LIVE=true SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=... go test -run Live -v .
//
// They decode the platform's real responses into the SDK's structs. They prompt an LLMClient, which
// makes billable LLM requests, and they apply and delete a throwaway Guardrail. They never deploy
// anything: deploy creates real, billable infrastructure. SMARTER_TEST_LLMCLIENT names the LLMClient
// to prompt (default calculator, which the platform creates as an example). These are the Python
// SDK's tests/live/test_live.py.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	smarter "github.com/smarter-sh/smarter-golang"
	"github.com/smarter-sh/smarter-golang/sam"
)

func liveClient(t *testing.T) *smarter.Client {
	t.Helper()
	if !strings.EqualFold(os.Getenv("SMARTER_TEST_LIVE"), "true") {
		t.Skip("set SMARTER_TEST_LIVE=true to test against a running platform")
	}
	client, err := smarter.NewClient(smarter.WithTimeout(3 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func liveLLMClient() string {
	if name := os.Getenv("SMARTER_TEST_LLMCLIENT"); name != "" {
		return name
	}
	return "calculator"
}

// skipPlatformError skips the test with the platform's message when the platform failed, and fails
// it for an SDK error.
func skipPlatformError(t *testing.T, what string, err error) {
	t.Helper()
	var status *smarter.APIStatusError
	if errors.As(err, &status) {
		t.Skipf("the platform's %s failed: %v", what, err)
	}
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func TestLivePlatform(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	me, err := client.WhoAmI(ctx)
	check(t, err)
	if me.User.Username == "" {
		t.Error("no username")
	}
	version, err := client.Version(ctx)
	check(t, err)
	if version.API.Version == "" {
		t.Error("no version")
	}
	_, err = client.Status(ctx)
	check(t, err)
	kinds, err := client.Kinds(ctx)
	check(t, err)
	if !slices.ContainsFunc(kinds, func(kind smarter.KindInfo) bool { return kind.APIKind == "LLMClient" }) {
		t.Error("LLMClient is not a kind")
	}
}

func TestLiveLists(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	for name, list := range map[string]func() error{
		"llmclients":     func() error { _, err := client.LLMClients.List(ctx); return err },
		"guardrails":     func() error { _, err := client.Guardrails.List(ctx); return err },
		"mcpclients":     func() error { _, err := client.MCPClients.List(ctx); return err },
		"secrets":        func() error { _, err := client.Secrets.List(ctx); return err },
		"plugins":        func() error { _, err := client.Plugins.List(ctx); return err },
		"llmhosts":       func() error { _, err := client.LLMHosts.List(ctx); return err },
		"orchestrators":  func() error { _, err := client.Orchestrators.List(ctx); return err },
		"vectorsearches": func() error { _, err := client.Vectorsearches.List(ctx); return err },
		"accounts":       func() error { _, err := client.Accounts.List(ctx); return err },
		"users":          func() error { _, err := client.Accounts.Users.List(ctx); return err },
		"contacts":       func() error { _, err := client.Accounts.Contacts.List(ctx); return err },
		"user profiles":  func() error { _, err := client.Accounts.UserProfiles.List(ctx); return err },
		"sessions":       func() error { _, err := client.History.Sessions(ctx); return err },
		"tool calls":     func() error { _, err := client.History.ToolCalls(ctx); return err },
		"plugin usage":   func() error { _, err := client.History.PluginUsage(ctx); return err },
	} {
		t.Run(name, func(t *testing.T) { check(t, list()) })
	}
}

func TestLiveRetrieves(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	users, err := client.Accounts.Users.List(ctx)
	check(t, err)
	if len(users) > 0 {
		user, err := client.Accounts.Users.Retrieve(ctx, smarter.IntID(users[0].ID))
		check(t, err)
		equal(t, user.ID, users[0].ID)
	}
	profiles, err := client.Accounts.UserProfiles.List(ctx)
	check(t, err)
	if len(profiles) > 0 {
		profile, err := client.Accounts.UserProfiles.Retrieve(ctx, smarter.IntID(profiles[0].ID))
		check(t, err)
		equal(t, profile.ID, profiles[0].ID)
	}
	contacts, err := client.Accounts.Contacts.List(ctx)
	check(t, err)
	if len(contacts) > 0 {
		contact, err := client.Accounts.Contacts.Retrieve(ctx, smarter.IntID(contacts[0].ID))
		check(t, err)
		equal(t, contact.ID, contacts[0].ID)
	}
	sessions, err := client.History.Sessions(ctx)
	check(t, err)
	if len(sessions) > 0 {
		session, err := client.History.Session(ctx, smarter.IntID(sessions[0].ID))
		check(t, err)
		equal(t, session.ID, sessions[0].ID)
	}
	toolCalls, err := client.History.ToolCalls(ctx)
	check(t, err)
	if len(toolCalls) > 0 {
		toolCall, err := client.History.ToolCall(ctx, smarter.IntID(toolCalls[0].ID))
		check(t, err)
		equal(t, toolCall.ID, toolCalls[0].ID)
	}
	usage, err := client.History.PluginUsage(ctx)
	check(t, err)
	if len(usage) > 0 {
		use, err := client.History.PluginUse(ctx, smarter.IntID(usage[0].ID))
		check(t, err)
		equal(t, use.ID, usage[0].ID)
	}
}

func TestLiveOptionalAPIs(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	for name, call := range map[string]func() error{
		"vectorstores": func() error { _, err := client.Vectorstores.List(ctx); return err },
		"proxies":      func() error { _, err := client.Proxies.List(ctx); return err },
		"journal":      func() error { _, err := client.Journal.List(ctx, nil); return err },
		"providers":    func() error { _, err := client.Providers.List(ctx); return err },
	} {
		t.Run(name, func(t *testing.T) { skipPlatformError(t, name, call()) })
	}
}

func TestLiveManifestKinds(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	kinds, err := client.Kinds(ctx)
	check(t, err)
	for _, kind := range kinds {
		t.Run(kind.APIKind, func(t *testing.T) {
			_, err := client.Manifests.Get(ctx, kind.APIKind, "")
			skipPlatformError(t, "get "+kind.APIKind, err)
			example, err := client.Manifests.ExampleManifest(ctx, kind.APIKind)
			check(t, err)
			equal(t, example.Kind, kind.APIKind)
			schema, err := client.Manifests.JSONSchema(ctx, kind.APIKind)
			check(t, err)
			if _, ok := schema["properties"]; !ok {
				t.Error("no properties")
			}
		})
	}
}

func TestLiveSAMStructs(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	plugins, err := client.Plugins.List(ctx)
	check(t, err)
	kinds, err := client.Kinds(ctx)
	check(t, err)
	for _, kind := range kinds {
		t.Run(kind.APIKind, func(t *testing.T) {
			var names []string
			if slices.Contains(smarter.PluginKinds, kind.APIKind) {
				// the platform's get of a plugin kind returns every plugin, so they are picked by kind.
				for _, plugin := range plugins {
					if plugin.Kind == kind.APIKind {
						names = append(names, plugin.Name)
					}
				}
			} else {
				list, err := client.Manifests.Get(ctx, kind.APIKind, "")
				skipPlatformError(t, "get "+kind.APIKind, err)
				names = list.Names()
			}
			if len(names) == 0 {
				t.Skipf("there is no %s to describe", kind.APIKind)
			}
			manifest, err := client.Manifests.Describe(ctx, kind.APIKind, names[0])
			skipPlatformError(t, "describe "+kind.APIKind+" "+names[0], err)
			typed, err := manifest.SAM()
			check(t, err)
			equal(t, typed.SAMKind(), kind.APIKind)
			// the platform returns some names with dashes in get, and underscores in describe.
			equal(t, strings.ReplaceAll(manifest.Metadata.Name, "-", "_"), strings.ReplaceAll(names[0], "-", "_"))
		})
	}
}

func TestLiveManifestLifecycle(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	manifest, err := client.Guardrails.ExampleManifest(ctx)
	check(t, err)
	suffix := make([]byte, 4)
	_, _ = rand.Read(suffix)
	name := "sdk_live_test_" + hex.EncodeToString(suffix)
	manifest.Metadata.Name = name
	validation, err := client.Manifests.Validate(ctx, manifest)
	check(t, err)
	equal(t, validation.Valid, true)
	_, err = client.Guardrails.Apply(ctx, manifest)
	check(t, err)
	t.Cleanup(func() { _, _ = client.Guardrails.Delete(context.Background(), name) })
	described, err := client.Guardrails.Describe(ctx, name)
	check(t, err)
	equal(t, described.Metadata.Name, name)
	_, err = smarter.ToSAM[sam.Guardrail](described)
	check(t, err)
	listed, err := client.Manifests.Get(ctx, "Guardrail", name)
	check(t, err)
	jsonEqual(t, listed.Names(), []string{name})
	guardrail, err := client.Guardrails.Find(ctx, name)
	check(t, err)
	retrieved, err := client.Guardrails.Retrieve(ctx, smarter.IntID(guardrail.ID))
	check(t, err)
	equal(t, retrieved.Name, name)
	stage := smarter.StageInput
	if guardrail.Stage == "output" {
		stage = smarter.StageOutput
	}
	evaluation, err := client.Guardrails.Evaluate(ctx, smarter.IntID(guardrail.ID), "My SSN is 078-05-1120.", &smarter.GuardrailEvaluateParams{Stage: stage})
	check(t, err)
	equal(t, evaluation.Guardrail, name)
	_, err = client.Guardrails.Events(ctx, smarter.IntID(guardrail.ID), nil)
	check(t, err)
}

func TestLivePrompt(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	llmclient, err := client.LLMClients.Find(ctx, liveLLMClient())
	check(t, err)
	response, err := client.LLMClients.Prompt(ctx, smarter.IntID(llmclient.ID), smarter.Text("What is 17 times 23? Use the calculator."), nil)
	check(t, err)
	if !strings.Contains(strings.ReplaceAll(response.Text(), ",", ""), "391") {
		t.Errorf("got %q", response.Text())
	}
	id := smarter.IntID(llmclient.ID)
	_, err = client.LLMClients.Functions(ctx, id)
	check(t, err)
	_, err = client.LLMClients.Plugins(ctx, id)
	check(t, err)
	_, err = client.LLMClients.APIKeys(ctx, id)
	check(t, err)
	_, err = client.LLMClients.CustomDomains(ctx, id)
	check(t, err)
}

func TestLiveStreamAndChat(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	llmclient, err := client.LLMClients.Find(ctx, liveLLMClient())
	check(t, err)
	chat, err := client.LLMClients.Chat(ctx, llmclient.Ref(), nil)
	check(t, err)
	response, err := chat.Send(ctx, "What is 2 to the 10th power? Use the calculator.")
	check(t, err)
	if !strings.Contains(strings.ReplaceAll(response.Text(), ",", ""), "1024") {
		t.Errorf("got %q", response.Text())
	}
	stream := chat.Stream(ctx, "Divide that by 4, with the calculator.")
	defer func() { _ = stream.Close() }()
	var types []string
	for stream.Next() {
		types = append(types, stream.Current().Type)
	}
	check(t, stream.Err())
	if !slices.Contains(types, "llm_request") {
		t.Errorf("no llm_request event in %v", types)
	}
	streamed, err := stream.Response()
	check(t, err)
	if !strings.Contains(streamed.Text(), "256") {
		t.Errorf("got %q", streamed.Text())
	}
	last := streamed.Data.Messages[len(streamed.Data.Messages)-1]
	jsonEqual(t, chat.Messages[len(chat.Messages)-1], last)
	resumed, err := client.LLMClients.Chat(ctx, smarter.IntID(llmclient.ID), &smarter.PromptParams{SessionKey: chat.SessionKey()})
	check(t, err)
	equal(t, resumed.SessionKey(), chat.SessionKey())
	if len(resumed.Messages) == 0 {
		t.Error("the resumed session has no messages")
	}
}

func TestLiveMCPAndLLMHosts(t *testing.T) {
	client, ctx := liveClient(t), context.Background()
	mcpclients, err := client.MCPClients.List(ctx)
	check(t, err)
	if len(mcpclients) > 0 {
		_, err := client.MCPClients.Tools(ctx, smarter.IntID(mcpclients[0].ID), nil)
		check(t, err)
	}
	llmhosts, err := client.LLMHosts.List(ctx)
	check(t, err)
	_, err = client.LLMHosts.Report(ctx)
	check(t, err)
	if len(llmhosts) > 0 {
		_, err := client.LLMHosts.Status(ctx, smarter.IntID(llmhosts[0].ID))
		check(t, err)
		_, err = client.LLMHosts.Plan(ctx, smarter.IntID(llmhosts[0].ID))
		check(t, err)
	}
}
