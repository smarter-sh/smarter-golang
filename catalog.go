package smarter

import (
	"context"
	"net/http"
	"slices"
	"strings"
)

// PluginKinds are the manifest kinds of plugins.
var PluginKinds = []string{"Plugin", "ApiPlugin", "SkillPlugin", "SqlPlugin", "WebsearchPlugin", "ImageSearchPlugin"}

// SecretsService holds the operations of Secrets: /api/v1/secrets/, and the manifest api's Secret
// kind. The api never returns a secret's value. Create and rotate secrets with Apply.
type SecretsService struct {
	*KindService[Secret]
}

// OrchestratorsService holds the operations of Orchestrators: /api/v1/orchestrators/, and the
// manifest api's Orchestrator kind.
type OrchestratorsService struct {
	*KindService[Orchestrator]
}

// VectorsearchesService holds the operations of Vectorsearches: /api/v1/vectorsearches/, and the
// manifest api's Vectorsearch kind.
type VectorsearchesService struct {
	*KindService[Vectorsearch]
}

// Run runs a Vectorsearch: it searches the Vectorsearch's vectorstore for chunks similar to query,
// with the Vectorsearch's own configuration (search type, k, score threshold, ...), and returns them,
// best first. Each result must also match filter, unless it is empty, and the Vectorsearch's own
// metadata filter, which wins where the two name the same key. Running it embeds the query, which
// may be billable.
//
// The error matches [ErrBadRequest] if the Vectorsearch is disabled, the request is invalid or the
// search failed, [ErrNotFound] if the user may not read it, and [ErrPaymentRequired] if a Budget has
// locked the Vectorsearch, the user's profile or their account.
func (s *VectorsearchesService) Run(ctx context.Context, id ID, query string, filter JSONObject) (*VectorsearchResults, error) {
	path, err := s.path(id, "vectorsearch/")
	if err != nil {
		return nil, err
	}
	body := JSONObject{"query": query}
	if len(filter) > 0 {
		body["filter"] = filter
	}
	var results VectorsearchResults
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, body: body}, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// PluginsService holds the operations of Plugins: /api/v1/plugins/, and the manifest api's plugin
// kinds. A plugin's manifest kind is its class, e.g. ApiPlugin or SqlPlugin: Apply accepts any of
// them, and Describe and Delete look the kind up by name.
type PluginsService struct {
	*KindService[Plugin]
}

// Apply creates or updates a plugin from its manifest, of any plugin kind. It returns a
// [*ValueError] if the manifest is not a plugin's.
func (s *PluginsService) Apply(ctx context.Context, manifest *Manifest) (*JournalMessage, error) {
	if !slices.Contains(PluginKinds, manifest.Kind) {
		return nil, &ValueError{Message: "expected a plugin manifest (" + strings.Join(PluginKinds, ", ") + "), not " + manifest.Kind}
	}
	return s.c.Manifests.Apply(ctx, manifest)
}

// KindOf returns a plugin's manifest kind, e.g. SqlPlugin.
func (s *PluginsService) KindOf(ctx context.Context, name string) (string, error) {
	plugin, err := s.Find(ctx, name)
	if err != nil {
		return "", err
	}
	if plugin.Kind == "" {
		return s.Kind, nil
	}
	return plugin.Kind, nil
}

// Describe returns a plugin's manifest, with its read-only status.
func (s *PluginsService) Describe(ctx context.Context, name string) (*Manifest, error) {
	kind, err := s.KindOf(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.c.Manifests.Describe(ctx, kind, name)
}

// Delete permanently deletes a plugin.
func (s *PluginsService) Delete(ctx context.Context, name string) (*JournalMessage, error) {
	kind, err := s.KindOf(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.c.Manifests.Delete(ctx, kind, name)
}

// Retrieve returns one plugin, by its id, from the list. The platform's plugins/<id>/ returns a
// plugin's manifest, not the plugin: see [PluginsService.RetrieveManifest]. The error matches
// [ErrResourceNotFound] if the user may read no plugin of that id.
func (s *PluginsService) Retrieve(ctx context.Context, id ID) (*Plugin, error) {
	plugins, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range plugins {
		if IntID(plugins[i].ID) == id || ID(plugins[i].HashedID) == id {
			return &plugins[i], nil
		}
	}
	return nil, &resourceNotFound{kind: s.Kind, name: string(id)}
}

// RetrieveManifest returns one plugin's manifest, by the plugin's id: what the Python SDK's
// plugins.retrieve() returns. The platform serves this only when its ALLOW_API_GET waffle switch is
// on. Otherwise it answers 405: use Find or Describe instead.
func (s *PluginsService) RetrieveManifest(ctx context.Context, id ID) (*Manifest, error) {
	path, err := s.path(id, "")
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path}, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
