package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// This intentionally small stdio adapter supports the stable 2025 lifecycle.
// Protocol versions are negotiated from implemented versions, never echoed blindly.
const toolsJSON = `[
 {"name":"brain_context","description":"Retrieve verified memories for a task in the active project.","inputSchema":{"type":"object","properties":{"task":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":30}},"required":["task"],"additionalProperties":false}},
 {"name":"brain_recall","description":"Search verified project and personal memories.","inputSchema":{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":30}},"required":["query"],"additionalProperties":false}},
 {"name":"brain_remember","description":"Propose a project memory for explicit user approval.","inputSchema":{"type":"object","properties":{"content":{"type":"string","minLength":1,"maxLength":20000},"type":{"type":"string","enum":["identity","semantic","episodic","project","procedural","decision"]},"scope":{"type":"string","enum":["project"]}},"required":["content"],"additionalProperties":false}},
 {"name":"brain_finish_task","description":"Record an outcome without promoting it to verified memory.","inputSchema":{"type":"object","properties":{"summary":{"type":"string"},"status":{"type":"string","enum":["completed","blocked","failed"]},"evidence":{"type":"string"}},"required":["summary","status"],"additionalProperties":false}},
 {"name":"brain_handoff","description":"Record a structured handoff for another agent.","inputSchema":{"type":"object","properties":{"objective":{"type":"string"},"completed":{"type":"array","items":{"type":"string"}},"remaining":{"type":"array","items":{"type":"string"}},"evidence":{"type":"string"}},"required":["objective"],"additionalProperties":false}}
]`

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func strictDecode(raw json.RawMessage, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(out)
}
func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	var v any
	if json.Unmarshal(id, &v) != nil {
		return false
	}
	switch v.(type) {
	case string, float64:
		return true
	}
	return false
}

func ServeMCP(store *memory.Store, project, version string, input io.Reader, output io.Writer) error {
	scan := bufio.NewScanner(input)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	encoder := json.NewEncoder(output)
	initialized, ready := false, false
	send := func(id json.RawMessage, result any, code int, message string) error {
		if len(id) == 0 {
			id = json.RawMessage("null")
		}
		response := map[string]any{"jsonrpc": "2.0", "id": id}
		if code != 0 {
			response["error"] = map[string]any{"code": code, "message": message}
		} else {
			response["result"] = result
		}
		return encoder.Encode(response)
	}
	for scan.Scan() {
		line := scan.Bytes()
		var req rpcRequest
		if !json.Valid(line) {
			if err := send(nil, nil, -32700, "Parse error"); err != nil {
				return err
			}
			continue
		}
		if len(bytes.TrimSpace(line)) == 0 || bytes.TrimSpace(line)[0] != '{' || json.Unmarshal(line, &req) != nil || req.JSONRPC != "2.0" || req.Method == "" || !validID(req.ID) {
			if err := send(nil, nil, -32600, "Invalid Request"); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) == 0 {
			if req.Method == "notifications/initialized" && initialized {
				ready = true
			}
			continue
		}
		var result any
		var err error
		code := 0
		switch req.Method {
		case "initialize":
			var params struct {
				ProtocolVersion string          `json:"protocolVersion"`
				Capabilities    json.RawMessage `json:"capabilities"`
				ClientInfo      json.RawMessage `json:"clientInfo"`
			}
			if initialized {
				code = -32600
				err = errors.New("already initialized")
			} else if json.Unmarshal(req.Params, &params) != nil || params.ProtocolVersion == "" {
				code = -32602
				err = errors.New("protocolVersion required")
			} else {
				protocol := "2025-06-18"
				switch params.ProtocolVersion {
				case "2024-11-05", "2025-03-26", "2025-06-18", "2025-11-25":
					protocol = params.ProtocolVersion
				}
				initialized = true
				result = map[string]any{"protocolVersion": protocol, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}, "serverInfo": map[string]any{"name": "personal-ai-brain", "version": version}}
			}
		case "ping":
			result = map[string]any{}
		case "tools/list", "tools/call":
			if !ready {
				code = -32000
				err = errors.New("initialize and send notifications/initialized first")
				break
			}
			if req.Method == "tools/list" {
				result = map[string]any{"tools": json.RawMessage(toolsJSON)}
				break
			}
			var call struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
				Meta      json.RawMessage `json:"_meta"`
			}
			if json.Unmarshal(req.Params, &call) != nil || call.Name == "" {
				code = -32602
				err = errors.New("invalid tool call")
				break
			}
			if len(call.Arguments) == 0 {
				call.Arguments = json.RawMessage(`{}`)
			}
			if bytes.Equal(bytes.TrimSpace(call.Arguments), []byte("null")) {
				code = -32602
				err = errors.New("arguments must be an object")
				break
			}
			var data any
			data, err = callTool(store, project, call.Name, call.Arguments)
			if errors.Is(err, errUnknownTool) {
				code = -32602
				break
			}
			// Tool execution/validation errors belong in a CallToolResult (MCP tools spec).
			text := ""
			failed := err != nil
			if failed {
				text = err.Error()
			} else {
				var b []byte
				b, err = json.MarshalIndent(data, "", "  ")
				text = string(b)
				if err != nil {
					return err
				}
			}
			result = map[string]any{"content": []any{map[string]any{"type": "text", "text": text}}, "isError": failed}
			err = nil
		default:
			code = -32601
			err = fmt.Errorf("method not found: %s", req.Method)
		}
		message := ""
		if err != nil {
			message = err.Error()
			if code == 0 {
				code = -32603
			}
		}
		if err := send(req.ID, result, code, message); err != nil {
			return err
		}
	}
	return scan.Err()
}

