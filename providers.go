package smarter

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// ProvidersService reads the LLM providers that the platform is configured for, e.g. openai or
// anthropic, and their models: /api/v1/providers/.
type ProvidersService struct {
	c *Client
}

// List returns the providers.
func (s *ProvidersService) List(ctx context.Context) ([]Provider, error) {
	var providers []Provider
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "providers/"}, &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

// Retrieve returns a provider, by name, e.g. openai.
func (s *ProvidersService) Retrieve(ctx context.Context, name string) (*Provider, error) {
	segment, err := ID(name).segment()
	if err != nil {
		return nil, err
	}
	var provider Provider
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "providers/" + segment + "/"}, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

// Models returns a provider's models.
func (s *ProvidersService) Models(ctx context.Context, name string) ([]ProviderModel, error) {
	segment, err := ID(name).segment()
	if err != nil {
		return nil, err
	}
	var models []ProviderModel
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "providers/" + segment + "/models/"}, &models); err != nil {
		return nil, err
	}
	return models, nil
}

// ProxiesService gives authenticated passthrough access to LLM providers' own apis: /api/v1/proxy/.
// The platform holds the provider's api key, and callers use their Smarter api key instead. The
// platform serves these endpoints only when SMARTER_ENABLE_PROXY is true. The easiest way to call a
// proxy is with the provider's own SDK, with BaseURL and the client's APIKey.
type ProxiesService struct {
	c *Client
}

// List returns the proxies that the user may call.
func (s *ProxiesService) List(ctx context.Context) ([]Proxy, error) {
	var response struct {
		Proxies []Proxy `json:"proxies"`
	}
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "proxy/"}, &response); err != nil {
		return nil, err
	}
	return response.Proxies, nil
}

// BaseURL returns a proxy's base url, for the provider's own SDK, e.g.
// https://platform.smarter.sh/api/v1/proxy/openai/.
func (s *ProxiesService) BaseURL(name string) string {
	return s.c.http.url("proxy/"+strings.Trim(name, "/")+"/", nil)
}

// ProxyRequestParams are the optional arguments of [ProxiesService.Request].
type ProxyRequestParams struct {
	// Method is the http method, POST by default.
	Method string
	// JSON is the JSON body, as the provider's api expects it, or nil for none.
	JSON JSONValue
	// Query holds query parameters.
	Query url.Values
}

// Request sends a request to a provider's api, through its proxy, and returns the provider's JSON
// response, or nil for an empty one. The path, e.g. v1/chat/completions, must be one of the proxy's
// AllowedPaths.
func (s *ProxiesService) Request(ctx context.Context, name, path string, params *ProxyRequestParams) (JSONValue, error) {
	segment, err := ID(name).segment()
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = &ProxyRequestParams{}
	}
	method := strings.ToUpper(params.Method)
	if method == "" {
		method = http.MethodPost
	}
	response, done, err := s.c.http.send(ctx, &request{method: method, path: "proxy/" + segment + "/" + strings.TrimLeft(path, "/"), query: params.Query, body: params.JSON}, false)
	if err != nil {
		return nil, err
	}
	defer done()
	defer func() { _ = response.Body.Close() }()
	var result JSONValue
	decoder := newJSONDecoder(response.Body)
	if err := decoder.Decode(&result); err != nil && !isEOF(err) {
		return nil, &APIResponseValidationError{Method: method, URL: s.BaseURL(name) + path, StatusCode: response.StatusCode, Err: err}
	}
	return result, nil
}
