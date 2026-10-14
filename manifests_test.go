package smarter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
	"github.com/smarter-sh/smarter-golang/sam"
)

func exampleGuardrail(t *testing.T) *smarter.Manifest {
	t.Helper()
	data := object(t, object(t, loadJSON(t, "example_guardrail"))["data"])
	manifest, err := smarter.ManifestFromObject(data)
	check(t, err)
	return manifest
}

func TestManifestsApply(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/apply/", reply{json: map[string]any{"message": "applied", "api": "smarter.sh/v1"}})
	manifest := exampleGuardrail(t)
	manifest.Status = smarter.JSONObject{"x": 1}
	response, err := p.client.Manifests.Apply(context.Background(), manifest)
	check(t, err)
	equal(t, response.Message, "applied")
	sent := object(t, p.lastJSON())
	equal(t, object(t, sent["metadata"])["name"], any(manifest.Metadata.Name))
	if _, ok := sent["status"]; ok {
		t.Error("the status was sent")
	}
}

func TestManifestsApplyRejected(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/apply/", reply{status: 400, fixture: "get_bad"})
	_, err := p.client.Manifests.Apply(context.Background(), exampleGuardrail(t))
	var statusError *smarter.APIStatusError
	if !errors.As(err, &statusError) || !errors.Is(err, smarter.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
	equal(t, statusError.ErrorClass(), "APIV1CLIViewBadRequestError")
}

func TestManifestsValidate(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/validate/", reply{fixture: "validate"})
	validation, err := p.client.Manifests.Validate(context.Background(), exampleGuardrail(t))
	check(t, err)
	equal(t, validation.Valid, true)
	p.route("POST", "cli/validate/", reply{json: map[string]any{"message": "no data"}})
	validation, err = p.client.Manifests.Validate(context.Background(), exampleGuardrail(t))
	check(t, err)
	equal(t, validation.Valid, false)
	p.route("POST", "cli/validate/", reply{status: 400, json: map[string]any{}})
	if _, err := p.client.Manifests.Validate(context.Background(), exampleGuardrail(t)); err == nil {
		t.Error("no error")
	}
}

func TestManifestsGet(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/get/LLMClient/", reply{fixture: "get_llmclient"})
	result, err := p.client.Manifests.Get(context.Background(), "LLMClient", "")
	check(t, err)
	equal(t, len(result.Data.Items), 2)
	name := result.Names()[0]
	filtered, err := p.client.Manifests.Get(context.Background(), "LLMClient", name)
	check(t, err)
	jsonEqual(t, filtered.Names(), []string{name})
	equal(t, filtered.Metadata.Count, 1)
	equal(t, p.last().URL.Query().Get("name"), name)
	p.route("POST", "cli/get/LLMClient/", reply{json: map[string]any{"api": "smarter.sh/v1"}})
	empty, err := p.client.Manifests.Get(context.Background(), "LLMClient", "")
	check(t, err)
	equal(t, len(empty.Data.Items), 0)
	if _, err := p.client.Manifests.Get(context.Background(), "a/b", ""); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
}

func TestManifestsDescribe(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/describe/LLMClient/", reply{fixture: "describe_llmclient"})
	manifest, err := p.client.Manifests.Describe(context.Background(), "LLMClient", "calculator")
	check(t, err)
	equal(t, manifest.Metadata.Name, "calculator")
	if manifest.Status == nil {
		t.Error("no status")
	}
	equal(t, p.last().URL.Query().Get("name"), "calculator")
	p.route("POST", "cli/describe/LLMClient/", reply{status: 503, fixture: "describe_missing"})
	if _, err := p.client.Manifests.Describe(context.Background(), "LLMClient", "nope"); !errors.Is(err, smarter.ErrInternalServer) {
		t.Errorf("got %v", err)
	}
	p.route("POST", "cli/describe/LLMClient/", reply{json: map[string]any{"api": "smarter.sh/v1"}})
	var value *smarter.ValueError
	if _, err := p.client.Manifests.Describe(context.Background(), "LLMClient", "nope"); !errors.As(err, &value) {
		t.Errorf("got %v", err)
	}
}

func TestManifestsVerbs(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	verbs := map[string]func() (string, error){
		"delete": func() (string, error) {
			r, err := p.client.Manifests.Delete(ctx, "LLMClient", "calculator")
			return message(r, err)
		},
		"deploy": func() (string, error) {
			r, err := p.client.Manifests.Deploy(ctx, "LLMClient", "calculator")
			return message(r, err)
		},
		"undeploy": func() (string, error) {
			r, err := p.client.Manifests.Undeploy(ctx, "LLMClient", "calculator")
			return message(r, err)
		},
		"logs": func() (string, error) {
			r, err := p.client.Manifests.Logs(ctx, "LLMClient", "calculator")
			if err != nil {
				return "", err
			}
			return r.Message, nil
		},
	}
	for verb, call := range verbs {
		p.route("POST", "cli/"+verb+"/LLMClient/", reply{json: map[string]any{"message": verb}})
		got, err := call()
		check(t, err)
		equal(t, got, verb)
		equal(t, p.last().URL.Query().Get("name"), "calculator")
		p.route("POST", "cli/"+verb+"/LLMClient/", reply{status: 404, json: map[string]any{}})
		if _, err := call(); !errors.Is(err, smarter.ErrNotFound) {
			t.Errorf("%s: got %v", verb, err)
		}
	}
}

func message(r *smarter.JournalMessage, err error) (string, error) {
	if err != nil {
		return "", err
	}
	return r.Message, nil
}

func TestManifestsExampleAndSchema(t *testing.T) {
	p := newMockPlatform(t)
	ctx := context.Background()
	p.route("POST", "cli/example-manifest/Guardrail/", reply{fixture: "example_guardrail"})
	p.route("POST", "cli/json-schema/Secret/", reply{fixture: "schema_secret"})
	example, err := p.client.Manifests.ExampleManifest(ctx, "Guardrail")
	check(t, err)
	equal(t, example.Kind, "Guardrail")
	schema, err := p.client.Manifests.JSONSchema(ctx, "Secret")
	check(t, err)
	if _, ok := schema["properties"]; !ok {
		t.Error("no properties")
	}
	p.route("POST", "cli/example-manifest/Guardrail/", reply{json: map[string]any{"api": "smarter.sh/v1"}})
	if _, err := p.client.Manifests.ExampleManifest(ctx, "Guardrail"); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	p.route("POST", "cli/json-schema/Secret/", reply{json: map[string]any{"api": "smarter.sh/v1"}})
	schema, err = p.client.Manifests.JSONSchema(ctx, "Secret")
	check(t, err)
	equal(t, len(schema), 0)
	for _, kind := range []string{"", "a b", "a/b"} {
		if _, err := p.client.Manifests.ExampleManifest(ctx, kind); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%q: got %v", kind, err)
		}
	}
}

func TestManifestInputs(t *testing.T) {
	manifest := exampleGuardrail(t)
	typed, err := smarter.ToSAM[sam.Guardrail](manifest)
	check(t, err)
	fromSAM, err := smarter.ManifestFromSAM(typed)
	check(t, err)
	equal(t, fromSAM.Kind, "Guardrail")
	text, err := manifest.ToYAML()
	check(t, err)
	fromYAML, err := smarter.ManifestFromYAML(text)
	check(t, err)
	equal(t, fromYAML.Metadata.Name, manifest.Metadata.Name)
	path := filepath.Join(t.TempDir(), "guardrail.yaml")
	check(t, os.WriteFile(path, []byte(text), 0o600))
	fromFile, err := smarter.ManifestFromFile(path)
	check(t, err)
	equal(t, fromFile.Kind, "Guardrail")
	if _, err := smarter.ManifestFromFile(filepath.Join(t.TempDir(), "missing.yaml")); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
}
