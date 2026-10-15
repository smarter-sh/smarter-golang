package smarter

import (
	"bytes"
	"encoding/json"
	"strings"
)

// The Smarter Chat api contract, version 2: an LLMClient's configuration, its prompt api's request
// and response, and the prompt's progress events. These mirror smarter.apps.prompt.contract.models
// in the Smarter platform, whose JSON Schema is chat-contract.schema.json, as the Python SDK's
// smarter.models.chat does. The SDK always asks for version 2 of the contract, so the deprecated
// version 1 models are not mirrored. The SDK's tests check the platform's schema and examples
// against them (contracts_test.go).
//
// Values that the platform may add to, such as a plugin's class or a progress event's type, are
// strings rather than named types. Their known values are listed in the fields' comments.

// ChatContractV2 is the version of the Smarter Chat api contract that the SDK reads.
const ChatContractV2 = 2

// ChatContractV2Capability is the X-Smarter-Capabilities value with which the SDK asks for version 2
// of the contract.
const ChatContractV2Capability = "chat-contract-v2"

// llmMessageRoles are the message roles that the LLM providers accept. Smarter's own roles, smarter
// and smarter_error, are informational.
var llmMessageRoles = map[string]bool{"system": true, "assistant": true, "user": true, "tool": true}

// PromptChatAPIImageURL is an image, by its url or as a data url.
type PromptChatAPIImageURL struct {
	URL string `json:"url"`
	// Detail is auto, low or high.
	Detail string `json:"detail,omitempty"`
}

// PromptChatAPIInputAudio is audio, base64 encoded.
type PromptChatAPIInputAudio struct {
	Data string `json:"data"`
	// Format is wav or mp3.
	Format string `json:"format"`
}

// PromptChatAPIFileRef is a file, base64 encoded, or by the provider's file id.
type PromptChatAPIFileRef struct {
	FileData string `json:"file_data,omitempty"`
	FileID   string `json:"file_id,omitempty"`
	Filename string `json:"filename,omitempty"`
}

// PromptChatAPIContentPart is a part of a message whose content is a list of parts. Its Type says
// which of its other fields it has: text (Text), image_url (ImageURL), input_audio (InputAudio),
// file (File) or refusal (Refusal). It mirrors the platform's PromptChatApiTextPart,
// PromptChatApiImagePart, PromptChatApiAudioPart, PromptChatApiFilePart and PromptChatApiRefusalPart.
type PromptChatAPIContentPart struct {
	Type       string                   `json:"type"`
	Text       string                   `json:"text,omitempty"`
	ImageURL   *PromptChatAPIImageURL   `json:"image_url,omitempty"`
	InputAudio *PromptChatAPIInputAudio `json:"input_audio,omitempty"`
	File       *PromptChatAPIFileRef    `json:"file,omitempty"`
	Refusal    string                   `json:"refusal,omitempty"`
}

// MessageContent is a message's content: a text, or a list of content parts. Build one with
// [TextContent] or [PartsContent].
type MessageContent struct {
	// Text is the content when it is a text.
	Text string
	// Parts is the content when it is a list of parts.
	Parts []PromptChatAPIContentPart
	// IsParts reports whether the content is a list of parts.
	IsParts bool
}

// TextContent returns a text content.
func TextContent(text string) *MessageContent {
	return &MessageContent{Text: text}
}

// PartsContent returns a content of parts.
func PartsContent(parts ...PromptChatAPIContentPart) *MessageContent {
	return &MessageContent{Parts: parts, IsParts: true}
}

// MarshalJSON encodes the content as a string, or as a list of parts.
func (c MessageContent) MarshalJSON() ([]byte, error) {
	if c.IsParts {
		if c.Parts == nil {
			return []byte("[]"), nil
		}
		return json.Marshal(c.Parts)
	}
	return json.Marshal(c.Text)
}

// UnmarshalJSON decodes a string, or a list of parts.
func (c *MessageContent) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		c.IsParts, c.Text = true, ""
		return json.Unmarshal(trimmed, &c.Parts)
	}
	c.IsParts, c.Parts = false, nil
	return json.Unmarshal(trimmed, &c.Text)
}

// PromptChatAPIFunctionCall is the function that a tool call calls.
type PromptChatAPIFunctionCall struct {
	Name string `json:"name"`
	// Arguments are the function's arguments, as a JSON string.
	Arguments string `json:"arguments"`
}

// PromptChatAPIToolCall is a tool call of an assistant message.
type PromptChatAPIToolCall struct {
	ID       string                    `json:"id"`
	Type     string                    `json:"type,omitempty"`
	Function PromptChatAPIFunctionCall `json:"function"`
}

