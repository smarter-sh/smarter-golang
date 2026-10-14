// Command samgen generates the SDK's sam package from the Smarter platform's committed JSON Schemas.
//
// The platform's SAM (Smarter Application Manifest) models are Pydantic models that depend on
// Django, so the SDK cannot import them. The platform commits each kind's JSON Schema, generated
// from those models, and an example manifest of each kind, in
// smarter/smarter/apps/api/v1/contract/sam/. scripts/sync-contracts.sh copies them into
// testdata/sam/. This command writes, into sam/:
//
//   - <kind>.go: the kind's struct (e.g. sam.Guardrail) and the structs of its parts,
//   - kinds.go: the Manifest interface, and Kinds, which maps each kind to its struct.
//
// It reads the subset of JSON Schema that Pydantic writes ($defs, $ref, properties, required, items,
// additionalProperties, enum, and anyOf with null), and fails on anything else, so that a new
// construct in the platform's models is noticed. See ADR-005.
//
// Usage, from the repository root (make sam-models runs it after make contracts):
//
//	go run ./internal/cmd/samgen
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// schema is a node of a JSON Schema, as Pydantic writes it.
type schema struct {
	Ref                  string             `json:"$ref"`
	Type                 json.RawMessage    `json:"type"`
	Title                string             `json:"title"`
	Description          string             `json:"description"`
	Properties           map[string]*schema `json:"properties"`
	Required             []string           `json:"required"`
	Items                *schema            `json:"items"`
	AdditionalProperties json.RawMessage    `json:"additionalProperties"`
	AnyOf                []*schema          `json:"anyOf"`
	Enum                 []any              `json:"enum"`
	Defs                 map[string]*schema `json:"$defs"`
	AllOf                []*schema          `json:"allOf"`
	OneOf                []*schema          `json:"oneOf"`
	Const                any                `json:"const"`
	Format               string             `json:"format"`
}

// typeName is the schema's single type, or "" when it has none.
func (s *schema) typeName() string {
	if len(s.Type) == 0 {
		return ""
	}
	var name string
	if err := json.Unmarshal(s.Type, &name); err != nil {
		fail("a list of types is not supported: %s", s.Type)
	}
	return name
}

// additional is the schema of an object's other properties, or nil when they have none.
func (s *schema) additional() *schema {
	if len(s.AdditionalProperties) == 0 || string(s.AdditionalProperties) == "true" || string(s.AdditionalProperties) == "false" {
		return nil
	}
	var additional schema
	if err := json.Unmarshal(s.AdditionalProperties, &additional); err != nil {
		fail("invalid additionalProperties: %v", err)
	}
	return &additional
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "samgen: "+format+"\n", args...)
	os.Exit(1)
}

// initialisms are written in one case in Go identifiers.
var initialisms = map[string]string{
	"api": "API", "aws": "AWS", "cpu": "CPU", "gpu": "GPU", "http": "HTTP", "https": "HTTPS", "id": "ID",
	"ids": "IDs", "ip": "IP", "json": "JSON", "llm": "LLM", "mcp": "MCP", "sql": "SQL", "ssh": "SSH",
	"ssl": "SSL", "tls": "TLS", "ttl": "TTL", "uri": "URI", "url": "URL", "urls": "URLs", "uuid": "UUID",
	"vram": "VRAM", "llmclient": "LLMClient", "llmclients": "LLMClients", "mcpclient": "MCPClient",
	"mcpclients": "MCPClients", "llmhost": "LLMHost", "llmhosts": "LLMHosts", "dns": "DNS", "db": "DB",
	"tos": "TOS", "ca": "CA",
}

var wordPattern = regexp.MustCompile(`[A-Z]+[a-z0-9]*|[a-z0-9]+`)

// goName is a Go identifier for a json name or a schema title, e.g. apiKeySecret -> APIKeySecret.
func goName(name string) string {
	var b strings.Builder
	for _, part := range regexp.MustCompile(`[^A-Za-z0-9]+`).Split(name, -1) {
		for _, word := range wordPattern.FindAllString(part, -1) {
			if upper, ok := initialisms[strings.ToLower(word)]; ok {
				b.WriteString(upper)
				continue
			}
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			b.WriteString(string(runes))
		}
	}
	result := b.String()
	if result == "" || unicode.IsDigit(rune(result[0])) {
		result = "X" + result
	}
	return result
}

// typeNameOf is the Go type name of a $defs entry: its title without the SAM prefix.
func typeNameOf(def string) string {
	return goName(strings.TrimPrefix(def, "SAM"))
}

