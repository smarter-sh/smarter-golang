package smarter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

func TestClientPlatform(t *testing.T) {
	p := newMockPlatform(t)
	for _, path := range []string{"whoami", "version", "status", "resources"} {
		p.route("POST", "cli/"+path+"/", reply{fixture: path})
	}
	ctx := context.Background()
	whoami, err := p.client.WhoAmI(ctx)
	check(t, err)
	equal(t, whoami.Account.AccountNumber, "1234-5678-9012")
	version, err := p.client.Version(ctx)
	check(t, err)
	if version.API.Version == "" {
		t.Error("no version")
	}
	status, err := p.client.Status(ctx)
	check(t, err)
	if status.Compute == nil {
		t.Error("no compute")
	}
	kinds, err := p.client.Kinds(ctx)
	check(t, err)
	found := false
	for _, kind := range kinds {
		found = found || kind.APIKind == "LLMClient"
	}
	if !found {
		t.Error("LLMClient is not a kind")
	}
}

func TestClientEmptyData(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{json: map[string]any{"api": "smarter.sh/v1"}})
	_, err := p.client.WhoAmI(context.Background())
	var validation *smarter.APIResponseValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("got %v", err)
	}
	p.route("POST", "cli/resources/", reply{status: 500, json: map[string]any{"detail": "x"}})
	if _, err := p.client.Kinds(context.Background()); !errors.Is(err, smarter.ErrInternalServer) {
		t.Fatalf("got %v", err)
	}
}

func TestClientProperties(t *testing.T) {
	p := newMockPlatform(t)
	equal(t, p.client.APIKey(), apiKey)
	equal(t, p.client.BaseURL(), apiURL)
	equal(t, p.client.Environment(), smarter.EnvironmentProd)
	var logged bytes.Buffer
	slog.New(slog.NewTextHandler(&logged, nil)).Info("client", "client", p.client)
	for _, shown := range []string{p.client.String(), fmt.Sprintf("%v %+v %#v", p.client, p.client, p.client), logged.String()} {
		if strings.Contains(shown, apiKey) || !strings.Contains(shown, "****beef") {
			t.Errorf("the api key is not masked: %s", shown)
		}
	}
	if smarter.Version == "" {
		t.Error("no version")
	}
	if p.client.Manifests == nil || p.client.Accounts.Users == nil || p.client.Proxies == nil || p.client.Journal == nil {
		t.Error("a service is missing")
	}
}

func TestClientDo(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "custom/endpoint/", reply{json: map[string]any{"ok": true}})
	var out map[string]bool
	check(t, p.client.Do(context.Background(), "POST", "custom/endpoint/", map[string]any{"a": 1}, &out))
	equal(t, out["ok"], true)
	jsonEqual(t, p.lastJSON(), map[string]any{"a": 1})
	check(t, p.client.Do(context.Background(), "POST", "custom/endpoint/", nil, nil))
}

func TestClientLogging(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{fixture: "whoami"})
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug}))
	_, err := p.makeClient(t, smarter.WithLogger(logger)).WhoAmI(context.Background())
	check(t, err)
	output := logged.String()
	if !strings.Contains(output, "component=smarter") || !strings.Contains(output, "****beef") || strings.Contains(output, apiKey) {
		t.Errorf("unexpected log: %s", output)
	}
}
