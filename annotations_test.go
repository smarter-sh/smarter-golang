package smarter_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	smarter "github.com/smarter-sh/smarter-golang"
	"github.com/smarter-sh/smarter-golang/sam"
)

func TestNormalizeAnnotations(t *testing.T) {
	empty, err := smarter.NormalizeAnnotations(nil)
	check(t, err)
	jsonEqual(t, empty, map[string]string{})

	fromMap, err := smarter.NormalizeAnnotations(map[string]any{"acme.com/a": "x", "acme.com/b": true, "acme.com/c": nil})
	check(t, err)
	jsonEqual(t, fromMap, map[string]string{"acme.com/a": "x", "acme.com/b": "true", "acme.com/c": ""})

	// the legacy list: a repeated key's last value wins.
	fromList, err := smarter.NormalizeAnnotations([]any{
		map[string]any{"acme.com/a": 1.5},
		map[string]any{"acme.com/b": false, "acme.com/a": "last"},
	})
	check(t, err)
	jsonEqual(t, fromList, map[string]string{"acme.com/a": "last", "acme.com/b": "false"})

	for _, bad := range []any{[]any{"not a map"}, "text", 42.0} {
		if _, err := smarter.NormalizeAnnotations(bad); !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%v: got %v", bad, err)
		}
	}
}

func TestAnnotationValue(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{nil, ""},
		{"text", "text"},
		{true, "true"},
		{false, "false"},
		{json.Number("42"), "42"},
		{3.0, "3.0"},
		{float32(1.5), "1.5"},
		{1234567.5, "1234567.5"},
		{1e20, "1e+20"},
		{0.00001, "1e-05"},
		{time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC), "2026-10-28"},
		{time.Date(2026, 10, 28, 10, 30, 0, 0, time.UTC), "2026-10-28T10:30:00+00:00"},
		{time.Date(2026, 10, 28, 10, 30, 0, 500000000, time.UTC), "2026-10-28T10:30:00.500000+00:00"},
		{42, "42"},
		{[]any{1.0, "a"}, `[1,"a"]`},
		{map[string]any{"k": "v"}, `{"k":"v"}`},
	} {
		equal(t, smarter.AnnotationValue(tc.value), tc.want)
	}
}

func TestAnnotationsUnmarshal(t *testing.T) {
	var resource smarter.Resource
	check(t, json.Unmarshal([]byte(`{"id": 1, "name": "a", "annotations": [{"acme.com/a": 1, "acme.com/b": [1, 2]}], "detailUrl": "/dashboard/x/"}`), &resource))
	jsonEqual(t, resource.Annotations, map[string]string{"acme.com/a": "1", "acme.com/b": "[1,2]"})
	equal(t, resource.DetailURL, "/dashboard/x/")

	var none smarter.Resource
	check(t, json.Unmarshal([]byte(`{"id": 1, "name": "a", "annotations": null}`), &none))
	equal(t, len(none.Annotations), 0)

	var bad smarter.Resource
	if err := json.Unmarshal([]byte(`{"annotations": ["x"]}`), &bad); err == nil {
		t.Error("no error for a list of strings")
	}
	if err := json.Unmarshal([]byte(`{"annotations": {`), &bad); err == nil {
		t.Error("no error for malformed JSON")
	}
	var annotations smarter.Annotations
	if err := annotations.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Error("no error for malformed JSON")
	}
}