var errUnknownTool = errors.New("unknown tool")

func callTool(s *memory.Store, project, name string, raw json.RawMessage) (any, error) {
	switch name {
	case "brain_context", "brain_recall":
		var args struct {
			Task  string `json:"task"`
			Query string `json:"query"`
			Limit *int   `json:"limit"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		text := args.Task
		if name == "brain_recall" {
			text = args.Query
		}
		if strings.TrimSpace(text) == "" {
			return nil, errors.New("task/query required")
		}
		limit := 8
		if args.Limit != nil {
			limit = *args.Limit
			if limit < 1 || limit > 30 {
				return nil, errors.New("limit must be an integer from 1 to 30")
			}
		}
		if name == "brain_context" {
			return s.Context(text, project, limit)
		}
		return s.Recall(text, project, limit)
	case "brain_remember":
		var args struct {
			Content string `json:"content"`
			Type    string `json:"type"`
			Scope   string `json:"scope"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		if args.Scope != "" && args.Scope != "project" {
			return nil, errors.New("only project candidates are permitted")
		}
		return s.Propose(memory.Input{Content: args.Content, Type: args.Type, Scope: args.Scope, ProjectID: project})
	case "brain_finish_task":
		var args struct {
			Summary  string `json:"summary"`
			Status   string `json:"status"`
			Evidence string `json:"evidence,omitempty"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.Summary) == "" || (args.Status != "completed" && args.Status != "blocked" && args.Status != "failed") {
			return nil, errors.New("summary and a valid status required")
		}
		id, err := s.Log("task_finish", project, args)
		return map[string]any{"eventId": id}, err
	case "brain_handoff":
		var args struct {
			Objective string   `json:"objective"`
			Completed []string `json:"completed"`
			Remaining []string `json:"remaining"`
			Evidence  string   `json:"evidence,omitempty"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.Objective) == "" {
			return nil, errors.New("objective required")
		}
		id, err := s.Log("handoff", project, args)
		return map[string]any{"eventId": id}, err
	}
	return nil, fmt.Errorf("%w: %s", errUnknownTool, name)
}