// PromptChatAPIMessage is a message of a chat thread, in the format of the OpenAI api.
//
// Smarter adds its own roles: smarter for informational messages (e.g. the prompt's configuration,
// the tools presented to the LLM, and its token charges), and smarter_error. These are saved in the
// session's history, and are never sent to the LLM.
type PromptChatAPIMessage struct {
	// Role is system, assistant, user, tool, smarter or smarter_error.
	Role string `json:"role"`
	// Content is the message's text, or a list of content parts, or nil, e.g. for an assistant's tool
	// calls. It is always encoded, as null when it is nil: the platform requires the key.
	Content *MessageContent `json:"content"`
	// ToolCalls are an assistant message's tool calls.
	ToolCalls []PromptChatAPIToolCall `json:"tool_calls,omitempty"`
	// ToolCallID is the tool call that a tool message answers.
	ToolCallID string `json:"tool_call_id,omitempty"`
	// Name is the name of a tool message's function.
	Name string `json:"name,omitempty"`
}

// UserMessage returns a user message with a text.
func UserMessage(text string) PromptChatAPIMessage {
	return PromptChatAPIMessage{Role: "user", Content: TextContent(text)}
}

// Text returns the message's text: its content, or the text of its text parts, joined by blank lines.
func (m PromptChatAPIMessage) Text() string {
	if m.Content == nil {
		return ""
	}
	if !m.Content.IsParts {
		return m.Content.Text
	}
	var texts []string
	for _, part := range m.Content.Parts {
		if part.Type == "text" {
			texts = append(texts, part.Text)
		}
	}
	return strings.Join(texts, "\n\n")
}

// PromptChatAPIJournalMetadata is the metadata of a chat response.
type PromptChatAPIJournalMetadata struct {
	// Command is the journal's command, e.g. prompt.
	Command string `json:"command,omitempty"`
	// Thing is the journal's thing, e.g. LLMClient.
	Thing string `json:"thing,omitempty"`
	Key   string `json:"key,omitempty"`
}

// PromptChatAPIErrorBody is the error of a chat error response.
type PromptChatAPIErrorBody struct {
	// ErrorClass is the name of the error's class, e.g. NotFound or LLMProviderError.
	ErrorClass string `json:"errorClass"`
	// Description is a human-readable description.
	Description string `json:"description"`
	// Status is the http status.
	Status int `json:"status"`
}

// PromptChatAPIErrorResponse is a chat error response.
type PromptChatAPIErrorResponse struct {
	Error    PromptChatAPIErrorBody        `json:"error"`
	API      string                        `json:"api,omitempty"`
	Metadata *PromptChatAPIJournalMetadata `json:"metadata,omitempty"`
}

// PromptChatAPIEnvelope holds the fields that every chat response adds beside its data.
type PromptChatAPIEnvelope struct {
	// API is the api version, e.g. "smarter.sh/v1".
	API      string                       `json:"api"`
	Metadata PromptChatAPIJournalMetadata `json:"metadata"`
}

// PromptChatAPIStyleSection holds the colors of one section of the chat ui.
type PromptChatAPIStyleSection struct {
	BackgroundColor       string `json:"background_color,omitempty"`
	Color                 string `json:"color,omitempty"`
	BorderColor           string `json:"border_color,omitempty"`
	InfoColor             string `json:"info_color,omitempty"`
	LinkColor             string `json:"link_color,omitempty"`
	EditorBackgroundColor string `json:"editor_background_color,omitempty"`
	PlaceholderColor      string `json:"placeholder_color,omitempty"`
}

// PromptChatAPIStyles are the LLMClient's chat ui styles.
type PromptChatAPIStyles struct {
	AppLogoURL             string                     `json:"app_logo_url,omitempty"`
	AppBackgroundImageURL  string                     `json:"app_background_image_url,omitempty"`
	AppBackgroundImageSize string                     `json:"app_background_image_size,omitempty"`
	AssistantAvatarURL     string                     `json:"assistant_avatar_url,omitempty"`
	UserAvatarURL          string                     `json:"user_avatar_url,omitempty"`
	FontFamily             string                     `json:"font_family,omitempty"`
	GoogleFont             string                     `json:"google_font,omitempty"`
	FontSize               string                     `json:"font_size,omitempty"`
	BorderRadius           string                     `json:"border_radius,omitempty"`
	MainContainer          *PromptChatAPIStyleSection `json:"main_container,omitempty"`
	ConversationHeader     *PromptChatAPIStyleSection `json:"conversation_header,omitempty"`
	MessageList            *PromptChatAPIStyleSection `json:"message_list,omitempty"`
	IncomingMessage        *PromptChatAPIStyleSection `json:"incoming_message,omitempty"`
	OutgoingMessage        *PromptChatAPIStyleSection `json:"outgoing_message,omitempty"`
	MessageInput           *PromptChatAPIStyleSection `json:"message_input,omitempty"`
	Buttons                *PromptChatAPIStyleSection `json:"buttons,omitempty"`
	TypingIndicator        *PromptChatAPIStyleSection `json:"typing_indicator,omitempty"`
}