// generator writes the Go declarations of one kind's schema.
type generator struct {
	kind     string
	root     *schema
	names    map[string]string // $defs name -> Go type name
	declared map[string]string // Go type name -> canonical JSON of the declared schema, for the whole package
	out      *bytes.Buffer
	pending  []func()
}

func (g *generator) comment(text, indent string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(g.out, "%s//\n", indent)
			continue
		}
		fmt.Fprintf(g.out, "%s// %s\n", indent, line)
	}
}

// resolve follows a $ref into $defs.
func (g *generator) resolve(ref string) (string, *schema) {
	name := strings.TrimPrefix(ref, "#/$defs/")
	def, ok := g.root.Defs[name]
	if !ok || !strings.HasPrefix(ref, "#/$defs/") {
		fail("%s: unresolved $ref %s", g.kind, ref)
	}
	return name, def
}

// goType is the Go type of a schema, and whether it is "optional by nature" (a pointer, slice or map).
// It declares the structs that the schema needs.
func (g *generator) goType(s *schema, nameHint string) string {
	if s.AllOf != nil || s.OneOf != nil || s.Const != nil {
		fail("%s: allOf, oneOf and const are not supported (%s)", g.kind, nameHint)
	}
	if s.Ref != "" {
		def, target := g.resolve(s.Ref)
		return g.declare(def, target)
	}
	if s.AnyOf != nil {
		var kinds []*schema
		for _, option := range s.AnyOf {
			if option.typeName() != "null" {
				kinds = append(kinds, option)
			}
		}
		if len(kinds) == 1 {
			return g.goType(kinds[0], nameHint)
		}
		return "any"
	}
	switch s.typeName() {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if s.Items == nil || (s.Items.typeName() == "" && s.Items.Ref == "" && s.Items.AnyOf == nil) {
			return "[]any"
		}
		return "[]" + g.goType(s.Items, nameHint+"Item")
	case "object":
		if len(s.Properties) > 0 {
			return g.declareStruct(nameHint, s)
		}
		if additional := s.additional(); additional != nil {
			return "map[string]" + g.goType(additional, nameHint+"Value")
		}
		return "map[string]any"
	case "":
		return "any"
	default:
		fail("%s: unsupported type %s (%s)", g.kind, s.typeName(), nameHint)
	}
	return ""
}

// declare declares a $defs entry, once per package, and returns its Go type name.
func (g *generator) declare(def string, s *schema) string {
	if name, ok := g.names[def]; ok {
		return name
	}
	name := typeNameOf(def)
	canonical, _ := json.Marshal(s)
	if existing, ok := g.declared[name]; ok && existing != string(canonical) {
		// another schema of this package already has the name, e.g. a kind's root struct.
		name += "Data"
	}
	g.names[def] = name
	if existing, ok := g.declared[name]; ok {
		if existing != string(canonical) {
			fail("%s: two different schemas are named %s", g.kind, name)
		}
		return name // declared by another kind's file
	}
	g.declared[name] = string(canonical)
	if len(s.Enum) > 0 {
		g.pending = append(g.pending, func() { g.writeEnum(name, s) })
		return name
	}
	if s.typeName() != "object" || len(s.Properties) == 0 {
		return g.goType(s, name)
	}
	g.pending = append(g.pending, func() { g.writeStruct(name, s) })
	return name
}

// declareStruct declares an inline object as a struct.
func (g *generator) declareStruct(name string, s *schema) string {
	canonical, _ := json.Marshal(s)
	if existing, ok := g.declared[name]; ok {
		if existing == string(canonical) {
			return name
		}
		name = goName(g.kind) + name
	}
	g.declared[name] = string(canonical)
	g.pending = append(g.pending, func() { g.writeStruct(name, s) })
	return name
}

func (g *generator) writeEnum(name string, s *schema) {
	g.comment(s.Description, "")
	fmt.Fprintf(g.out, "type %s = string\n\n", name)
}

// field is one field of a struct.
type field struct {
	name, json, goType, comment string
	omitempty                   bool
}

