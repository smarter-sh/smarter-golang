package smarter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// LLMHostsService holds the operations of LLMHosts: /api/v1/llmhosts/, and the manifest api's
// LLMHost kind. An LLMHost is an open-weight model that the platform serves on its own Kubernetes
// cluster.
type LLMHostsService struct {
	*KindService[LLMHost]
}

func llmhostDo[T any](ctx context.Context, s *LLMHostsService, method string, id ID, suffix string, q query) (*T, error) {
	path, err := s.path(id, suffix)
	if err != nil {
		return nil, err
	}
	var result T
	if err := s.c.http.do(ctx, &request{method: method, path: path, query: url.Values(q)}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Status returns an LLMHost's state, read from Kubernetes now: its status, replicas and pods.
func (s *LLMHostsService) Status(ctx context.Context, id ID) (*LLMHostStatus, error) {
	return llmhostDo[LLMHostStatus](ctx, s, http.MethodGet, id, "status/", nil)
}

// LogsParams are the optional arguments of [LLMHostsService.Logs].
type LogsParams struct {
	// Tail is how many lines, 200 by default.
	Tail int
}

// Logs returns the latest log lines of an LLMHost's pods.
func (s *LLMHostsService) Logs(ctx context.Context, id ID, params *LogsParams) (*LLMHostLogs, error) {
	tail := 200
	if params != nil && params.Tail > 0 {
		tail = params.Tail
	}
	return llmhostDo[LLMHostLogs](ctx, s, http.MethodGet, id, "logs/", query{}.num("tail", tail))
}

// Plan returns the Kubernetes resources that deploying an LLMHost would create, without creating them.
func (s *LLMHostsService) Plan(ctx context.Context, id ID) (*LLMHostPlan, error) {
	return llmhostDo[LLMHostPlan](ctx, s, http.MethodGet, id, "plan/", nil)
}

// Deploy deploys an LLMHost on the platform's cluster. This creates real, billable infrastructure,
// e.g. GPU nodes. Check [LLMHostsService.Plan] first.
func (s *LLMHostsService) Deploy(ctx context.Context, id ID) (*LLMHostStatus, error) {
	return llmhostDo[LLMHostStatus](ctx, s, http.MethodPost, id, "deploy/", nil)
}

// UndeployParams are the optional arguments of [LLMHostsService.Undeploy].
type UndeployParams struct {
	// Purge deletes the LLMHost's model storage too.
	Purge bool
}

// Undeploy undeploys an LLMHost, and returns its state.
func (s *LLMHostsService) Undeploy(ctx context.Context, id ID, params *UndeployParams) (*LLMHostStatus, error) {
	purge := params != nil && params.Purge
	return llmhostDo[LLMHostStatus](ctx, s, http.MethodPost, id, "undeploy/", query{}.flag("purge", purge))
}

// Report returns a report of every LLMHost that the user may read: counts by status and engine, and
// estimated costs.
func (s *LLMHostsService) Report(ctx context.Context) (*LLMHostReport, error) {
	var report LLMHostReport
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: s.Path + "report/"}, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

// DiscoverParams are the optional arguments of [LLMHostsService.Discover].
type DiscoverParams struct {
	// Query is the search text, e.g. qwen.
	Query string
	// Task selects only models of this task, e.g. chat or embedding.
	Task string
	// Catalog is the catalog, huggingface by default.
	Catalog string
	// Limit is the maximum number of models, 20 by default.
	Limit int
}

// Discover searches a public model catalog for models that an LLMHost can serve, and returns them,
// each with the LLMHost manifest that would serve it.
func (s *LLMHostsService) Discover(ctx context.Context, params *DiscoverParams) ([]DiscoveredModel, error) {
	if params == nil {
		params = &DiscoverParams{}
	}
	catalog, limit := params.Catalog, params.Limit
	if catalog == "" {
		catalog = "huggingface"
	}
	if limit <= 0 {
		limit = 20
	}
	q := query{}.str("q", params.Query).str("task", params.Task).str("catalog", catalog).num("limit", limit)
	var response struct {
		Models []DiscoveredModel `json:"models"`
	}
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: s.Path + "discover/", query: url.Values(q)}, &response); err != nil {
		return nil, err
	}
	return response.Models, nil
}

// DiscoverManifestParams are the optional arguments of [LLMHostsService.DiscoverManifest].
type DiscoverManifestParams struct {
	// Catalog is the catalog, huggingface by default.
	Catalog string
	// Engine is the inference engine, or "" for the platform's choice.
	Engine string
	// Name is the LLMHost's name, or "" for one derived from the repository.
	Name string
	// TokenSecret is the name of a Secret that holds the catalog's access token, for gated models.
	TokenSecret string
}

// DiscoverManifest returns an LLMHost manifest for a model in a public catalog, e.g.
// Qwen/Qwen2.5-7B-Instruct, ready to apply.
func (s *LLMHostsService) DiscoverManifest(ctx context.Context, repository string, params *DiscoverManifestParams) (*Manifest, error) {
	if params == nil {
		params = &DiscoverManifestParams{}
	}
	catalog := params.Catalog
	if catalog == "" {
		catalog = "huggingface"
	}
	q := query{}.str("repository", repository).str("catalog", catalog).str("engine", params.Engine).str("name", params.Name).str("tokenSecret", params.TokenSecret)
	var response struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: s.Path + "discover/manifest/", query: url.Values(q)}, &response); err != nil {
		return nil, err
	}
	return ParseManifest(response.Manifest)
}
