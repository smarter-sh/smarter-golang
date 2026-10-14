package smarter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/smarter-sh/smarter-golang/sam"
)

// APIVersionV1 is the apiVersion of every Smarter manifest.
const APIVersionV1 = "smarter.sh/v1"

// ManifestMetadata is a manifest's metadata: the resource's name, and descriptive information about it.
// Fields that the SDK doesn't declare are kept in Extra.
type ManifestMetadata struct {
	// Name is the resource's name, unique per account and kind, in snake_case.
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Version is the resource's semantic version, e.g. 1.0.0.
	Version string   `json:"version,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	// Annotations are key-value pairs, as a list of single-key objects.
	Annotations []JSONObject `json:"annotations,omitempty"`
	// Extra holds the fields that the SDK doesn't declare.
	Extra map[string]json.RawMessage `json:"-"`
}

// Manifest is a Smarter Application Manifest: the YAML document that declares a resource.
//
// It has the same four sections as a Kubernetes manifest: APIVersion, Kind, Metadata and Spec.
// Describe adds a read-only Status. Spec is a JSON object, because its fields depend on the kind:
// use [ToSAM] for the kind's typed struct. Top-level fields that the SDK doesn't declare are kept in
// Extra, so describe, change and apply loses nothing.
type Manifest struct {
	APIVersion string `json:"apiVersion"`
	// Kind is the resource kind, e.g. LLMClient. See [Client.Kinds].
	Kind     string           `json:"kind"`
	Metadata ManifestMetadata `json:"metadata"`
	// Spec is the resource's specification.
	Spec JSONObject `json:"spec"`
	// Status is read-only state, returned by describe.
	Status JSONObject `json:"status,omitempty"`
	// Extra holds the top-level fields that the SDK doesn't declare.
	Extra map[string]json.RawMessage `json:"-"`
}

// splitExtra decodes data into known, and returns the fields whose names aren't in names.
func splitExtra(data []byte, known any, names ...string) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(data, known); err != nil {
		return nil, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	for _, name := range names {
		delete(all, name)
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all, nil
}

// joinExtra encodes known, with the extra fields added.
func joinExtra(known any, extra map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(known)
	if err != nil || len(extra) == 0 {
		return data, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	for name, value := range extra {
		if _, ok := all[name]; !ok {
			all[name] = value
		}
	}
	return json.Marshal(all)
}

// UnmarshalJSON decodes the metadata, and keeps the fields that the SDK doesn't declare.
func (m *ManifestMetadata) UnmarshalJSON(data []byte) error {
	type plain ManifestMetadata
	extra, err := splitExtra(data, (*plain)(m), "name", "description", "version", "tags", "annotations")
	m.Extra = extra
	return err
}

// MarshalJSON encodes the metadata, with the fields that the SDK doesn't declare.
func (m ManifestMetadata) MarshalJSON() ([]byte, error) {
	type plain ManifestMetadata
	return joinExtra(plain(m), m.Extra)
}

// UnmarshalJSON decodes the manifest, and keeps the fields that the SDK doesn't declare.
func (m *Manifest) UnmarshalJSON(data []byte) error {
	type plain Manifest
	extra, err := splitExtra(data, (*plain)(m), "apiVersion", "kind", "metadata", "spec", "status")
	m.Extra = extra
	return err
}

// MarshalJSON encodes the manifest, with the fields that the SDK doesn't declare.
func (m Manifest) MarshalJSON() ([]byte, error) {
	type plain Manifest
	if m.Spec == nil {
		m.Spec = JSONObject{}
	}
	return joinExtra(plain(m), m.Extra)
}

// ParseManifest decodes a manifest from JSON, and checks that it has a kind and a name.
func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, &ValueError{Message: "not a Smarter manifest: " + err.Error(), Err: err}
	}
	if manifest.Kind == "" || manifest.Metadata.Name == "" {
		return nil, &ValueError{Message: "not a Smarter manifest: it needs a kind, and a metadata.name"}
	}
	if manifest.APIVersion == "" {
		manifest.APIVersion = APIVersionV1
	}
	return &manifest, nil
}

// ManifestFromObject converts a manifest's decoded JSON (or YAML) object into a [Manifest].
func ManifestFromObject(object JSONObject) (*Manifest, error) {
	data, err := json.Marshal(object)
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be encoded: " + err.Error(), Err: err}
	}
	return ParseManifest(data)
}

// ManifestFromYAML parses a manifest from YAML (or JSON) text.
func ManifestFromYAML(text string) (*Manifest, error) {
	var document any
	if err := yaml.Unmarshal([]byte(text), &document); err != nil {
		return nil, &ValueError{Message: "the manifest is not valid YAML: " + err.Error(), Err: err}
	}
	if _, ok := document.(map[string]any); !ok {
		return nil, &ValueError{Message: "a manifest must be a YAML object with apiVersion, kind, metadata and spec"}
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be converted to JSON: " + err.Error(), Err: err}
	}
	return ParseManifest(data)
}

// ManifestFromFile reads a manifest from a YAML or JSON file.
func ManifestFromFile(path string) (*Manifest, error) {
	text, err := os.ReadFile(path) //nolint:gosec // G304: the caller names the file to read
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be read: " + err.Error(), Err: err}
	}
	return ManifestFromYAML(string(text))
}

// ManifestFromSAM converts a typed SAM struct, e.g. a *sam.Guardrail, into a [Manifest].
func ManifestFromSAM(manifest sam.Manifest) (*Manifest, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be encoded: " + err.Error(), Err: err}
	}
	return ParseManifest(data)
}

// ToSAM decodes a manifest into its kind's typed struct, e.g. smarter.ToSAM[sam.Guardrail](m). Fields
// that the struct doesn't declare are ignored (ADR-004). It returns a [*ValueError] if the manifest is
// of another kind, or doesn't match the struct.
func ToSAM[T any, PT interface {
	*T
	sam.Manifest
}](m *Manifest) (*T, error) {
	result := PT(new(T))
	if kind := result.SAMKind(); kind != m.Kind {
		return nil, &ValueError{Message: fmt.Sprintf("expected a %s manifest, not %s", kind, m.Kind)}
	}
	if err := m.decode(result); err != nil {
		return nil, err
	}
	return (*T)(result), nil
}

// SAM decodes the manifest into its kind's typed struct, e.g. a *sam.Guardrail, which callers get
// with a type switch or assertion. It returns a [*ValueError] if the SDK has no struct of the kind,
// or the manifest doesn't match it. See also [ToSAM].
func (m *Manifest) SAM() (sam.Manifest, error) {
	newManifest, ok := sam.Kinds[m.Kind]
	if !ok {
		return nil, &ValueError{Message: fmt.Sprintf("the SDK has no SAM struct of the %s kind", m.Kind)}
	}
	result := newManifest()
	if err := m.decode(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *Manifest) decode(target sam.Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return &ValueError{Message: "the manifest can't be encoded: " + err.Error(), Err: err}
	}
	if err := json.Unmarshal(data, target); err != nil {
		return &ValueError{Message: fmt.Sprintf("the manifest does not match the %s struct: %v", m.Kind, err), Err: err}
	}
	return nil
}

// ToApply returns the manifest as the api's apply and validate commands take it: without its
// read-only Status.
func (m *Manifest) ToApply() ([]byte, error) {
	applied := *m
	applied.Status = nil
	if applied.APIVersion == "" {
		applied.APIVersion = APIVersionV1
	}
	return json.Marshal(applied)
}

// ToYAML returns the manifest as YAML text, without its read-only Status, ready to apply, with its
// sections in the order of the manifest.
func (m *Manifest) ToYAML() (string, error) {
	data, err := m.ToApply()
	if err != nil {
		return "", err
	}
	// JSON is YAML: decoding it into a node keeps the key order, which a map would lose.
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return "", err
	}
	blockStyle(&node)
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// blockStyle turns a node decoded from JSON into YAML's block style.
func blockStyle(node *yaml.Node) {
	if node.Kind == yaml.ScalarNode {
		if node.Tag == "!!str" && !strings.ContainsAny(node.Value, "\n") {
			node.Style = 0
		} else if node.Tag != "!!str" {
			node.Style = 0
		}
	} else {
		node.Style = 0
	}
	for _, child := range node.Content {
		blockStyle(child)
	}
}

// ManifestListTitle is a column of a get result: the field's name, and its Django field type.
type ManifestListTitle struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ManifestListData is the table of a get result.
type ManifestListData struct {
	Titles []ManifestListTitle `json:"titles"`
	Items  []JSONObject        `json:"items"`
}

// ManifestListMetadata is the metadata of a get result.
type ManifestListMetadata struct {
	Count int `json:"count"`
}

// ManifestList is the result of get: the resources of one kind, as a table. Data.Items are the rows,
// and Data.Titles their columns. The columns depend on the kind.
type ManifestList struct {
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Name       string               `json:"name,omitempty"`
	Metadata   ManifestListMetadata `json:"metadata"`
	Data       ManifestListData     `json:"data"`
}

// Names returns the names of the resources: each row's name, or its username for the User kind.
func (l *ManifestList) Names() []string {
	var names []string
	for _, item := range l.Data.Items {
		name, ok := item["name"]
		if !ok || name == nil {
			name = item["username"]
		}
		switch value := name.(type) {
		case string:
			names = append(names, value)
		case float64:
			names = append(names, fmt.Sprint(value))
		}
	}
	return names
}

// ManifestValidation is the result of validating a manifest, without applying it.
type ManifestValidation struct {
	Valid  bool        `json:"valid"`
	Errors []JSONValue `json:"errors,omitempty"`
}
