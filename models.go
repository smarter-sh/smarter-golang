package smarter

import "encoding/json"

// The structs of the REST api's responses. They are the Python SDK's models (smarter/models/), with
// Go field names and the api's own JSON names (ADR-021). Unknown fields are ignored when a response is
// decoded, optional numbers and booleans are pointers, and timestamps are ISO 8601 strings (ADR-004).

// UserRef is the user of a resource's owner.
type UserRef struct {
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
}

// AccountRef is the account of a resource's owner.
type AccountRef struct {
	AccountNumber string `json:"accountNumber"`
}

// Owner is the owner of a resource: its user profile, as the user and the account.
type Owner struct {
	User    *UserRef    `json:"user,omitempty"`
	Account *AccountRef `json:"account,omitempty"`
}

// Resource holds the fields that every Smarter resource shares, as the REST api returns them: the
// same metadata as a manifest's metadata. Each resource's struct embeds it.
type Resource struct {
	ID          int      `json:"id"`
	HashedID    string   `json:"hashedId,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	// Annotations are Kubernetes-style annotations: string keys mapped to string values. See [Annotations].
	Annotations Annotations `json:"annotations,omitempty"`
	CreatedAt   string      `json:"createdAt,omitempty"`
	UpdatedAt   string      `json:"updatedAt,omitempty"`
	UserProfile *Owner      `json:"userProfile,omitempty"`
	CanDelete   *bool       `json:"canDelete,omitempty"`
	ManifestURL string      `json:"manifestUrl,omitempty"`
	// DetailURL is the url of the resource's detail in the web console: its manifest, the resources
	// that it uses, and those that use it.
	DetailURL string `json:"detailUrl,omitempty"`
	Ready     *bool  `json:"ready,omitempty"`
}

// ResourceName returns the resource's name.
func (r Resource) ResourceName() string { return r.Name }

// Ref returns the resource's ID, for the methods that take one: its hashed id when it has one.
func (r Resource) Ref() ID {
	if r.HashedID != "" {
		return ID(r.HashedID)
	}
	return IntID(r.ID)
}

// Account is a Smarter account: the tenant that owns users and resources.
type Account struct {
	Resource
	AccountNumber    string `json:"accountNumber"`
	IsDefaultAccount *bool  `json:"isDefaultAccount,omitempty"`
	CompanyName      string `json:"companyName,omitempty"`
	PhoneNumber      string `json:"phoneNumber,omitempty"`
	Address1         string `json:"address1,omitempty"`
	Address2         string `json:"address2,omitempty"`
	City             string `json:"city,omitempty"`
	State            string `json:"state,omitempty"`
	PostalCode       string `json:"postalCode,omitempty"`
	Country          string `json:"country,omitempty"`
	Language         string `json:"language,omitempty"`
	Timezone         string `json:"timezone,omitempty"`
	Currency         string `json:"currency,omitempty"`
	IsActive         *bool  `json:"isActive,omitempty"`
}

// User is a user of the platform.
type User struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Email       string `json:"email,omitempty"`
	IsStaff     *bool  `json:"isStaff,omitempty"`
	IsSuperuser *bool  `json:"isSuperuser,omitempty"`
}

// UserProfile is a user's membership of an account, which owns the resources that the user creates in it.
type UserProfile struct {
	Resource
	User    User       `json:"user"`
	Account AccountRef `json:"account"`
	// ProfileImage is the url of the image that the user uploaded, which takes precedence over
	// ProfileImageURL.
	ProfileImage string `json:"profileImage,omitempty"`
	// ProfileImageURL is the image that the oauth provider supplied.
	ProfileImageURL string `json:"profileImageUrl,omitempty"`
	IsTest          *bool  `json:"isTest,omitempty"`
}

// AccountContact is a contact of an account, who receives its notices.
type AccountContact struct {
	ID        int         `json:"id"`
	Account   *AccountRef `json:"account,omitempty"`
	CreatedAt string      `json:"createdAt,omitempty"`
	UpdatedAt string      `json:"updatedAt,omitempty"`
	FirstName string      `json:"firstName,omitempty"`
	LastName  string      `json:"lastName,omitempty"`
	Email     string      `json:"email,omitempty"`
	Phone     string      `json:"phone,omitempty"`
	IsPrimary *bool       `json:"isPrimary,omitempty"`
	IsTest    *bool       `json:"isTest,omitempty"`
	Welcomed  *bool       `json:"welcomed,omitempty"`
}

// WhoAmI is the owner of the api key: the user, their account, and the platform's environment.
type WhoAmI struct {
	User    User    `json:"user"`
	Account Account `json:"account"`
	// Environment is the platform's environment, e.g. local or prod.
	Environment string `json:"environment,omitempty"`
}

// APIVersion is the platform's version, and the versions of its main Python packages.
type APIVersion struct {
	// Version is the Smarter installation's semantic version.
	Version string `json:"version"`
	// Python holds Python package versions, by package; nil when a package isn't installed.
	Python map[string]*string `json:"python,omitempty"`
}

// CLIVersion is the version of the smarter cli that the platform recommends.
type CLIVersion struct {
	Version string `json:"version,omitempty"`
}

// PlatformVersion is the response of Version.
type PlatformVersion struct {
	API APIVersion  `json:"api"`
	CLI *CLIVersion `json:"cli,omitempty"`
}

// PlatformStatus is the response of Status: the state of the platform's infrastructure and compute.
// The sections' contents depend on how the platform is hosted, so they are JSON objects.
type PlatformStatus struct {
	Infrastructures JSONObject `json:"infrastructures,omitempty"`
	Compute         JSONObject `json:"compute,omitempty"`
}

// KindInfo is a resource kind that the platform supports, as /api/v1/cli/resources/ describes it.
type KindInfo struct {
	Singular string `json:"Singular"`
	Plural   string `json:"Plural"`
	// APIKind is the kind, as it appears in a manifest.
	APIKind       string `json:"APIKind"`
	Display       string `json:"Display,omitempty"`
	DisplayPlural string `json:"DisplayPlural,omitempty"`
	// Deployable is true if the kind can be deployed.
	Deployable bool `json:"Deployable"`
}

// Guardrail is a check that the platform runs on prompts (input) or on the LLM's replies (output).
type Guardrail struct {
	Resource
	// Stage is input or output.
	Stage    string `json:"stage,omitempty"`
	Category string `json:"category,omitempty"`
	// Strategy is e.g. regex, detector, classifier or llm.
	Strategy  string     `json:"strategy,omitempty"`
	Pattern   string     `json:"pattern,omitempty"`
	Config    JSONObject `json:"config,omitempty"`
	Threshold *float64   `json:"threshold,omitempty"`
	// Action is e.g. block, redact or flag.
	Action      string `json:"action,omitempty"`
	Replacement string `json:"replacement,omitempty"`
	Message     string `json:"message,omitempty"`
	Severity    *int   `json:"severity,omitempty"`
	// Mode is enforce or monitor.
	Mode       string `json:"mode,omitempty"`
	FailClosed *bool  `json:"failClosed,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
	IsActive   *bool  `json:"isActive,omitempty"`
}

