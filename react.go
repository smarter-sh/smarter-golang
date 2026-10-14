package smarter

import (
	"encoding/json"
	"sort"
)

// The Smarter platform's Django-React integration: a React app's Vite manifest, and the assets that
// a template includes. These are a copy of the platform's
// smarter.lib.django.templatetags.smarter_react_contract, as the Python SDK's smarter.models.react
// is, so that tools that build or check React apps for a Smarter platform can use them without the
// platform.

// ReactBuildInfoKey is the key of the build information that each React package's vite.config.ts
// adds to its manifest.json.
const ReactBuildInfoKey = "_custom"

// ViteManifestChunk is one entry of a Vite-generated manifest.json: a chunk, or a static asset. See
// https://vite.dev/guide/backend-integration.
type ViteManifestChunk struct {
	// File is the output file of this chunk, relative to the app's static directory.
	File           string   `json:"file"`
	Name           string   `json:"name,omitempty"`
	Names          []string `json:"names,omitempty"`
	Src            string   `json:"src,omitempty"`
	IsEntry        bool     `json:"isEntry,omitempty"`
	IsDynamicEntry bool     `json:"isDynamicEntry,omitempty"`
	// Imports are the manifest keys of statically imported chunks.
	Imports []string `json:"imports,omitempty"`
	// DynamicImports are the manifest keys of dynamically imported chunks.
	DynamicImports []string `json:"dynamicImports,omitempty"`
	// CSS are the CSS files that this chunk loads.
	CSS []string `json:"css,omitempty"`
	// Assets are the other static files that this chunk references.
	Assets []string `json:"assets,omitempty"`
}

// ReactAppBuildInfo is the build information that each React package's vite.config.ts adds to its
// manifest.json, as _custom.
type ReactAppBuildInfo struct {
	// BuildTime is when the app was built.
	BuildTime string `json:"buildTime"`
	// Version is the version of the app's npm package.
	Version string `json:"version"`
	// BuildEnv is the build's NODE_ENV, e.g. production.
	BuildEnv string `json:"buildEnv"`
}

// ReactAppAssets are the ordered JS and CSS files of a React app's entry point, which a Django
// template includes.
type ReactAppAssets struct {
	// JS are the JS files, the entry point's first.
	JS []string `json:"js"`
	// CSS are the CSS files, the entry point's first.
	CSS []string `json:"css"`
}

// ViteManifest is a React app's Vite-generated manifest.json: its chunks, keyed by source path, and
// Smarter's build information. It decodes from and encodes to the file's shape.
type ViteManifest struct {
	// Chunks are the manifest's chunks, keyed by source path.
	Chunks map[string]ViteManifestChunk
	// BuildInfo is the _custom build information, if the package's vite.config.ts adds it.
	BuildInfo *ReactAppBuildInfo
}

// UnmarshalJSON splits the manifest.json's _custom entry from its chunks.
func (m *ViteManifest) UnmarshalJSON(data []byte) error {
	var entries map[string]json.RawMessage
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	m.Chunks, m.BuildInfo = map[string]ViteManifestChunk{}, nil
	for key, raw := range entries {
		if key == ReactBuildInfoKey {
			m.BuildInfo = &ReactAppBuildInfo{}
			if err := json.Unmarshal(raw, m.BuildInfo); err != nil {
				return err
			}
			continue
		}
		var chunk ViteManifestChunk
		if err := json.Unmarshal(raw, &chunk); err != nil {
			return err
		}
		m.Chunks[key] = chunk
	}
	return nil
}

// MarshalJSON encodes the manifest in the shape of manifest.json, with the build information as its
// _custom entry.
func (m ViteManifest) MarshalJSON() ([]byte, error) {
	entries := map[string]any{}
	for key, chunk := range m.Chunks {
		entries[key] = chunk
	}
	if m.BuildInfo != nil {
		entries[ReactBuildInfoKey] = m.BuildInfo
	}
	return json.Marshal(entries)
}

// EntryKey returns the key of the manifest's entry point, the first chunk (in key order) with
// isEntry, or "" if the manifest has none.
func (m ViteManifest) EntryKey() string {
	keys := make([]string, 0, len(m.Chunks))
	for key := range m.Chunks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if m.Chunks[key].IsEntry {
			return key
		}
	}
	return ""
}

// CollectAssets returns the JS ("js") or CSS ("css") files of a chunk and of its static imports, the
// chunk's own first, then each import's, depth first, in the order they are listed. Each chunk is
// visited once, so circular imports terminate.
func (m ViteManifest) CollectAssets(key, assetType string) []string {
	return m.collect(key, assetType, map[string]bool{})
}

func (m ViteManifest) collect(key, assetType string, seen map[string]bool) []string {
	chunk, ok := m.Chunks[key]
	if seen[key] || !ok {
		return nil
	}
	seen[key] = true
	var assets []string
	if assetType == "js" {
		assets = append(assets, chunk.File)
	} else {
		assets = append(assets, chunk.CSS...)
	}
	for _, imported := range chunk.Imports {
		assets = append(assets, m.collect(imported, assetType, seen)...)
	}
	return assets
}