func (g *generator) writeStruct(name string, s *schema) {
	required := map[string]bool{}
	for _, key := range s.Required {
		required[key] = true
	}
	keys := make([]string, 0, len(s.Properties))
	for key := range s.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	used := map[string]bool{}
	var fields []field
	for _, key := range keys {
		property := s.Properties[key]
		fieldName := goName(key)
		for used[fieldName] {
			fieldName += "_"
		}
		used[fieldName] = true
		goType := g.goType(property, name+goName(key))
		nullable := false
		for _, option := range property.AnyOf {
			if option.typeName() == "null" {
				nullable = true
			}
		}
		optional := !required[key] || nullable
		isStruct := goType != "" && unicode.IsUpper(rune(goType[0])) && !g.isAlias(goType)
		switch {
		case optional && (goType == "int" || goType == "float64" || goType == "bool" || isStruct):
			goType = "*" + goType
		case required[key] && nullable && isStruct:
			goType = "*" + goType
		}
		fields = append(fields, field{
			name:      fieldName,
			json:      key,
			goType:    goType,
			comment:   property.Description,
			omitempty: optional,
		})
	}
	description := s.Description
	if name == goName(g.kind) {
		description = fmt.Sprintf("%s is a manifest of the %s kind.\n\n%s", name, g.kind, description)
	} else if description != "" {
		description = name + ": " + description
	}
	g.comment(description, "")
	fmt.Fprintf(g.out, "type %s struct {\n", name)
	for _, f := range fields {
		g.comment(f.comment, "\t")
		tag := f.json
		if f.omitempty {
			tag += ",omitempty"
		}
		fmt.Fprintf(g.out, "\t%s %s `json:%q`\n", f.name, f.goType, tag)
	}
	fmt.Fprint(g.out, "}\n\n")
}

// isAlias reports whether a declared name is an enum, which is an alias of string.
func (g *generator) isAlias(name string) bool {
	for def, declared := range g.names {
		if declared == name && len(g.root.Defs[def].Enum) > 0 {
			return true
		}
	}
	return false
}

func (g *generator) generate() []byte {
	rootName := goName(g.kind)
	canonical, _ := json.Marshal(g.root)
	g.declared[rootName] = string(canonical)
	g.pending = append(g.pending, func() { g.writeStruct(rootName, g.root) })
	for len(g.pending) > 0 {
		next := g.pending[0]
		g.pending = g.pending[1:]
		next()
	}
	fmt.Fprintf(g.out, "// SAMKind returns %q.\nfunc (*%s) SAMKind() string { return %q }\n", g.kind, rootName, g.kind)
	return g.out.Bytes()
}

func write(path string, source []byte) {
	formatted, err := format.Source(source)
	if err != nil {
		fail("%s does not compile: %v\n%s", path, err, source)
	}
	if err := os.WriteFile(path, formatted, 0o600); err != nil {
		fail("%v", err)
	}
	fmt.Println("generated", path)
}

func main() {
	data := filepath.Join("testdata", "sam")
	matches, err := filepath.Glob(filepath.Join(data, "*.schema.json"))
	if err != nil || len(matches) == 0 {
		fail("no schemas in %s: run scripts/sync-contracts.sh first", data)
	}
	sort.Strings(matches)
	old, _ := filepath.Glob(filepath.Join("sam", "*.go"))
	for _, path := range old {
		// only the files that samgen wrote: sam/doc.go and the tests are hand-written.
		content, err := os.ReadFile(path) //nolint:gosec // G304: a file of the sam package, from a glob
		if err == nil && bytes.HasPrefix(content, []byte("// Code generated by samgen")) {
			_ = os.Remove(path)
		}
	}

	declared := map[string]string{}
	var kinds []string
	for _, path := range matches {
		kinds = append(kinds, strings.TrimSuffix(filepath.Base(path), ".schema.json"))
	}
	// the kinds' own names first, so that a definition that shares one is renamed, not the kind.
	for _, kind := range kinds {
		declared[goName(kind)] = ""
	}
	for _, kind := range kinds {
		raw, err := os.ReadFile(filepath.Join(data, kind+".schema.json")) //nolint:gosec // G304: a schema of testdata/sam, from a glob
		if err != nil {
			fail("%v", err)
		}
		var root schema
		if err := json.Unmarshal(raw, &root); err != nil {
			fail("%s: %v", kind, err)
		}
		delete(declared, goName(kind))
		g := &generator{kind: kind, root: &root, names: map[string]string{}, declared: declared, out: &bytes.Buffer{}}
		header := fmt.Sprintf("// Code generated by samgen from the Smarter platform's JSON Schema of the %s kind. DO NOT EDIT.\n\npackage sam\n\n", kind)
		write(filepath.Join("sam", strings.ToLower(kind)+".go"), append([]byte(header), g.generate()...))
	}

	var b bytes.Buffer
	b.WriteString("// Code generated by samgen. DO NOT EDIT.\n\npackage sam\n\n")
	b.WriteString("// Kinds maps each manifest kind to a function that returns a new, empty struct of the kind.\nvar Kinds = map[string]func() Manifest{\n")
	for _, kind := range kinds {
		fmt.Fprintf(&b, "\t%q: func() Manifest { return &%s{} },\n", kind, goName(kind))
	}
	b.WriteString("}\n")
	write(filepath.Join("sam", "kinds.go"), b.Bytes())
}