// GuardrailMatch is a span of the text that a guardrail matched.
type GuardrailMatch struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	Label string `json:"label,omitempty"`
}

// GuardrailFinding is one finding of a guardrail evaluation.
type GuardrailFinding struct {
	Confidence *float64         `json:"confidence,omitempty"`
	Rationale  string           `json:"rationale,omitempty"`
	Matches    []GuardrailMatch `json:"matches,omitempty"`
}

// GuardrailEvaluation is the result of a guardrail's dry run on a text. Nothing is saved: an
// evaluation does not create an event.
type GuardrailEvaluation struct {
	Guardrail string `json:"guardrail"`
	Stage     string `json:"stage"`
	Triggered bool   `json:"triggered"`
	// Disposition is e.g. passed, blocked, redacted or flagged.
	Disposition string `json:"disposition,omitempty"`
	// Text is the text after the guardrail's action, e.g. redacted.
	Text       string             `json:"text,omitempty"`
	Message    string             `json:"message,omitempty"`
	Error      string             `json:"error,omitempty"`
	DurationMs *float64           `json:"durationMs,omitempty"`
	Findings   []GuardrailFinding `json:"findings,omitempty"`
}

// GuardrailEvent is a guardrail's finding in a real prompt, saved for review.
type GuardrailEvent struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"createdAt,omitempty"`
	// LLMClient is the name of the LLMClient whose prompt triggered it.
	LLMClient   string `json:"llmclient,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Disposition string `json:"disposition,omitempty"`
	Reviewed    *bool  `json:"reviewed,omitempty"`
}

// GuardrailEvents are the latest events of a guardrail.
type GuardrailEvents struct {
	Guardrail string           `json:"guardrail"`
	Events    []GuardrailEvent `json:"events"`
}

// Plugin is a plugin's metadata. Its Kind is the manifest kind, e.g. ApiPlugin or SqlPlugin, and its
// PluginClass the implementation: api, skill, sql, static, websearch or imagesearch.
type Plugin struct {
	Resource
	PluginClass string `json:"pluginClass,omitempty"`
	Kind        string `json:"kind,omitempty"`
}

// LLMClientPluginRef is a plugin of an LLMClient, by its id and name.
type LLMClientPluginRef struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// LLMClientMCPClientRef is an MCPClient of an LLMClient, by the link's id and the MCPClient's name.
type LLMClientMCPClientRef struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// LLMClientGuardrailRef is a guardrail of an LLMClient, by the link's id and the guardrail's name.
type LLMClientGuardrailRef struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// LLMClientTool is something that an LLMClient adds to a prompt: an extension, plugin, MCPClient or
// guardrail.
type LLMClientTool struct {
	// ID is the id of the link row: the LLMClient's extension, plugin, MCPClient or guardrail.
	ID int `json:"id"`
	// Name is the extension's slug, or the linked resource's name.
	Name string `json:"name"`
	// Kind is Extension, or the linked resource's manifest kind: Plugin, MCPClient or Guardrail.
	Kind string `json:"kind"`
}