// PromptChatAPILLMClient is the LLMClient, as its configuration describes it.
type PromptChatAPILLMClient struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	// URLLLMClient is the url of the LLMClient's prompt api.
	URLLLMClient string `json:"url_llmclient,omitempty"`
	Version      string `json:"version,omitempty"`
	Deployed     *bool  `json:"deployed,omitempty"`
	Provider     string `json:"provider,omitempty"`
	DefaultModel string `json:"default_model,omitempty"`
	// DefaultSystemRole is the system prompt of a new chat session.
	DefaultSystemRole string               `json:"default_system_role,omitempty"`
	AppName           string               `json:"app_name,omitempty"`
	AppAssistant      string               `json:"app_assistant,omitempty"`
	AppWelcomeMessage string               `json:"app_welcome_message,omitempty"`
	AppExamplePrompts []string             `json:"app_example_prompts,omitempty"`
	AppPlaceholder    string               `json:"app_placeholder,omitempty"`
	AppInfoURL        string               `json:"app_info_url,omitempty"`
	AppFileAttachment *bool                `json:"app_file_attachment,omitempty"`
	Styles            *PromptChatAPIStyles `json:"styles,omitempty"`
}

// PromptChatAPIOrmModel holds the fields of a Django row that the contract returns.
type PromptChatAPIOrmModel struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// PromptChatAPIPluginMeta is a plugin's metadata.
type PromptChatAPIPluginMeta struct {
	PromptChatAPIOrmModel
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	// Annotations are a list of JSON objects, or a JSON object.
	Annotations JSONValue `json:"annotations,omitempty"`
	// PluginClass is api, skill, sql, static, websearch or imagesearch.
	PluginClass string `json:"plugin_class"`
}

// PromptChatAPILLMClientPlugin is a plugin of the LLMClient.
type PromptChatAPILLMClientPlugin struct {
	PromptChatAPIOrmModel
	LLMClient  int                     `json:"llmclient"`
	PluginMeta PromptChatAPIPluginMeta `json:"plugin_meta"`
}

// PromptChatAPILLMClientExtension is an extension of the LLMClient, whose tools it offers the LLM.
type PromptChatAPILLMClientExtension struct {
	PromptChatAPIOrmModel
	LLMClient int `json:"llmclient"`
	// Extension is the Extension's id.
	Extension int `json:"extension"`
	// Name is the Extension's slug, e.g. weather.
	Name string `json:"name"`
	// Version is the Extension's installed version, e.g. 1.0.0.
	Version string `json:"version,omitempty"`
	// Enabled reports whether the Extension is enabled, so that its tools are offered.
	Enabled bool `json:"enabled"`
}

// PromptChatAPIPromptRequest is the prompt api's request body.
type PromptChatAPIPromptRequest struct {
	SessionKey string                 `json:"session_key,omitempty"`
	Messages   []PromptChatAPIMessage `json:"messages"`
}

// PromptChatAPILLMClientRequest is a request that Smarter sent to the LLM.
type PromptChatAPILLMClientRequest struct {
	PromptChatAPIOrmModel
	LLMClient     int                         `json:"llmclient"`
	Request       *PromptChatAPIPromptRequest `json:"request,omitempty"`
	SessionKey    string                      `json:"session_key,omitempty"`
	IsAggregation *bool                       `json:"is_aggregation,omitempty"`
}

// PromptChatAPIPromptToolCall is a tool call of the chat session.
type PromptChatAPIPromptToolCall struct {
	PromptChatAPIOrmModel
	Prompt       int       `json:"prompt"`
	Plugin       *int      `json:"plugin,omitempty"`
	FunctionName string    `json:"function_name,omitempty"`
	FunctionArgs string    `json:"function_args,omitempty"`
	Request      JSONValue `json:"request,omitempty"`
	Response     JSONValue `json:"response,omitempty"`
}

