package smarter_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
	"github.com/smarter-sh/smarter-golang/sam"
)

func TestManifestYAML(t *testing.T) {
	manifest, err := smarter.ManifestFromYAML("apiVersion: smarter.sh/v1\nkind: Secret\nmetadata:\n  name: s\nspec: {}\n")
	check(t, err)
	equal(t, manifest.Kind, "Secret")
	encoded, err := json.Marshal(manifest)
	check(t, err)
	again, err := smarter.ManifestFromYAML(string(encoded))
	check(t, err)
	equal(t, again.Metadata.Name, "s")
	for _, text := range []string{"- a list", "key: [unclosed", "kind: Secret", "metadata: 3"} {
		if _, err := smarter.ManifestFromYAML(text); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%q: got %v", text, err)
		}
	}
	defaulted, err := smarter.ManifestFromYAML("kind: Secret\nmetadata:\n  name: s\n")
	check(t, err)
	equal(t, defaulted.APIVersion, smarter.APIVersionV1)
}

func TestManifestRoundTrip(t *testing.T) {
	data := object(t, object(t, loadJSON(t, "describe_llmclient"))["data"])
	data["extraSection"] = map[string]any{"a": 1}
	object(t, data["metadata"])["extraField"] = "kept"
	manifest, err := smarter.ManifestFromObject(data)
	check(t, err)
	text, err := manifest.ToYAML()
	check(t, err)
	if strings.Contains(text, "status:") || !strings.HasPrefix(text, "apiVersion:") || strings.Contains(text, "{") {
		t.Errorf("unexpected YAML:\n%s", text)
	}
	loaded, err := smarter.ManifestFromYAML(text)
	check(t, err)
	equal(t, loaded.Status == nil, true)
	jsonEqual(t, loaded.Spec, manifest.Spec)
	equal(t, string(loaded.Extra["extraSection"]), `{"a":1}`)
	equal(t, string(loaded.Metadata.Extra["extraField"]), `"kept"`)
	applied, err := manifest.ToApply()
	check(t, err)
	if strings.Contains(string(applied), `"status"`) {
		t.Error("ToApply kept the status")
	}
	full, err := json.Marshal(manifest)
	check(t, err)
	if !strings.Contains(string(full), `"status"`) || !strings.Contains(string(full), `"extraSection"`) {
		t.Errorf("got %s", full)
	}
	bare, err := json.Marshal(smarter.Manifest{Kind: "Secret", Metadata: smarter.ManifestMetadata{Name: "s"}})
	check(t, err)
	if !strings.Contains(string(bare), `"spec":{}`) {
		t.Errorf("got %s", bare)
	}
}

func TestToSAM(t *testing.T) {
	manifest, err := smarter.ManifestFromObject(object(t, object(t, loadJSON(t, "describe_llmclient"))["data"]))
	check(t, err)
	typed, err := smarter.ToSAM[sam.LLMClient](manifest)
	check(t, err)
	equal(t, typed.Status.RecordLocator, manifest.Status["recordLocator"].(string))
	generic, err := manifest.SAM()
	check(t, err)
	if _, ok := generic.(*sam.LLMClient); !ok {
		t.Errorf("got %T", generic)
	}
	var value *smarter.ValueError
	if _, err := smarter.ToSAM[sam.Guardrail](manifest); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
	unknown := &smarter.Manifest{Kind: "Nope", Metadata: smarter.ManifestMetadata{Name: "x"}}
	if _, err := unknown.SAM(); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
	broken := *manifest
	broken.Spec = smarter.JSONObject{"config": "not an object"}
	if _, err := broken.SAM(); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
	if _, err := smarter.ToSAM[sam.LLMClient](&broken); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
	// a field that the platform adds doesn't fail a decode.
	newer := *manifest
	newer.Spec = smarter.JSONObject{}
	for key, value := range manifest.Spec {
		newer.Spec[key] = value
	}
	newer.Spec["fieldOfANewerPlatform"] = 1
	_, err = smarter.ToSAM[sam.LLMClient](&newer)
	check(t, err)
}

func TestManifestList(t *testing.T) {
	var result smarter.ManifestList
	data, _ := json.Marshal(object(t, loadJSON(t, "get_llmclient"))["data"])
	check(t, json.Unmarshal(data, &result))
	equal(t, len(result.Data.Items), 2)
	equal(t, len(result.Names()), 2)
	users := smarter.ManifestList{Kind: "User", Data: smarter.ManifestListData{Items: []smarter.JSONObject{{"username": "u"}, {"id": 1}, {"name": float64(7)}}}}
	jsonEqual(t, users.Names(), []string{"u", "7"})
}

func TestKindInfo(t *testing.T) {
	var kinds []smarter.KindInfo
	data, _ := json.Marshal(object(t, loadJSON(t, "resources"))["data"])
	check(t, json.Unmarshal(data, &kinds))
	equal(t, kinds[0].APIKind, "Plugin")
	equal(t, kinds[0].Deployable, false)
}

func TestPromptText(t *testing.T) {
	var response smarter.PromptChatAPIPromptResponse
	check(t, json.Unmarshal(loadBytes(t, "prompt.json"), &response))
	if response.Text() == "" || len(response.Data.AssistantMessages()) == 0 {
		t.Error("no text")
	}
	var event smarter.PromptChatAPIProgressEvent
	if err := json.Unmarshal([]byte(`{"type": 3}`), &event); err == nil {
		t.Error("an invalid event decoded")
	}
}

func TestViteManifest(t *testing.T) {
	text := `{
		"index.html": {"file": "assets/index.js", "isEntry": true, "imports": ["_x.js", "_y.js"], "css": ["a.css"]},
		"_x.js": {"file": "assets/x.js", "imports": ["index.html"], "css": ["b.css"]},
		"_custom": {"buildTime": "2026-10-12T01:09:47.988Z", "version": "0.3.0", "buildEnv": "production"}
	}`
	var manifest smarter.ViteManifest
	check(t, json.Unmarshal([]byte(text), &manifest))
	equal(t, manifest.EntryKey(), "index.html")
	jsonEqual(t, manifest.CollectAssets("index.html", "js"), []string{"assets/index.js", "assets/x.js"})
	jsonEqual(t, manifest.CollectAssets("index.html", "css"), []string{"a.css", "b.css"})
	equal(t, manifest.BuildInfo.Version, "0.3.0")
	encoded, err := json.Marshal(manifest)
	check(t, err)
	var keys map[string]any
	check(t, json.Unmarshal(encoded, &keys))
	equal(t, len(keys), 3)
	var empty smarter.ViteManifest
	check(t, json.Unmarshal([]byte(`{}`), &empty))
	equal(t, empty.EntryKey(), "")
	equal(t, len(smarter.ReactAppAssets{JS: []string{"a.js"}}.CSS), 0)
	for _, bad := range []string{`[]`, `{"a": 3}`, `{"_custom": 3}`} {
		if err := json.Unmarshal([]byte(bad), &empty); err == nil {
			t.Errorf("%s decoded", bad)
		}
	}
}
