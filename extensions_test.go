package smarter_test

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
	"testing/iotest"

	smarter "github.com/smarter-sh/smarter-golang"
)

func TestExtensionsRead(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("GET", "extensions/", reply{fixture: "extensions"})
	p.route("GET", "extensions/registry/", reply{fixture: "extension_registry"})
	p.route("GET", "extensions/weather/", reply{fixture: "extension_detail"})

	list, err := p.client.Extensions.List(ctx)
	check(t, err)
	equal(t, list.RegistryEnabled, true)
	equal(t, list.UpdatesAvailable, 0)
	equal(t, len(list.Objects), 3)
	weather := list.Objects[2]
	equal(t, weather.Slug, "weather")
	equal(t, weather.Source, "registry")
	equal(t, weather.TrustBadge, "official")
	equal(t, weather.Enabled, true)
	equal(t, weather.InstalledVersion, "0.2.1")

	registry, err := p.client.Extensions.Registry(ctx, false)
	check(t, err)
	equal(t, p.last().URL.RawQuery, "")
	equal(t, registry.Enabled, true)
	equal(t, registry.Error, "")
	equal(t, registry.Objects[0].Slug, "calculator")
	equal(t, registry.Objects[0].Action, "installed")
	equal(t, registry.Objects[0].Versions[0].Compatible, true)
	equal(t, registry.Objects[0].Versions[0].RequiresSmarter, ">=0.27")
	_, err = p.client.Extensions.Registry(ctx, true)
	check(t, err)
	equal(t, p.last().URL.RawQuery, "refresh=true")

	detail, err := p.client.Extensions.Retrieve(ctx, "weather")
	check(t, err)
	equal(t, detail.Slug, "weather")
	equal(t, detail.Tools[0].FunctionName, "ext_weather_get_weather")
	equal(t, object(t, detail.Tools[0].Parameters)["type"].(string), "object")
	equal(t, detail.Settings[0].Type, "secret")
	equal(t, smarter.Deref(detail.Settings[0].SecretID), 6)
	equal(t, detail.Settings[0].IsSet, true)
	equal(t, detail.Settings[1].Default.(string), "https://api.open-meteo.com/v1/forecast")
	equal(t, detail.Versions[0].IsInstalled, true)
	equal(t, detail.AllAccounts, true)
	equal(t, detail.AvailableSecrets[1].Name, "google_maps_api_key")
	equal(t, detail.LoadError, "")

	p.route("GET", "extensions/nope/", reply{status: 404, json: map[string]any{"detail": "No extension nope is installed."}})
	_, err = p.client.Extensions.Retrieve(ctx, "nope")
	var statusError *smarter.APIStatusError
	if !errors.Is(err, smarter.ErrNotFound) || !errors.As(err, &statusError) || statusError.Message != "No extension nope is installed." {
		t.Errorf("got %v", err)
	}
	p.route("GET", "extensions/", reply{status: 500, text: "boom"})
	p.route("GET", "extensions/registry/", reply{status: 500, text: "boom"})
	for _, call := range []func() error{
		func() error { _, err := p.client.Extensions.List(ctx); return err },
		func() error { _, err := p.client.Extensions.Registry(ctx, false); return err },
	} {
		if err := call(); !errors.Is(err, smarter.ErrInternalServer) {
			t.Errorf("got %v", err)
		}
	}
}

