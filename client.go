package smarter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// Client is a client of a Smarter platform's api. Its fields group the api's operations, and it is
// safe for concurrent use by many goroutines. Create one with [NewClient].
type Client struct {
	// Manifests is the manifest api: apply, validate, get, describe, delete, deploy, undeploy and
	// logs, for any kind.
	Manifests *ManifestsService
	// LLMClients are LLMClients, and prompting and chatting with them.
	LLMClients *LLMClientsService
	// Guardrails are Guardrails.
	Guardrails *GuardrailsService
	// MCPClients are MCPClients.
	MCPClients *MCPClientsService
	// Vectorstores are Vectorstores.
	Vectorstores *VectorstoresService
	// LLMHosts are LLMHosts.
	LLMHosts *LLMHostsService
	// Secrets are Secrets.
	Secrets *SecretsService
	// Plugins are Plugins, of every plugin kind.
	Plugins *PluginsService
	// Orchestrators are Orchestrators.
	Orchestrators *OrchestratorsService
	// Vectorsearches are Vectorsearches.
	Vectorsearches *VectorsearchesService
	// Accounts are Accounts, and their users, contacts and user profiles.
	Accounts *AccountsService
	// History is the history of chat sessions, tool calls and plugin uses.
	History *PromptHistoryService
	// Journal is the Smarter Journal. Superusers only.
	Journal *JournalService
	// Providers are the LLM providers and their models.
	Providers *ProvidersService
	// Proxies are proxies to LLM providers' own apis.
	Proxies *ProxiesService

	http   *httpClient
	logger *slog.Logger
}

// NewClient returns a client. Each value is taken from the first of: the options, the SMARTER_*
// environment variables (SMARTER_API_KEY, SMARTER_BASE_URL, SMARTER_ENVIRONMENT,
// SMARTER_ROOT_DOMAIN, SMARTER_TIMEOUT in seconds, SMARTER_MAX_RETRIES, SMARTER_LOG and
// SMARTER_CONFIG_FILE), the smarter cli's configuration file (~/.smarter/config.yaml), and the
// defaults: the prod environment of platform.smarter.sh. Without a base url, the url is derived from
// the environment as the cli derives it: see [EnvironmentURL].
//
// It returns a [*ConfigurationError] if no api key is found, or a value is invalid.
func NewClient(opts ...Option) (*Client, error) {
	o := newOptions(opts)
	s, err := readSettings(o.lookupEnv)
	if err != nil {
		return nil, err
	}
	logger := newLogger(o.logger, s.log)
	cfg, err := resolveConfig(o, s, logger)
	if err != nil {
		return nil, err
	}
	sender := o.httpClient
	if sender == nil {
		sender = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	c := &Client{
		http:   &httpClient{cfg: cfg, client: sender, headers: o.headers.Clone(), logger: logger},
		logger: logger,
	}
	c.Manifests = &ManifestsService{c: c}
	c.LLMClients = &LLMClientsService{newKindService[LLMClient](c, "LLMClient", "llm-clients/")}
	c.Guardrails = &GuardrailsService{newKindService[Guardrail](c, "Guardrail", "guardrails/")}
	c.MCPClients = &MCPClientsService{newKindService[MCPClient](c, "MCPClient", "mcpclients/")}
	c.Vectorstores = &VectorstoresService{newKindService[Vectorstore](c, "Vectorstore", "vectorstores/")}
	c.LLMHosts = &LLMHostsService{newKindService[LLMHost](c, "LLMHost", "llmhosts/")}
	c.Secrets = &SecretsService{newKindService[Secret](c, "Secret", "secrets/")}
	c.Plugins = &PluginsService{newKindService[Plugin](c, "Plugin", "plugins/")}
	c.Orchestrators = &OrchestratorsService{newKindService[Orchestrator](c, "Orchestrator", "orchestrators/")}
	c.Vectorsearches = &VectorsearchesService{newKindService[Vectorsearch](c, "Vectorsearch", "vectorsearches/")}
	c.Accounts = &AccountsService{
		KindService:  newKindService[Account](c, "Account", "accounts/"),
		Users:        &ReadService[User]{c: c, path: "accounts/users/"},
		Contacts:     &ReadService[AccountContact]{c: c, path: "accounts/contacts/"},
		UserProfiles: &ReadService[UserProfile]{c: c, path: "accounts/user-profiles/"},
	}
	c.History = &PromptHistoryService{c: c}
	c.Journal = &JournalService{c: c}
	c.Providers = &ProvidersService{c: c}
	c.Proxies = &ProxiesService{c: c}
	return c, nil
}

// APIKey returns the api key, e.g. to use a proxy with a provider's own SDK.
func (c *Client) APIKey() string { return c.http.cfg.apiKey }

// BaseURL returns the api's base url, e.g. https://platform.smarter.sh/api/v1/.
func (c *Client) BaseURL() string { return c.http.cfg.baseURL }

// Environment returns the environment that the configuration was resolved for.
func (c *Client) Environment() Environment { return c.http.cfg.environment }

// String returns the client's base url and masked api key.
func (c *Client) String() string {
	return fmt.Sprintf("smarter.Client(baseURL=%q, apiKey=%q)", c.BaseURL(), MaskSecret(c.APIKey()))
}

// GoString is String, so that %#v doesn't print the api key.
func (c *Client) GoString() string { return c.String() }

// LogValue is how slog logs the client: without its api key.
func (c *Client) LogValue() slog.Value {
	return slog.GroupValue(slog.String("baseURL", c.BaseURL()), slog.String("apiKey", MaskSecret(c.APIKey())))
}

// Do sends a request to an api endpoint that the SDK doesn't wrap, with the client's authentication,
// retries and error handling, and decodes its JSON response into out, unless out is nil. The path is
// relative to the api's base url, e.g. "cli/whoami/", and body, unless nil, is sent as JSON.
func (c *Client) Do(ctx context.Context, method, path string, body JSONValue, out any) error {
	return c.http.do(ctx, &request{method: method, path: path, body: body}, out)
}

// cliData sends a cli api request, and decodes its journaled response's data.
func cliData[T any](ctx context.Context, c *Client, path string) (*T, error) {
	var response JournaledResponse[T]
	if err := c.http.do(ctx, &request{method: http.MethodPost, path: "cli/" + path}, &response); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, &APIResponseValidationError{Method: http.MethodPost, URL: c.http.url("cli/"+path, nil), StatusCode: http.StatusOK, Err: fmt.Errorf("the response has no data")}
	}
	return response.Data, nil
}

// WhoAmI returns the owner of the api key: the user, their account, and the platform's environment.
func (c *Client) WhoAmI(ctx context.Context) (*WhoAmI, error) {
	return cliData[WhoAmI](ctx, c, "whoami/")
}

// Version returns the platform's version, and the versions of its main Python packages.
func (c *Client) Version(ctx context.Context) (*PlatformVersion, error) {
	return cliData[PlatformVersion](ctx, c, "version/")
}

// Status returns the state of the platform's infrastructure and compute.
func (c *Client) Status(ctx context.Context) (*PlatformStatus, error) {
	return cliData[PlatformStatus](ctx, c, "status/")
}

// Kinds returns the resource kinds that the platform supports, and which of them can be deployed.
func (c *Client) Kinds(ctx context.Context) ([]KindInfo, error) {
	kinds, err := cliData[[]KindInfo](ctx, c, "resources/")
	if err != nil {
		return nil, err
	}
	return *kinds, nil
}
