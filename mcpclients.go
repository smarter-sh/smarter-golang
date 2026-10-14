package smarter

import (
	"context"
	"net/http"
	"net/url"
)

// MCPClientsService holds the operations of MCPClients: /api/v1/mcpclients/, and the manifest api's
// MCPClient kind. An MCPClient is the platform's connection to a Model Context Protocol server.
type MCPClientsService struct {
	*KindService[MCPClient]
}

// MCPToolsParams are the optional arguments of [MCPClientsService.Tools].
type MCPToolsParams struct {
	// Refresh reads the catalog from the server now, instead of from the cache.
	Refresh bool
}

// Tools returns the tools that an MCPClient allows the LLM to call, from the server's cached
// catalog, with the connection's state and the server's instructions.
func (s *MCPClientsService) Tools(ctx context.Context, id ID, params *MCPToolsParams) (*MCPClientTools, error) {
	path, err := s.path(id, "tools/")
	if err != nil {
		return nil, err
	}
	refresh := params != nil && params.Refresh
	var tools MCPClientTools
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path, query: url.Values(query{}.flag("refresh", refresh))}, &tools); err != nil {
		return nil, err
	}
	return &tools, nil
}

// Refresh reconnects to an MCPClient's server, reloads its catalog, and returns the connection's
// state. Only the owner may refresh.
func (s *MCPClientsService) Refresh(ctx context.Context, id ID) (*MCPClientStatus, error) {
	path, err := s.path(id, "refresh/")
	if err != nil {
		return nil, err
	}
	var status MCPClientStatus
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path}, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// CallTool calls one of an MCPClient's tools with its arguments, as its input schema describes them,
// or nil for none. Only the owner may call tools. Check the result's IsError: a tool reports its own
// failures in its result.
func (s *MCPClientsService) CallTool(ctx context.Context, id ID, tool string, arguments JSONObject) (*MCPToolResult, error) {
	toolSegment, err := ID(tool).segment()
	if err != nil {
		return nil, err
	}
	path, err := s.path(id, "tools/"+toolSegment+"/")
	if err != nil {
		return nil, err
	}
	if arguments == nil {
		arguments = JSONObject{}
	}
	var result MCPToolResult
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, body: arguments}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