func TestManifestLegacyAnnotations(t *testing.T) {
	// a manifest of Smarter 0.37 or earlier, with unquoted YAML scalars.
	manifest, err := smarter.ManifestFromYAML(`apiVersion: smarter.sh/v1
kind: Guardrail
metadata:
  name: legacy
  description: A legacy manifest.
  version: 1.0.0
  annotations:
    - acme.com/active: true
    - acme.com/count: 42
    - acme.com/ratio: 1.0
    - acme.com/since: 2026-10-28
spec:
  config:
    stage: input
    category: pii
    strategy: detector
`)
	check(t, err)
	want := map[string]string{"acme.com/active": "true", "acme.com/count": "42", "acme.com/ratio": "1.0", "acme.com/since": "2026-10-28"}
	jsonEqual(t, manifest.Metadata.Annotations, want)

	// ManifestFromObject converts too, and leaves the caller's object unchanged.
	object := map[string]any{
		"apiVersion": "smarter.sh/v1", "kind": "Guardrail", "spec": map[string]any{},
		"metadata": map[string]any{"name": "legacy", "annotations": []any{map[string]any{"acme.com/since": time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)}}},
	}
	fromObject, err := smarter.ManifestFromObject(object)
	check(t, err)
	equal(t, fromObject.Metadata.Annotations["acme.com/since"], "2026-10-28")
	if _, ok := object["metadata"].(map[string]any)["annotations"].([]any); !ok {
		t.Error("the caller's object was changed")
	}
	if _, err := smarter.ManifestFromYAML("kind: Guardrail\nmetadata:\n  name: x\n  annotations: [text]\n"); !errors.Is(err, smarter.ErrSmarter) {
		t.Errorf("got %v", err)
	}
	guardrail, err := smarter.ToSAM[sam.Guardrail](manifest)
	check(t, err)
	jsonEqual(t, guardrail.Metadata.Annotations, want)

	// the map form survives a round trip.
	data, err := json.Marshal(manifest)
	check(t, err)
	again, err := smarter.ParseManifest(data)
	check(t, err)
	jsonEqual(t, again.Metadata.Annotations, want)
}

func TestFixtureAnnotations(t *testing.T) {
	p := newMockPlatform(t)
	p.route("GET", "guardrails/", reply{fixture: "guardrails"})
	guardrails, err := p.client.Guardrails.List(context.Background())
	check(t, err)
	equal(t, guardrails[0].Annotations["smarter.sh/guardrail/builtin"], "true")

	p.route("POST", "cli/example-manifest/Guardrail/", reply{fixture: "example_guardrail"})
	example, err := p.client.Manifests.ExampleManifest(context.Background(), "Guardrail")
	check(t, err)
	equal(t, example.Metadata.Annotations["smarter.sh/guardrail/purpose"], "example")
}

func TestManifestsGetAnnotationSelector(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/get/LLMClient/", reply{fixture: "get_llmclient"})
	_, err := p.client.Manifests.Get(context.Background(), "LLMClient", "", &smarter.GetParams{
		Annotation: []string{" acme.com/crm-account-id=A-10042 ", "", "!acme.com/archived"},
	})
	check(t, err)
	jsonEqual(t, p.last().URL.Query()["annotation"], []string{"acme.com/crm-account-id=A-10042", "!acme.com/archived"})

	_, err = p.client.Manifests.Get(context.Background(), "LLMClient", "", &smarter.GetParams{})
	check(t, err)
	if _, ok := p.last().URL.Query()["annotation"]; ok {
		t.Error("an empty selector was sent")
	}

	p.route("POST", "cli/get/LLMClient/", reply{status: 400, json: map[string]any{"error": "invalid annotation selector"}})
	_, err = p.client.Manifests.Get(context.Background(), "LLMClient", "", &smarter.GetParams{Annotation: []string{"a=b=c"}})
	if !errors.Is(err, smarter.ErrBadRequest) {
		t.Errorf("got %v", err)
	}
}

func TestJournalPaginationSelector(t *testing.T) {
	var pagination smarter.JournalPagination
	check(t, json.Unmarshal([]byte(`{"page": 1, "pageSize": 25, "numPages": 1, "count": 0, "tags": ["a"], "annotations": ["acme.com/x=1"], "annotationError": "bad"}`), &pagination))
	jsonEqual(t, pagination.Tags, []string{"a"})
	jsonEqual(t, pagination.Annotations, []string{"acme.com/x=1"})
	equal(t, pagination.AnnotationError, "bad")
}