func TestExtensionsActions(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	action := reply{fixture: "extension_action"}
	for _, path := range []string{"check-updates/", "install/", "upload/", "weather/enable/", "weather/disable/", "weather/update/", "weather/rollback/", "weather/configure/", "weather/availability/", "weather/uninstall/"} {
		p.route("POST", "extensions/"+path, action)
	}

	result, err := p.client.Extensions.Enable(ctx, "weather")
	check(t, err)
	equal(t, result.Message, "Enabled Weather.")
	equal(t, result.Extension.Slug, "weather")
	equal(t, len(p.last().body), 0)
	for name, call := range map[string]func() (*smarter.ExtensionAction, error){
		"disable":   func() (*smarter.ExtensionAction, error) { return p.client.Extensions.Disable(ctx, "weather") },
		"update":    func() (*smarter.ExtensionAction, error) { return p.client.Extensions.Update(ctx, "weather") },
		"uninstall": func() (*smarter.ExtensionAction, error) { return p.client.Extensions.Uninstall(ctx, "weather") },
		"check":     func() (*smarter.ExtensionAction, error) { return p.client.Extensions.CheckUpdates(ctx) },
	} {
		t.Run(name, func(t *testing.T) {
			_, err := call()
			check(t, err)
			equal(t, len(p.last().body), 0)
		})
	}

	_, err = p.client.Extensions.Install(ctx, "weather", "0.2.0")
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"slug": "weather", "version": "0.2.0"})
	_, err = p.client.Extensions.Install(ctx, "weather", "")
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"slug": "weather"})

	_, err = p.client.Extensions.Rollback(ctx, "weather", 3)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"versionId": 3})

	_, err = p.client.Extensions.Configure(ctx, "weather", &smarter.ExtensionConfigureParams{
		Values:  map[string]smarter.JSONValue{"weather_api_url": "https://example.com/forecast", "units": nil},
		Secrets: map[string]smarter.ExtensionSecretInput{"google_maps_api_key": {SecretID: smarter.Ptr(6)}, "other_key": {Value: smarter.Ptr("value")}},
	})
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{
		"secrets": map[string]any{"google_maps_api_key": map[string]any{"secretId": 6}, "other_key": map[string]any{"value": "value"}},
		"values":  map[string]any{"units": nil, "weather_api_url": "https://example.com/forecast"},
	})
	_, err = p.client.Extensions.Configure(ctx, "weather", nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{})

	_, err = p.client.Extensions.SetAvailability(ctx, "weather", false, []int{1, 2})
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"accountIds": []int{1, 2}, "allAccounts": false})
	_, err = p.client.Extensions.SetAvailability(ctx, "weather", true, nil)
	check(t, err)
	jsonEqual(t, p.lastJSON(), map[string]any{"accountIds": []int{}, "allAccounts": true})

	_, err = p.client.Extensions.Upload(ctx, bytes.NewReader([]byte("PK\x03\x04")), "weather-0.2.1.zip")
	check(t, err)
	_, params, err := mime.ParseMediaType(p.last().Header.Get("Content-Type"))
	check(t, err)
	form, err := multipart.NewReader(strings.NewReader(string(p.last().body)), params["boundary"]).ReadForm(1 << 20)
	check(t, err)
	equal(t, form.File["archive"][0].Filename, "weather-0.2.1.zip")
	_, err = p.client.Extensions.Upload(ctx, strings.NewReader("PK"), "")
	check(t, err)
	if !strings.Contains(string(p.last().body), `filename="extension.zip"`) {
		t.Errorf("got %s", p.last().body)
	}
	before := p.count()
	if _, err := p.client.Extensions.Upload(ctx, iotest.ErrReader(errors.New("unreadable")), "x.zip"); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	equal(t, p.count(), before)

	p.route("POST", "extensions/weather/disable/", reply{status: 403, json: map[string]any{"detail": "Only a superuser may manage extensions."}})
	if _, err := p.client.Extensions.Disable(ctx, "weather"); !errors.Is(err, smarter.ErrPermissionDenied) {
		t.Errorf("got %v", err)
	}
	p.route("POST", "extensions/weather/enable/", reply{status: 400, json: map[string]any{"detail": "Weather is missing google_maps_api_key."}})
	if _, err := p.client.Extensions.Enable(ctx, "weather"); !errors.Is(err, smarter.ErrBadRequest) {
		t.Errorf("got %v", err)
	}
	p.route("POST", "extensions/weather/uninstall/", reply{json: map[string]any{"message": "Uninstalled Weather.", "extension": nil}})
	uninstalled, err := p.client.Extensions.Uninstall(ctx, "weather")
	check(t, err)
	if uninstalled.Extension != nil {
		t.Errorf("got %+v", uninstalled.Extension)
	}
	before = p.count()
	for _, slug := range []string{"", "a/b", "  "} {
		if _, err := p.client.Extensions.Enable(ctx, slug); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%q: got %v", slug, err)
		}
		if _, err := p.client.Extensions.Retrieve(ctx, slug); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%q: got %v", slug, err)
		}
	}
	equal(t, p.count(), before)
}
