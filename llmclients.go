package smarter

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// LLMClientsService holds the operations of LLMClients: /api/v1/llm-clients/, and the manifest api's
// LLMClient kind. Prompt one, stream a prompt's progress, or hold a multi-turn chat.
type LLMClientsService struct {
	*KindService[LLMClient]
}

// PromptParams are the optional arguments of a prompt.
type PromptParams struct {
	// SessionKey is the chat session's key, from [LLMClientsService.Config], to keep the prompt's
	// history in that session, or "" for none.
	SessionKey string
}

// Text returns a prompt of one user message, for [LLMClientsService.Prompt] and
// [LLMClientsService.Stream].
func Text(text string) []PromptChatAPIMessage {
	return []PromptChatAPIMessage{UserMessage(text)}
}

// requestMessages returns messages as the prompt api's request sends them: only those that the LLM
// accepts. Smarter's own smarter and smarter_error messages are left out, as Smarter Chat leaves them
// out. Every message keeps its content, even when it is null: the platform requires the key.
func requestMessages(messages []PromptChatAPIMessage) []PromptChatAPIMessage {
	result := []PromptChatAPIMessage{}
	for _, message := range messages {
		if llmMessageRoles[message.Role] {
			result = append(result, message)
		}
	}
	return result
}

// promptBody is the prompt api's request body. Its session_key is null when there is none.
type promptBody struct {
	SessionKey *string                `json:"session_key"`
	Messages   []PromptChatAPIMessage `json:"messages,omitempty"`
}

func sessionKeyOf(params *PromptParams) *string {
	if params == nil || params.SessionKey == "" {
		return nil
	}
	return &params.SessionKey
}

// Config returns an LLMClient's chat configuration, and a chat session: an existing session, with its
// history, when params has its SessionKey, or a new one. The configuration's SessionKey identifies it.
func (s *LLMClientsService) Config(ctx context.Context, id ID, params *PromptParams) (*PromptChatAPIConfig, error) {
	path, err := s.path(id, "config/")
	if err != nil {
		return nil, err
	}
	var response PromptChatAPIConfigResponse
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, body: promptBody{SessionKey: sessionKeyOf(params)}}, &response); err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// Prompt prompts an LLMClient with a chat thread, e.g. smarter.Text("What is 17 times 23?").
//
// The platform runs the LLMClient's whole prompt: it selects plugins, calls the LLM, runs the tools
// that the LLM calls, and applies the Guardrails. Prompts are billed by the LLM provider. The
// response's Text() is the LLM's final reply, and Data.Messages every message that the prompt added,
// including Smarter's own. A failed prompt, e.g. one that the LLM provider refused, returns an
// [*APIStatusError] whose Body is the error response.
func (s *LLMClientsService) Prompt(ctx context.Context, id ID, messages []PromptChatAPIMessage, params *PromptParams) (*PromptChatAPIPromptResponse, error) {
	path, err := s.path(id, "prompt/")
	if err != nil {
		return nil, err
	}
	body := promptBody{SessionKey: sessionKeyOf(params), Messages: requestMessages(messages)}
	var response PromptChatAPIPromptResponse
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, body: body}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Stream prompts an LLMClient, and streams its progress: the LLM requests, tool calls and plugin
// calls. The request is sent by the first call of the stream's Next. Close the stream when you are
// done with it. The client's timeout applies to each event.
func (s *LLMClientsService) Stream(ctx context.Context, id ID, messages []PromptChatAPIMessage, params *PromptParams) *PromptStream {
	path, err := s.path(id, "prompt/")
	body := promptBody{SessionKey: sessionKeyOf(params), Messages: requestMessages(messages)}
	return &PromptStream{ctx: ctx, http: s.c.http, request: &request{method: http.MethodPost, path: path, body: body}, err: err}
}

// Chat starts a conversation with an LLMClient, or resumes the one of params.SessionKey.
func (s *LLMClientsService) Chat(ctx context.Context, id ID, params *PromptParams) (*ChatSession, error) {
	config, err := s.Config(ctx, id, params)
	if err != nil {
		return nil, err
	}
	session := &ChatSession{LLMClientID: id, Config: config, llmclients: s}
	if len(config.History.ChatHistory) > 0 {
		session.Messages = append(session.Messages, config.History.ChatHistory...)
	} else {
		// a new thread: the LLMClient's system prompt and welcome message, as Smarter Chat starts one.
		if config.LLMClient.DefaultSystemRole != "" {
			session.Messages = append(session.Messages, PromptChatAPIMessage{Role: "system", Content: TextContent(config.LLMClient.DefaultSystemRole)})
		}
		if config.LLMClient.AppWelcomeMessage != "" {
			session.Messages = append(session.Messages, PromptChatAPIMessage{Role: "assistant", Content: TextContent(config.LLMClient.AppWelcomeMessage)})
		}
	}
	return session, nil
}

