package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

type mcpSession struct {
	store       *memory.Store
	project     memory.Project
	version     string
	initialized bool
	ready       bool
}

func (s *mcpSession) handle(request rpcRequest) (any, error) {
	switch request.Method {
	case "initialize":
		return s.initialize(request.Params)
	case "ping":
		return map[string]any{}, nil
	case "tools/list", "tools/call":
		if !s.ready {
			return nil, &rpcProtocolError{-32000, "initialize and send notifications/initialized first"}
		}
		if request.Method == "tools/list" {
			return map[string]any{"tools": json.RawMessage(toolsJSON)}, nil
		}
		return s.call(request.Params)
	default:
		return nil, &rpcProtocolError{-32601, fmt.Sprintf("method not found: %s", request.Method)}
	}
}

func (s *mcpSession) initialize(raw json.RawMessage) (any, error) {
	if s.initialized {
		return nil, &rpcProtocolError{-32600, "already initialized"}
	}
	var params struct {
		ProtocolVersion string          `json:"protocolVersion"`
		Capabilities    json.RawMessage `json:"capabilities"`
		ClientInfo      json.RawMessage `json:"clientInfo"`
	}
	if json.Unmarshal(raw, &params) != nil || params.ProtocolVersion == "" {
		return nil, &rpcProtocolError{-32602, "protocolVersion required"}
	}
	protocol := "2025-06-18"
	switch params.ProtocolVersion {
	case "2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25":
		protocol = params.ProtocolVersion
	}
	s.initialized = true
	return map[string]any{"protocolVersion": protocol, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "personal-ai-brain", "version": s.version}}, nil
}

func (s *mcpSession) call(raw json.RawMessage) (any, error) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
		Meta      json.RawMessage `json:"_meta"`
	}
	if json.Unmarshal(raw, &call) != nil || call.Name == "" {
		return nil, &rpcProtocolError{-32602, "invalid tool call"}
	}
	if len(call.Arguments) == 0 {
		call.Arguments = json.RawMessage(`{}`)
	}
	if bytes.Equal(bytes.TrimSpace(call.Arguments), []byte("null")) {
		return nil, &rpcProtocolError{-32602, "arguments must be an object"}
	}
	data, err := callTool(s.store, s.project, call.Name, call.Arguments)
	if errors.Is(err, errUnknownTool) {
		return nil, &rpcProtocolError{-32602, err.Error()}
	}
	// Tool validation and execution failures are CallToolResult errors, not RPC errors.
	if err != nil {
		return mcpToolResult(err.Error(), true), nil
	}
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcpToolResult(string(body), false), nil
}

func mcpToolResult(text string, failed bool) map[string]any {
	return map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}, "isError": failed}
}