// PromptChatAPIPromptPluginUsage is a use of a plugin in the chat session.
type PromptChatAPIPromptPluginUsage struct {
	PromptChatAPIOrmModel
	Prompt    int    `json:"prompt"`
	Plugin    int    `json:"plugin"`
	InputText string `json:"input_text,omitempty"`
}

// PromptChatAPIPluginSelection is a plugin selector's decision in the chat session.
type PromptChatAPIPluginSelection struct {
	PromptChatAPIOrmModel
	PluginSelector int                    `json:"plugin_selector"`
	SearchTerm     string                 `json:"search_term,omitempty"`
	Messages       []PromptChatAPIMessage `json:"messages,omitempty"`
	SessionKey     string                 `json:"session_key,omitempty"`
}

// PromptChatAPIHistory is the chat session's history.
type PromptChatAPIHistory struct {
	// ChatHistory holds the session's messages, which restore the chat thread.
	ChatHistory              []PromptChatAPIMessage           `json:"chat_history"`
	PromptToolCallHistory    []PromptChatAPIPromptToolCall    `json:"prompt_tool_call_history,omitempty"`
	PromptPluginUsageHistory []PromptChatAPIPromptPluginUsage `json:"prompt_plugin_usage_history,omitempty"`
	PluginSelectorHistory    []PromptChatAPIPluginSelection   `json:"plugin_selector_history,omitempty"`
	LLMClientRequestHistory  []PromptChatAPILLMClientRequest  `json:"llmclient_request_history,omitempty"`
}

// PromptChatAPIMetaData is the LLMClient's state.
type PromptChatAPIMetaData struct {
	Ready      *bool `json:"ready,omitempty"`
	IsValid    *bool `json:"is_valid,omitempty"`
	IsDeployed *bool `json:"is_deployed,omitempty"`
}

// PromptChatAPIListMetaData holds the counts of a list of the LLMClient's plugins or extensions.
type PromptChatAPIListMetaData struct {
	TotalPlugins       *int `json:"total_plugins,omitempty"`
	PluginsReturned    *int `json:"plugins_returned,omitempty"`
	TotalExtensions    *int `json:"total_extensions,omitempty"`
	ExtensionsReturned *int `json:"extensions_returned,omitempty"`
}

// PromptChatAPIPluginList holds the LLMClient's plugins.
type PromptChatAPIPluginList struct {
	MetaData PromptChatAPIListMetaData      `json:"meta_data"`
	Plugins  []PromptChatAPILLMClientPlugin `json:"plugins"`
}

// PromptChatAPIExtensionList holds the LLMClient's extensions.
type PromptChatAPIExtensionList struct {
	MetaData   PromptChatAPIListMetaData         `json:"meta_data"`
	Extensions []PromptChatAPILLMClientExtension `json:"extensions"`
}

// PromptChatAPIConfig is an LLMClient's chat configuration: the LLMClient, the chat session's key
// and history, and its plugins and extensions. A new session's configuration has an empty history.
// Pass its SessionKey to the prompt api to continue the session.
type PromptChatAPIConfig struct {
	Contract int `json:"contract,omitempty"`
	// SessionKey is the chat session's key.
	SessionKey string `json:"session_key"`
	// SandboxMode is true for a workbench url.
	SandboxMode bool `json:"sandbox_mode"`
	// DebugMode is the platform's ENABLE_REACTAPP_DEBUG_MODE waffle switch.
	DebugMode  bool                        `json:"debug_mode"`
	MetaData   PromptChatAPIMetaData       `json:"meta_data"`
	LLMClient  PromptChatAPILLMClient      `json:"llmclient"`
	History    PromptChatAPIHistory        `json:"history"`
	Plugins    PromptChatAPIPluginList     `json:"plugins"`
	Extensions *PromptChatAPIExtensionList `json:"extensions,omitempty"`
}

// PromptChatAPIConfigRequest is the configuration api's request body.
type PromptChatAPIConfigRequest struct {
	// SessionKey is the chat session's key, or "" for a new session.
	SessionKey string `json:"session_key,omitempty"`
}

// PromptChatAPIConfigResponse is the configuration api's response.
type PromptChatAPIConfigResponse struct {
	PromptChatAPIEnvelope
	Data PromptChatAPIConfig `json:"data"`
}

// PromptChatAPIChoice is a choice of a chat completion.
type PromptChatAPIChoice struct {
	Index int `json:"index"`
	// FinishReason is e.g. stop, length or tool_calls.
	FinishReason string                `json:"finish_reason,omitempty"`
	Message      *PromptChatAPIMessage `json:"message,omitempty"`
}