// LLMClientExtension is an extension of an LLMClient, whose tools the LLMClient offers the LLM, e.g.
// calculator.
type LLMClientExtension struct {
	ID int `json:"id"`
	// Name is the extension's slug, e.g. weather.
	Name string `json:"name,omitempty"`
	// Extension is the Extension's id.
	Extension *int   `json:"extension,omitempty"`
	LLMClient *int   `json:"llmclient,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// LLMClientPlugin is the association of a plugin with an LLMClient.
type LLMClientPlugin struct {
	ID         int    `json:"id"`
	PluginMeta Plugin `json:"pluginMeta"`
	LLMClient  *int   `json:"llmclient,omitempty"`
}

// LLMClientAPIKey is an api key that may call an LLMClient that requires authentication.
type LLMClientAPIKey struct {
	ID        int  `json:"id"`
	LLMClient *int `json:"llmclient,omitempty"`
	// APIKey is the api key's public key_id, a UUID. Never the key itself.
	APIKey    string `json:"apiKey,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// LLMClientCustomDomain is a custom domain on which an LLMClient is deployed: the domain itself, and its verification.
type LLMClientCustomDomain struct {
	ID          int    `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	// DomainName is the domain, e.g. example.com.
	DomainName string `json:"domainName,omitempty"`
	// VerificationStatus is Not Verified, Verifying, Verified or Failed.
	VerificationStatus  string `json:"verificationStatus,omitempty"`
	VerifiedAt          string `json:"verifiedAt,omitempty"`
	VerificationMessage string `json:"verificationMessage,omitempty"`
	// UserProfile is the owner's user profile id.
	UserProfile *int   `json:"userProfile,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// LLMClient is a deployable AI agent, built from an LLM provider and model, a system prompt, plugins
// and functions. Prompt it with [LLMClientsService.Prompt], or start a chat with [LLMClientsService.Chat].
type LLMClient struct {
	Resource
	RFC1034CompliantName string `json:"rfc1034CompliantName,omitempty"`
	Provider             string `json:"provider,omitempty"`
	DefaultModel         string `json:"defaultModel,omitempty"`
	// DefaultSystemRole is the system prompt of a new chat session.
	DefaultSystemRole string `json:"defaultSystemRole,omitempty"`
	Deployed          *bool  `json:"deployed,omitempty"`
	DefaultHost       string `json:"defaultHost,omitempty"`
	DefaultURL        string `json:"defaultUrl,omitempty"`
	CustomHost        string `json:"customHost,omitempty"`
	CustomURL         string `json:"customUrl,omitempty"`
	SandboxHost       string `json:"sandboxHost,omitempty"`
	SandboxURL        string `json:"sandboxUrl,omitempty"`
	// URLLLMClient is where to send prompts: the chat api's prompt url.
	URLLLMClient string `json:"urlLlmclient,omitempty"`
	// URLChatConfig is the chat api's configuration url.
	URLChatConfig string `json:"urlChatConfig,omitempty"`
	// IsAuthenticationRequired reports whether prompting the LLMClient requires an api key.
	IsAuthenticationRequired *bool                   `json:"isAuthenticationRequired,omitempty"`
	Extensions               []LLMClientExtension    `json:"extensions,omitempty"`
	Plugins                  []LLMClientPluginRef    `json:"plugins,omitempty"`
	MCPClients               []LLMClientMCPClientRef `json:"mcpclients,omitempty"`
	Guardrails               []LLMClientGuardrailRef `json:"guardrails,omitempty"`
	// Tools is everything that the LLMClient adds to a prompt: its extensions, plugins, MCPClients and
	// guardrails, in that order.
	Tools         []LLMClientTool         `json:"tools,omitempty"`
	APIKeys       []LLMClientAPIKey       `json:"apiKeys,omitempty"`
	CustomDomains []LLMClientCustomDomain `json:"customDomains,omitempty"`
}

// LLMHost is an open-weight model that the platform serves on its own Kubernetes cluster: the model,
// the inference engine that serves it, and the compute it runs on.
type LLMHost struct {
	Resource
	Spec            JSONObject `json:"spec,omitempty"`
	ModelSource     string     `json:"modelSource,omitempty"`
	ModelRepository string     `json:"modelRepository,omitempty"`
	ModelRevision   string     `json:"modelRevision,omitempty"`
	// ModelTask is e.g. chat, completion or embedding.
	ModelTask string `json:"modelTask,omitempty"`
	// InferenceEngine is e.g. vllm, tgi, tei, ollama or llama_cpp.
	InferenceEngine string `json:"inferenceEngine,omitempty"`
	APIFormat       string `json:"apiFormat,omitempty"`
	// Compute is the id of the LLMHostCompute that the LLMHost runs on.
	Compute         *int   `json:"compute,omitempty"`
	GPUCount        *int   `json:"gpuCount,omitempty"`
	GPUType         string `json:"gpuType,omitempty"`
	Replicas        *int   `json:"replicas,omitempty"`
	ReadyReplicas   *int   `json:"readyReplicas,omitempty"`
	Status          string `json:"status,omitempty"`
	StatusMessage   string `json:"statusMessage,omitempty"`
	EndpointURL     string `json:"endpointUrl,omitempty"`
	PublicURL       string `json:"publicUrl,omitempty"`
	ServedModelName string `json:"servedModelName,omitempty"`
	IsActive        *bool  `json:"isActive,omitempty"`
	DeployedAt      string `json:"deployedAt,omitempty"`
	CostPerHour     string `json:"costPerHour,omitempty"`
}

// LLMHostStatus is an LLMHost's state, read from Kubernetes now.
type LLMHostStatus struct {
	LLMHost       string       `json:"llmhost"`
	Status        string       `json:"status"`
	Message       string       `json:"message,omitempty"`
	Replicas      *int         `json:"replicas,omitempty"`
	ReadyReplicas *int         `json:"readyReplicas,omitempty"`
	Healthy       *bool        `json:"healthy,omitempty"`
	Endpoint      string       `json:"endpoint,omitempty"`
	PublicURL     string       `json:"publicUrl,omitempty"`
	Pods          []JSONObject `json:"pods,omitempty"`
	CheckedAt     string       `json:"checkedAt,omitempty"`
}

// LLMHostLogs are the latest log lines of an LLMHost's pods.
type LLMHostLogs struct {
	LLMHost string    `json:"llmhost"`
	Logs    JSONValue `json:"logs,omitempty"`
}

// LLMHostPlan holds the Kubernetes resources that deploying an LLMHost would create, without creating them.
type LLMHostPlan struct {
	LLMHost   string       `json:"llmhost"`
	Resources []JSONObject `json:"resources,omitempty"`
}

// LLMHostReport is a report of every LLMHost that the user may read: counts, and estimated costs.
type LLMHostReport struct {
	Totals   JSONObject     `json:"totals,omitempty"`
	ByStatus map[string]int `json:"byStatus,omitempty"`
	ByEngine map[string]int `json:"byEngine,omitempty"`
	LLMHosts []JSONObject   `json:"llmhosts,omitempty"`
}

// DiscoveredModel is a model found in a public catalog, e.g. Hugging Face, with the LLMHost manifest
// that would serve it.
type DiscoveredModel struct {
	Catalog     string     `json:"catalog,omitempty"`
	Source      string     `json:"source,omitempty"`
	Repository  string     `json:"repository"`
	Task        string     `json:"task,omitempty"`
	Description string     `json:"description,omitempty"`
	License     string     `json:"license,omitempty"`
	Gated       *bool      `json:"gated,omitempty"`
	Downloads   *int       `json:"downloads,omitempty"`
	Likes       *int       `json:"likes,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Manifest    JSONObject `json:"manifest,omitempty"`
}

// MCPClient is a connection to a Model Context Protocol server: the server, how to reach it, and
// which of its tools the LLM may call.
type MCPClient struct {
	Resource
	// Transport is http, sse or stdio.
	Transport           string   `json:"transport,omitempty"`
	EndpointURL         string   `json:"endpointUrl,omitempty"`
	Command             string   `json:"command,omitempty"`
	Timeout             *int     `json:"timeout,omitempty"`
	AuthType            string   `json:"authType,omitempty"`
	AllowedTools        []string `json:"allowedTools,omitempty"`
	AllowedResources    []string `json:"allowedResources,omitempty"`
	IncludeInstructions *bool    `json:"includeInstructions,omitempty"`
	CacheTTL            *int     `json:"cacheTtl,omitempty"`
	IsActive            *bool    `json:"isActive,omitempty"`
	// Status is e.g. connected or error.
	Status          string `json:"status,omitempty"`
	Priority        *int   `json:"priority,omitempty"`
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	ServerName      string `json:"serverName,omitempty"`
	ServerVersion   string `json:"serverVersion,omitempty"`
	// Tools are the names of every tool that the server offers.
	Tools           []string `json:"tools,omitempty"`
	LastConnectedAt string   `json:"lastConnectedAt,omitempty"`
	LastError       string   `json:"lastError,omitempty"`
}

// MCPClientStatus is the connection state of an MCPClient.
type MCPClientStatus struct {
	Name            string `json:"name"`
	Status          string `json:"status,omitempty"`
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	ServerName      string `json:"serverName,omitempty"`
	ServerVersion   string `json:"serverVersion,omitempty"`
	LastConnected   string `json:"lastConnected,omitempty"`
	LastError       string `json:"lastError,omitempty"`
	// Error is the error of a failed refresh.
	Error string `json:"error,omitempty"`
}

// MCPTool is a tool of an MCP server, which the MCPClient allows the LLM to call.
type MCPTool struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// InputSchema is the tool's arguments, as a JSON Schema.
	InputSchema JSONObject `json:"inputSchema,omitempty"`
	ReadOnly    *bool      `json:"readOnly,omitempty"`
	Destructive *bool      `json:"destructive,omitempty"`
}

// MCPClientTools are an MCPClient's allowed tools, and the server's instructions.
type MCPClientTools struct {
	MCPClientStatus
	Instructions string    `json:"instructions,omitempty"`
	Tools        []MCPTool `json:"tools"`
}

// MCPToolResult is the result of a tool call. Check IsError: a tool reports its own failures in its result.
type MCPToolResult struct {
	// Name is the MCPClient's name.
	Name    string `json:"name"`
	Tool    string `json:"tool"`
	IsError bool   `json:"isError"`
	// Content is the tool's result, as text.
	Content string `json:"content,omitempty"`
}

// Secret is a secret's metadata. The api never returns a secret's value.
type Secret struct {
	Resource
	ExpiresAt    string `json:"expiresAt,omitempty"`
	LastAccessed string `json:"lastAccessed,omitempty"`
}

// Orchestrator is a multi-agent workflow that composes LLMClients.
type Orchestrator struct {
	Resource
	// Strategy is how its LLMClients collaborate: sequential, parallel, supervisor, router, voting or debate.
	Strategy      string `json:"strategy,omitempty"`
	MaxIterations *int   `json:"maxIterations,omitempty"`
	IsActive      *bool  `json:"isActive,omitempty"`
	// LLMClients are the ids of its LLMClients.
	LLMClients []int `json:"llmclients,omitempty"`
}

// Vectorsearch is a retrieval pipeline over a vectorstore.
type Vectorsearch struct {
	Resource
	// SearchType is similarity, similarity_score_threshold or mmr.
	SearchType string `json:"searchType,omitempty"`
	// K is the number of chunks to return.
	K              *int     `json:"k,omitempty"`
	ScoreThreshold *float64 `json:"scoreThreshold,omitempty"`
	// FetchK is the candidates of an mmr search.
	FetchK *int `json:"fetchK,omitempty"`
	// LambdaMult is an mmr search's diversity.
	LambdaMult     *float64  `json:"lambdaMult,omitempty"`
	MetadataFilter JSONValue `json:"metadataFilter,omitempty"`
	IsEnabled      *bool     `json:"isEnabled,omitempty"`
	// Vectorstore is the id of its vectorstore.
	Vectorstore *int `json:"vectorstore,omitempty"`
	// AuthSecret is the id of its Secret, if any.
	AuthSecret *int `json:"authSecret,omitempty"`
}

// PromptSession is a chat session: one conversation with an LLMClient, by its session key.
type PromptSession struct {
	ID          int    `json:"id"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	SessionKey  string `json:"session_key,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	URL         string `json:"url,omitempty"`
	UserProfile *int   `json:"user_profile,omitempty"`
	LLMClient   *int   `json:"llmclient,omitempty"`
}

// SessionRef is a chat session that a history row refers to: the session itself, or only its id.
type SessionRef struct {
	// ID is the session's id.
	ID int
	// Session is the session, when the platform returned it, not only its id.
	Session *PromptSession
}

// UnmarshalJSON decodes a session, or a session's id.
func (s *SessionRef) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '{' {
		s.Session = &PromptSession{}
		if err := json.Unmarshal(data, s.Session); err != nil {
			return err
		}
		s.ID = s.Session.ID
		return nil
	}
	return json.Unmarshal(data, &s.ID)
}

// MarshalJSON encodes the session, or its id.
func (s SessionRef) MarshalJSON() ([]byte, error) {
	if s.Session != nil {
		return json.Marshal(s.Session)
	}
	return json.Marshal(s.ID)
}

// PromptToolCall is a tool call that the LLM made in a chat session.
type PromptToolCall struct {
	ID        int         `json:"id"`
	Prompt    *SessionRef `json:"prompt,omitempty"`
	CreatedAt string      `json:"created_at,omitempty"`
	UpdatedAt string      `json:"updated_at,omitempty"`
	// Plugin is the id of the plugin that the tool belongs to, if any.
	Plugin       *int      `json:"plugin,omitempty"`
	FunctionName string    `json:"function_name,omitempty"`
	FunctionArgs string    `json:"function_args,omitempty"`
	Request      JSONValue `json:"request,omitempty"`
	Response     JSONValue `json:"response,omitempty"`
}

// PromptPluginUsage is a use of a plugin in a chat session.
type PromptPluginUsage struct {
	ID        int         `json:"id"`
	Prompt    *SessionRef `json:"prompt,omitempty"`
	CreatedAt string      `json:"created_at,omitempty"`
	UpdatedAt string      `json:"updated_at,omitempty"`
	Plugin    *Plugin     `json:"plugin,omitempty"`
	InputText string      `json:"input_text,omitempty"`
}

// Proxy is authenticated passthrough access to an LLM provider's own api. The platform holds the
// provider's api key. Callers use their Smarter api key instead, and may reach only AllowedPaths.
type Proxy struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Provider    string `json:"provider,omitempty"`
	// URL is the proxy's base url on the Smarter installation.
	URL          string   `json:"url,omitempty"`
	UpstreamURL  string   `json:"upstreamUrl,omitempty"`
	AllowedPaths []string `json:"allowedPaths,omitempty"`
	IsActive     *bool    `json:"isActive,omitempty"`
}

// Provider is an LLM provider that the platform is configured for, e.g. openai.
type Provider struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	BaseURL     string `json:"baseUrl,omitempty"`
	IsActive    *bool  `json:"isActive,omitempty"`
	// DetailURL is the url of the provider's detail in the web console.
	DetailURL string `json:"detailUrl,omitempty"`
}

// ProviderModel is a model of an LLM provider.
type ProviderModel struct {
	Name              string     `json:"name"`
	Description       string     `json:"description,omitempty"`
	MaxTokens         *int       `json:"maxTokens,omitempty"`
	SupportsStreaming *bool      `json:"supportsStreaming,omitempty"`
	SupportsTools     *bool      `json:"supportsTools,omitempty"`
	Metadata          JSONObject `json:"metadata,omitempty"`
}

// Vectorstore is a vector database, hosted by Smarter or by a provider, for retrieval-augmented generation.
type Vectorstore struct {
	Resource
	// Backend is e.g. qdrant or pinecone.
	Backend string `json:"backend,omitempty"`
	// Hosting is self_hosted or managed.
	Hosting            string `json:"hosting,omitempty"`
	IsActive           *bool  `json:"isActive,omitempty"`
	Dimension          *int   `json:"dimension,omitempty"`
	Metric             string `json:"metric,omitempty"`
	EmbeddingsProvider string `json:"embeddingsProvider,omitempty"`
	EmbeddingsModel    string `json:"embeddingsModel,omitempty"`
	// Status is e.g. pending, ready or error.
	Status        string     `json:"status,omitempty"`
	StatusMessage string     `json:"statusMessage,omitempty"`
	IndexName     string     `json:"indexName,omitempty"`
	EndpointURL   string     `json:"endpointUrl,omitempty"`
	VectorCount   *int       `json:"vectorCount,omitempty"`
	DocumentCount *int       `json:"documentCount,omitempty"`
	SnapshotCount *int       `json:"snapshotCount,omitempty"`
	Spec          JSONObject `json:"spec,omitempty"`
	DeployedAt    string     `json:"deployedAt,omitempty"`
}

// VectorstoreStatus is a vectorstore's state, checked now.
type VectorstoreStatus struct {
	Vectorstore string     `json:"vectorstore"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	VectorCount *int       `json:"vectorCount,omitempty"`
	Stats       JSONObject `json:"stats,omitempty"`
}

// VectorstoreDocument is a document of a vectorstore: a file, a text or a url, split into chunks and embedded.
type VectorstoreDocument struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	// Source is upload, text or url.
	Source      string `json:"source,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	// Status is e.g. pending, loaded, error or deleting.
	Status     string     `json:"status,omitempty"`
	ChunkCount *int       `json:"chunkCount,omitempty"`
	Metadata   JSONObject `json:"metadata,omitempty"`
}

// VectorstoreDocuments are a vectorstore's documents.
type VectorstoreDocuments struct {
	Documents []VectorstoreDocument `json:"documents"`
}

// VectorstoreDocumentsAdded is the result of adding documents. Documents are loaded asynchronously,
// and only once the vectorstore is deployed: see Queued.
type VectorstoreDocumentsAdded struct {
	Documents []VectorstoreDocument `json:"documents"`
	// Duplicates are documents that already existed.
	Duplicates []VectorstoreDocument `json:"duplicates,omitempty"`
	Queued     bool                  `json:"queued"`
	Message    string                `json:"message,omitempty"`
}

// VectorstoreDocumentDeleted is a document that is being deleted, with its chunks.
type VectorstoreDocumentDeleted struct {
	Document string `json:"document"`
	Status   string `json:"status"`
}

// VectorstoreSearchResult is a chunk that matched a search.
type VectorstoreSearchResult struct {
	ID       JSONValue  `json:"id,omitempty"`
	Text     string     `json:"text"`
	Metadata JSONObject `json:"metadata,omitempty"`
	Score    *float64   `json:"score,omitempty"`
}

// VectorstoreSearchResults are the chunks that matched a search, best first.
type VectorstoreSearchResults struct {
	Vectorstore string                    `json:"vectorstore"`
	Results     []VectorstoreSearchResult `json:"results"`
}

// VectorstoreSnapshot is a snapshot of a vectorstore's data.
type VectorstoreSnapshot struct {
	ID        int    `json:"id"`
	Name      string `json:"name,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Status    string `json:"status,omitempty"`
	Scheduled *bool  `json:"scheduled,omitempty"`
	SizeBytes *int64 `json:"sizeBytes,omitempty"`
}

// VectorstoreSnapshots are a vectorstore's snapshots.
type VectorstoreSnapshots struct {
	Snapshots []VectorstoreSnapshot `json:"snapshots"`
}

// VectorstoreRestored is the result of restoring a snapshot.
type VectorstoreRestored struct {
	Vectorstore string `json:"vectorstore"`
	Restored    string `json:"restored"`
	VectorCount *int   `json:"vectorCount,omitempty"`
}

// VectorsearchResults are the chunks that a Vectorsearch found, best first: the results of
// [VectorsearchesService.Run].
type VectorsearchResults struct {
	// Vectorsearch is the Vectorsearch's name.
	Vectorsearch string `json:"vectorsearch"`
	// Vectorstore is the name of the vectorstore that it searched.
	Vectorstore string                    `json:"vectorstore"`
	Results     []VectorstoreSearchResult `json:"results"`
}

// Extension is an installed extension, as the list of extensions shows it: a package of tools,
// settings and resources, installed from the Extension Registry or uploaded.
type Extension struct {
	ID int `json:"id"`
	// Slug is the extension's slug, e.g. weather.
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	// Publisher is the publisher's registry id, or an uploaded archive's authors.
	Publisher string `json:"publisher,omitempty"`
	// Source is registry or upload.
	Source string `json:"source,omitempty"`
	// TrustBadge is official, verified, community or unverified.
	TrustBadge string `json:"trustBadge,omitempty"`
	// TrustBadgeLabel is the badge, as people read it, e.g. Official Smarter Extension.
	TrustBadgeLabel string `json:"trustBadgeLabel,omitempty"`
	Enabled         bool   `json:"enabled"`
	// InstalledVersion is the version that runs, if any.
	InstalledVersion string `json:"installedVersion,omitempty"`
	// LatestVersion is the newest compatible version in the registry, if known.
	LatestVersion   string `json:"latestVersion,omitempty"`
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestCheckedAt string `json:"latestCheckedAt,omitempty"`
	// Error is the last error from loading the extension or running its lifecycle hooks.
	Error string `json:"error,omitempty"`
	// DetailURL is the url of the extension's detail page in the web console.
	DetailURL string `json:"detailUrl,omitempty"`
}

// ExtensionList is the list of installed extensions that the user may see.
type ExtensionList struct {
	Objects []Extension `json:"objects"`
	// RegistryEnabled is whether the Extension Registry's public key is configured.
	RegistryEnabled bool `json:"registryEnabled"`
	// UpdatesAvailable is how many installed extensions have an update in the registry.
	UpdatesAvailable int `json:"updatesAvailable"`
}

// ExtensionTool is a tool of an extension, which LLMClients that use the extension offer the LLM.
type ExtensionTool struct {
	// Name is the tool's name in the extension, e.g. get_weather.
	Name string `json:"name"`
	// FunctionName is the name that the LLM sees, e.g. ext_weather_get_weather.
	FunctionName string `json:"functionName"`
	Description  string `json:"description,omitempty"`
	// Parameters is the JSON Schema of the tool's arguments.
	Parameters JSONObject `json:"parameters,omitempty"`
}

// ExtensionSetting is a setting of an extension. A secret setting's value is never returned: its
// Secret is named instead.
type ExtensionSetting struct {
	Name string `json:"name"`
	// Type is string, integer, float, boolean, url, choice or secret.
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	// Default is the setting's default value: a string, number or boolean, or nil.
	Default JSONValue `json:"default,omitempty"`
	// Choices are the values of a choice setting.
	Choices []string `json:"choices,omitempty"`
	// Value is the configured value. Always nil for a secret setting.
	Value JSONValue `json:"value,omitempty"`
	// SecretID is the id of a secret setting's Secret.
	SecretID *int `json:"secretId,omitempty"`
	// SecretName is the name of a secret setting's Secret.
	SecretName string `json:"secretName,omitempty"`
	// IsSet is whether the setting has a configured value, or a Secret.
	IsSet bool `json:"isSet"`
}

// ExtensionVersion is an installed version of an extension.
type ExtensionVersion struct {
	// ID identifies the version for [ExtensionsService.Rollback].
	ID        int    `json:"id"`
	Version   string `json:"version"`
	SHA256    string `json:"sha256,omitempty"`
	SourceURL string `json:"sourceUrl,omitempty"`
	// InstalledBy is the username of the superuser who installed it.
	InstalledBy string `json:"installedBy,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	// IsInstalled is whether it is the version that runs.
	IsInstalled bool `json:"isInstalled"`
}

// ExtensionResource is a Smarter resource that an extension created.
type ExtensionResource struct {
	// Kind is the resource's manifest kind, e.g. Plugin.
	Kind string `json:"kind"`
	Name string `json:"name"`
	// Owner is the username of the resource's owner.
	Owner string `json:"owner,omitempty"`
	// Version is the extension version that created or last applied it.
	Version string `json:"version,omitempty"`
	// Managed is whether it came from the package's manifests/ directory.
	Managed bool `json:"managed"`
}

// ExtensionSwitch is a feature switch of an extension.
type ExtensionSwitch struct {
	// Name is the switch's full name, e.g. ext_weather_verbose.
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
}

// ExtensionAccount is an account that may use an extension.
type ExtensionAccount struct {
	ID            int    `json:"id"`
	AccountNumber string `json:"accountNumber"`
	CompanyName   string `json:"companyName,omitempty"`
}

// SecretRef is a Secret, by its id and name.
type SecretRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ExtensionDetail is everything about an installed extension: its tools, settings, versions,
// resources, switches and availability. A user who is not a superuser gets less: no settings.
type ExtensionDetail struct {
	Extension
	Homepage string `json:"homepage,omitempty"`
	License  string `json:"license,omitempty"`
	// DocsHTML is the extension's documentation, as HTML rendered safely.
	DocsHTML string `json:"docsHtml,omitempty"`
	// LoadError is why the extension's code does not load, if it does not.
	LoadError string             `json:"loadError,omitempty"`
	Tools     []ExtensionTool    `json:"tools,omitempty"`
	Settings  []ExtensionSetting `json:"settings,omitempty"`
	// MissingSettings are the required settings that block enabling the extension.
	MissingSettings []string            `json:"missingSettings,omitempty"`
	Versions        []ExtensionVersion  `json:"versions,omitempty"`
	Resources       []ExtensionResource `json:"resources,omitempty"`
	Switches        []ExtensionSwitch   `json:"switches,omitempty"`
	// AllAccounts is whether every account may use it. Otherwise only Accounts may.
	AllAccounts bool               `json:"allAccounts"`
	Accounts    []ExtensionAccount `json:"accounts,omitempty"`
	// AvailableSecrets are the platform admin's Secrets, for secret settings.
	AvailableSecrets []SecretRef `json:"availableSecrets,omitempty"`
}

// ExtensionAction is the result of an action on extensions: what happened, and the extension as it
// now is, or nil if the action has no one extension (CheckUpdates) or removed it (Uninstall).
type ExtensionAction struct {
	Message   string           `json:"message"`
	Extension *ExtensionDetail `json:"extension,omitempty"`
}

// RegistryVersion is a released version of an extension in the Extension Registry.
type RegistryVersion struct {
	Version string `json:"version"`
	// RequiresSmarter is the Smarter versions that it works with, e.g. >=0.27.
	RequiresSmarter string `json:"requiresSmarter,omitempty"`
	ReleasedAt      string `json:"releasedAt,omitempty"`
	// Compatible is whether it works with this platform's Smarter version.
	Compatible bool `json:"compatible"`
}

// RegistryExtension is an extension of the Extension Registry.
type RegistryExtension struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
	// Publisher is the publisher's name.
	Publisher    string `json:"publisher,omitempty"`
	PublisherURL string `json:"publisherUrl,omitempty"`
	// TrustBadge is official, verified, community or unverified.
	TrustBadge      string `json:"trustBadge,omitempty"`
	TrustBadgeLabel string `json:"trustBadgeLabel,omitempty"`
	Homepage        string `json:"homepage,omitempty"`
	License         string `json:"license,omitempty"`
	// DocsHTML is the extension's documentation, as HTML rendered safely.
	DocsHTML string `json:"docsHtml,omitempty"`
	// LatestVersion is the newest version that is compatible with this platform.
	LatestVersion string `json:"latestVersion,omitempty"`
	// Versions are every released version, newest first.
	Versions         []RegistryVersion `json:"versions,omitempty"`
	InstalledVersion string            `json:"installedVersion,omitempty"`
	// Action is what the extension allows: install, update, installed, uploaded or incompatible.
	Action string `json:"action"`
	// DetailURL is the installed extension's detail page, if it is installed.
	DetailURL string `json:"detailUrl,omitempty"`
}

// ExtensionRegistry is the Extension Registry's extensions.
type ExtensionRegistry struct {
	// Enabled is whether the registry's public key is configured.
	Enabled bool `json:"enabled"`
	// Error is why the registry cannot be read, if it cannot.
	Error   string              `json:"error,omitempty"`
	Objects []RegistryExtension `json:"objects"`
}

// TokenUsagePoint is the tokens that the user's LLM requests used in one period.
type TokenUsagePoint struct {
	// PeriodStart is the start of the period, in UTC.
	PeriodStart string `json:"period_start"`
	// Records is how many charges the period has.
	Records          int `json:"records"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	// TotalCost is the period's cost, in US dollars.
	TotalCost float64 `json:"total_cost"`
}

// TokenUsageSeries is the tokens that the user's LLM requests used over a window of time, one point
// per period, oldest first. Its JSON is snake_case, unlike most of the REST api's.
type TokenUsageSeries struct {
	// Periodicity is the window, e.g. 24_hours.
	Periodicity string `json:"periodicity"`
	// Period is the period of each point: hour, day or month.
	Period string            `json:"period"`
	Points []TokenUsagePoint `json:"points"`
}

// DataExport is one of the user's data exports, and its progress: a zip file of the user's
// manifests, whose link is emailed to the user.
type DataExport struct {
	ID int `json:"id"`
	// Status is queued, running, succeeded or failed.
	Status string `json:"status"`
	// Step is queued, inventory, manifests, archive, upload, email or done.
	Step string `json:"step"`
	// ResourceCount is how many resources the user owns.
	ResourceCount int `json:"resourceCount"`
	// ManifestCount is how many manifests have been written so far.
	ManifestCount int `json:"manifestCount"`
	// FailedCount is how many resources had no manifest.
	FailedCount int `json:"failedCount"`
	// SizeBytes is the size of the zip file.
	SizeBytes int64 `json:"sizeBytes"`
	// Email is the address that the link is emailed to.
	Email     string `json:"email,omitempty"`
	CreatedAt string `json:"createdAt"`
	// ExpiresAt is when the emailed link expires.
	ExpiresAt string `json:"expiresAt,omitempty"`
	// Error is why the export failed.
	Error string `json:"error,omitempty"`
	// Started is set by [DataExportsService.Start]: true if this call started the export (the
	// platform answered 202), false if one was already in progress (200). It is not in the JSON.
	Started bool `json:"-"`
}

// DataExportOverview is the overview of the user's data exports.
type DataExportOverview struct {
	// Email is the address that the link is emailed to, or empty if the user has none: then no export
	// can be started.
	Email string `json:"email,omitempty"`
	// LinkHours is how many hours the emailed link works.
	LinkHours int `json:"linkHours"`
	// ExportsPerDay is the most exports that a user may start in 24 hours.
	ExportsPerDay int `json:"exportsPerDay"`
	// ExportsLeftToday is how many more the user may start now.
	ExportsLeftToday int `json:"exportsLeftToday"`
	// Latest is the user's most recent export, or nil if they never started one.
	Latest *DataExport `json:"latest,omitempty"`
}
