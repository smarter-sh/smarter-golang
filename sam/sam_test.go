package sam_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/smarter-sh/smarter-golang/sam"
)

const data = "../testdata/sam"

func kindsOfSchemas(t *testing.T) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(data, "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, path := range matches {
		kinds = append(kinds, strings.TrimSuffix(filepath.Base(path), ".schema.json"))
	}
	sort.Strings(kinds)
	return kinds
}

// TestKinds checks that there is a struct of every kind whose schema was saved, and only of those.
func TestKinds(t *testing.T) {
	kinds := kindsOfSchemas(t)
	if len(kinds) < 20 {
		t.Fatalf("only %d schemas in %s", len(kinds), data)
	}
	var registered []string
	for kind := range sam.Kinds {
		registered = append(registered, kind)
	}
	sort.Strings(registered)
	if strings.Join(registered, ",") != strings.Join(kinds, ",") {
		t.Fatalf("sam.Kinds has %v, the schemas are of %v", registered, kinds)
	}
}

// TestExamplesDecode checks that every kind's example manifest decodes into its struct, without a
// field that the struct doesn't declare, and keeps its kind and name.
func TestExamplesDecode(t *testing.T) {
	for _, kind := range kindsOfSchemas(t) {
		t.Run(kind, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(data, kind+".example.json"))
			if err != nil {
				t.Fatal(err)
			}
			manifest := sam.Kinds[kind]()
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(manifest); err != nil {
				t.Fatalf("the example of %s does not decode: %v", kind, err)
			}
			if manifest.SAMKind() != kind {
				t.Errorf("SAMKind() = %q, want %q", manifest.SAMKind(), kind)
			}
			encoded, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			var document struct {
				Kind     string `json:"kind"`
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			}
			if err := json.Unmarshal(encoded, &document); err != nil {
				t.Fatal(err)
			}
			var example struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			}
			_ = json.Unmarshal(raw, &example)
			if document.Kind != kind || document.Metadata.Name != example.Metadata.Name {
				t.Errorf("round trip gave kind %q name %q", document.Kind, document.Metadata.Name)
			}
		})
	}
}

// TestBuild checks that a manifest can be built in code, and encodes without its empty optional fields.
func TestBuild(t *testing.T) {
	guardrail := sam.Guardrail{
		APIVersion: "smarter.sh/v1",
		Kind:       "Guardrail",
		Metadata:   sam.GuardrailMetadata{Name: "pii_redactor"},
		Spec:       sam.GuardrailSpec{Config: sam.GuardrailSpecConfig{Action: "redact", Stage: "input", Strategy: "detector"}},
	}
	encoded, err := json.Marshal(guardrail)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "status") || strings.Contains(string(encoded), "caseSensitive") {
		t.Errorf("empty optional fields were encoded: %s", encoded)
	}
}