// PromptChatAPITokenDetails is the breakdown of a completion's tokens.
type PromptChatAPITokenDetails struct {
	ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
	CachedTokens    *int `json:"cached_tokens,omitempty"`
	AudioTokens     *int `json:"audio_tokens,omitempty"`
}

// PromptChatAPIUsage is a chat completion's token usage.
type PromptChatAPIUsage struct {
	PromptTokens            *int                       `json:"prompt_tokens,omitempty"`
	CompletionTokens        *int                       `json:"completion_tokens,omitempty"`
	TotalTokens             *int                       `json:"total_tokens,omitempty"`
	PromptTokensDetails     *PromptChatAPITokenDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *PromptChatAPITokenDetails `json:"completion_tokens_details,omitempty"`
}

// PromptChatAPICompletion is the LLM's chat completion, in the format of the OpenAI api.
type PromptChatAPICompletion struct {
	ID                string                `json:"id,omitempty"`
	Object            string                `json:"object,omitempty"`
	Created           *int64                `json:"created,omitempty"`
	Model             string                `json:"model,omitempty"`
	Choices           []PromptChatAPIChoice `json:"choices"`
	Usage             *PromptChatAPIUsage   `json:"usage,omitempty"`
	SystemFingerprint string                `json:"system_fingerprint,omitempty"`
	ServiceTier       string                `json:"service_tier,omitempty"`
}

// PromptChatAPIPromptResult is the result of a prompt.
type PromptChatAPIPromptResult struct {
	Contract int `json:"contract,omitempty"`
	// Status is the prompt's http status: 200, or the LLM provider's error status.
	Status int `json:"status"`
	// Messages are the messages that the prompt added to the chat thread, in order.
	Messages []PromptChatAPIMessage `json:"messages"`
	// Completion is the LLM's chat completion.
	Completion *PromptChatAPICompletion `json:"completion,omitempty"`
	// Plugins are the plugins that Smarter selected.
	Plugins []string `json:"plugins"`
	// Tools are the functions that the LLM could call.
	Tools []string `json:"tools"`
}

// AssistantMessages returns the assistant messages that the prompt added: the LLM's replies and tool calls.
func (r PromptChatAPIPromptResult) AssistantMessages() []PromptChatAPIMessage {
	var messages []PromptChatAPIMessage
	for _, message := range r.Messages {
		if message.Role == "assistant" {
			messages = append(messages, message)
		}
	}
	return messages
}

// Text returns the text of the LLM's final reply: the last assistant message that has content, or "".
func (r PromptChatAPIPromptResult) Text() string {
	assistant := r.AssistantMessages()
	for i := len(assistant) - 1; i >= 0; i-- {
		if text := assistant[i].Text(); text != "" {
			return text
		}
	}
	return ""
}

// PromptChatAPIPromptResponse is the prompt api's response. A failed prompt has Error too, and its
// Data.Messages include a smarter_error message. A streamed prompt's error response has only Error.
type PromptChatAPIPromptResponse struct {
	PromptChatAPIEnvelope
	Data  PromptChatAPIPromptResult `json:"data"`
	Error *PromptChatAPIErrorBody   `json:"error,omitempty"`
}

// Text returns the text of the LLM's final reply. See [PromptChatAPIPromptResult.Text].
func (r PromptChatAPIPromptResponse) Text() string {
	return r.Data.Text()
}

// PromptChatAPIProgressEvent is the data of a progress event: a step of a running prompt. Events may
// carry more fields, depending on their type, e.g. tool and arguments: they are in Fields.
type PromptChatAPIProgressEvent struct {
	// Type is llm_request, tool_requested, tool_responded, plugin_called, mcp_tool_called,
	// mcp_tool_responded or mcp_tool_failed.
	Type string `json:"type"`
	// Message is a short sentence, which a chat ui can display as is.
	Message string `json:"message"`
	// Fields holds every field of the event, including Type and Message.
	Fields JSONObject `json:"-"`
}

// UnmarshalJSON decodes the event, and keeps all of its fields in Fields.
func (e *PromptChatAPIProgressEvent) UnmarshalJSON(data []byte) error {
	type plain PromptChatAPIProgressEvent
	if err := json.Unmarshal(data, (*plain)(e)); err != nil {
		return err
	}
	return json.Unmarshal(data, &e.Fields)
}

// PromptChatAPIProgressResult is the data of the final result event.
type PromptChatAPIProgressResult struct {
	// Status is the prompt's http status.
	Status int `json:"status"`
	// Response is exactly the JSON of a non-streaming response, or of an error response.
	Response PromptChatAPIPromptResponse `json:"response"`
}