// list reads a list of an LLMClient's related rows.
func llmclientList[T any](ctx context.Context, s *LLMClientsService, id ID, suffix string) ([]T, error) {
	path, err := s.path(id, suffix)
	if err != nil {
		return nil, err
	}
	var items []T
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path}, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Plugins returns an LLMClient's plugins.
func (s *LLMClientsService) Plugins(ctx context.Context, id ID) ([]LLMClientPlugin, error) {
	return llmclientList[LLMClientPlugin](ctx, s, id, "plugins/")
}

// Functions returns an LLMClient's built-in functions, e.g. calculator.
func (s *LLMClientsService) Functions(ctx context.Context, id ID) ([]LLMClientFunction, error) {
	return llmclientList[LLMClientFunction](ctx, s, id, "functions")
}

// APIKeys returns the api keys that may call an LLMClient that requires authentication.
func (s *LLMClientsService) APIKeys(ctx context.Context, id ID) ([]LLMClientAPIKey, error) {
	return llmclientList[LLMClientAPIKey](ctx, s, id, "apikeys/")
}

// CustomDomains returns the custom domains of an LLMClient.
func (s *LLMClientsService) CustomDomains(ctx context.Context, id ID) ([]LLMClientCustomDomain, error) {
	return llmclientList[LLMClientCustomDomain](ctx, s, id, "customdomains")
}

// Deploy deploys an LLMClient on its own host name. This creates real, possibly billable
// infrastructure: DNS records, a TLS certificate and a Kubernetes ingress.
func (s *LLMClientsService) Deploy(ctx context.Context, name string) (*JournalMessage, error) {
	return s.c.Manifests.Deploy(ctx, s.Kind, name)
}

// Undeploy undoes an LLMClient's deployment.
func (s *LLMClientsService) Undeploy(ctx context.Context, name string) (*JournalMessage, error) {
	return s.c.Manifests.Undeploy(ctx, s.Kind, name)
}

// PromptStream is a prompt whose progress is streamed as Server-Sent Events. Read it like a
// bufio.Scanner:
//
//	stream := client.LLMClients.Stream(ctx, id, smarter.Text("What is sin(pi / 2)?"), nil)
//	defer stream.Close()
//	for stream.Next() {
//		fmt.Println(stream.Current().Message)
//	}
//	response, err := stream.Response()
//
// Response is the prompt's response, exactly as a non-streaming prompt returns it, once Next has
// returned false.
type PromptStream struct {
	ctx      context.Context
	http     *httpClient
	request  *request
	body     io.ReadCloser
	done     context.CancelFunc
	reader   *sseReader
	current  PromptChatAPIProgressEvent
	result   *PromptChatAPIProgressResult
	err      error
	started  bool
	finished bool
	onResult func(*PromptChatAPIProgressResult)
}

// Next reads the next progress event, which Current returns. It sends the request on its first
// call. It returns false at the end of the stream, or on an error, which Err returns.
func (s *PromptStream) Next() bool {
	if s.err != nil || s.finished {
		return false
	}
	if !s.started {
		s.started = true
		response, done, err := s.http.send(s.ctx, s.request, true)
		if err != nil {
			s.err = err
			return false
		}
		s.body, s.done, s.reader = response.Body, done, newSSEReader(response.Body)
	}
	for {
		event, err := s.reader.next()
		if errors.Is(err, io.EOF) {
			s.finish()
			return false
		}
		if err != nil {
			body, _ := s.body.(*idleBody)
			s.err = &APIConnectionError{Method: s.request.method, URL: s.http.url(s.request.path, nil), Err: err, timeout: body != nil && body.expired.Load()}
			if s.ctx.Err() != nil {
				s.err = &APIConnectionError{Method: s.request.method, URL: s.http.url(s.request.path, nil), Err: s.ctx.Err()}
			}
			s.finish()
			return false
		}
		switch event.Event {
		case "progress":
			var progress PromptChatAPIProgressEvent
			if err := json.Unmarshal([]byte(event.Data), &progress); err != nil {
				s.err = &APIResponseValidationError{Method: s.request.method, URL: s.http.url(s.request.path, nil), StatusCode: http.StatusOK, Body: []byte(event.Data), Err: err}
				return false
			}
			s.current = progress
			return true
		case "result":
			var result PromptChatAPIProgressResult
			if err := json.Unmarshal([]byte(event.Data), &result); err != nil {
				s.err = &APIResponseValidationError{Method: s.request.method, URL: s.http.url(s.request.path, nil), StatusCode: http.StatusOK, Body: []byte(event.Data), Err: err}
				return false
			}
			s.result = &result
			if s.onResult != nil {
				s.onResult(&result)
			}
		}
	}
}

