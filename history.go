package smarter

import (
	"context"
	"net/http"
	"net/url"
)

// PromptHistoryService reads the chat sessions of the api key's user, and their tool calls and plugin
// uses, newest first: /api/v1/prompts/history/. An account admin also reads the sessions of their
// account's users, and a superuser every session.
type PromptHistoryService struct {
	c *Client
}

const historyPath = "prompts/history/"

func historyList[T any](ctx context.Context, c *Client, suffix string) ([]T, error) {
	var items []T
	if err := c.http.do(ctx, &request{method: http.MethodGet, path: historyPath + suffix}, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func historyItem[T any](ctx context.Context, c *Client, suffix string, id ID) (*T, error) {
	segment, err := id.segment()
	if err != nil {
		return nil, err
	}
	var item T
	if err := c.http.do(ctx, &request{method: http.MethodGet, path: historyPath + suffix + segment + "/"}, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// Sessions returns the chat sessions.
func (s *PromptHistoryService) Sessions(ctx context.Context) ([]PromptSession, error) {
	return historyList[PromptSession](ctx, s.c, "prompt/")
}

// Session returns a chat session, by id.
func (s *PromptHistoryService) Session(ctx context.Context, id ID) (*PromptSession, error) {
	return historyItem[PromptSession](ctx, s.c, "prompt/", id)
}

// ToolCalls returns the tool calls of the chat sessions, with the LLM requests that made them.
func (s *PromptHistoryService) ToolCalls(ctx context.Context) ([]PromptToolCall, error) {
	return historyList[PromptToolCall](ctx, s.c, "tool-calls/")
}

// ToolCall returns a tool call, by id.
func (s *PromptHistoryService) ToolCall(ctx context.Context, id ID) (*PromptToolCall, error) {
	return historyItem[PromptToolCall](ctx, s.c, "tool-calls/", id)
}

// PluginUsage returns the plugin uses of the chat sessions.
func (s *PromptHistoryService) PluginUsage(ctx context.Context) ([]PromptPluginUsage, error) {
	return historyList[PromptPluginUsage](ctx, s.c, "plugin-usage/")
}

// PluginUse returns a plugin use, by id.
func (s *PromptHistoryService) PluginUse(ctx context.Context, id ID) (*PromptPluginUsage, error) {
	return historyItem[PromptPluginUsage](ctx, s.c, "plugin-usage/", id)
}

// JournalService reads the Smarter Journal: a record of every journaled api request and response.
// Superusers only: each entry's request document holds the caller's headers, including their api key.
type JournalService struct {
	c *Client
}

// JournalListParams are the optional arguments of [JournalService.List].
type JournalListParams struct {
	// Page is the page number, from 1. 1 by default.
	Page int
	// PageSize is how many entries per page, or 0 for the platform's default.
	PageSize int
	// Search selects only entries whose key, thing or command contains this text.
	Search string
	// Ordering is a sort field, e.g. createdAt or -statusCode.
	Ordering string
}

// List returns one page of the journal's entries, newest first, without their request and response
// documents, and its pagination.
func (s *JournalService) List(ctx context.Context, params *JournalListParams) (*JournalPage, error) {
	if params == nil {
		params = &JournalListParams{}
	}
	page := max(params.Page, 1)
	q := query{}.num("page", page).str("search", params.Search).str("ordering", params.Ordering)
	if params.PageSize > 0 {
		q.num("page_size", params.PageSize)
	}
	var result JournalPage
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "journal/", query: url.Values(q)}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Retrieve returns a journal entry, with its request and response documents, by its key: the
// metadata key of a journaled response.
func (s *JournalService) Retrieve(ctx context.Context, key string) (*JournalEntry, error) {
	segment, err := ID(key).segment()
	if err != nil {
		return nil, err
	}
	var entry JournalEntry
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "journal/" + segment + "/"}, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