func (s *PromptStream) finish() {
	s.finished = true
	_ = s.Close()
}

// Current returns the progress event that the last call of Next read, e.g. "Calling tool calculator".
func (s *PromptStream) Current() PromptChatAPIProgressEvent { return s.current }

// Err returns the error that ended the stream, if any.
func (s *PromptStream) Err() error { return s.err }

// Result returns the final result event, once the stream has been read: the prompt's status and
// response. It is nil until then.
func (s *PromptStream) Result() *PromptChatAPIProgressResult { return s.result }

// Response returns the prompt's response, once the stream has been read. It returns the stream's
// error, a [*PromptError] if the prompt failed, or an error that matches [ErrSmarter] if the stream
// hasn't been read, or ended without a result.
func (s *PromptStream) Response() (*PromptChatAPIPromptResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result == nil {
		return nil, &ValueError{Message: "the stream has not been read to its end, or it ended without a result event"}
	}
	response := s.result.Response
	if response.Error != nil {
		return nil, &PromptError{Message: response.Error.Description, Response: &response}
	}
	return &response, nil
}

// UntilDone reads the stream to its end, discarding the progress events, and returns the prompt's
// response. It closes the stream.
func (s *PromptStream) UntilDone() (*PromptChatAPIPromptResponse, error) {
	for s.Next() { //nolint:revive // the progress events are discarded
	}
	_ = s.Close()
	return s.Response()
}

// Close closes the stream's connection. It is safe to call more than once.
func (s *PromptStream) Close() error {
	if s.body == nil {
		return nil
	}
	err := s.body.Close()
	s.body = nil
	if s.done != nil {
		s.done()
	}
	return err
}

// ChatSession is a multi-turn conversation with an LLMClient. It keeps the chat session's key, so
// that the platform keeps the session's history, and the thread, which it sends with each prompt as
// Smarter Chat does. Create one with [LLMClientsService.Chat]. A session is not safe for concurrent use.
type ChatSession struct {
	// LLMClientID is the LLMClient's id or hashed id.
	LLMClientID ID
	// Config is the chat session's configuration.
	Config *PromptChatAPIConfig
	// Messages is the chat thread.
	Messages   []PromptChatAPIMessage
	llmclients *LLMClientsService
}

// SessionKey returns the chat session's key, with which the session can be resumed: see
// [LLMClientsService.Chat].
func (s *ChatSession) SessionKey() string { return s.Config.SessionKey }

// Send sends a user message, and adds it and the prompt's messages to the thread. The response's
// Text() is the LLM's reply.
func (s *ChatSession) Send(ctx context.Context, text string) (*PromptChatAPIPromptResponse, error) {
	return s.SendMessage(ctx, UserMessage(text))
}

// SendMessage sends a message, and adds it and the prompt's messages to the thread.
func (s *ChatSession) SendMessage(ctx context.Context, message PromptChatAPIMessage) (*PromptChatAPIPromptResponse, error) {
	thread := append(append([]PromptChatAPIMessage{}, s.Messages...), message)
	response, err := s.llmclients.Prompt(ctx, s.LLMClientID, thread, &PromptParams{SessionKey: s.SessionKey()})
	if err != nil {
		return nil, err
	}
	s.Messages = append(thread, response.Data.Messages...)
	return response, nil
}

// Stream sends a user message, and streams the prompt's progress. The thread is updated when the
// stream has been read to its end, if the prompt succeeded.
func (s *ChatSession) Stream(ctx context.Context, text string) *PromptStream {
	return s.StreamMessage(ctx, UserMessage(text))
}

// StreamMessage sends a message, and streams the prompt's progress. See [ChatSession.Stream].
func (s *ChatSession) StreamMessage(ctx context.Context, message PromptChatAPIMessage) *PromptStream {
	thread := append(append([]PromptChatAPIMessage{}, s.Messages...), message)
	stream := s.llmclients.Stream(ctx, s.LLMClientID, thread, &PromptParams{SessionKey: s.SessionKey()})
	stream.onResult = func(result *PromptChatAPIProgressResult) {
		if result.Response.Error == nil {
			s.Messages = append(thread, result.Response.Data.Messages...)
		}
	}
	return stream
}
