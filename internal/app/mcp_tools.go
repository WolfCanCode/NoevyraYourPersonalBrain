package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// This intentionally small stdio adapter supports the stable 2025 lifecycle.
// Protocol versions are negotiated from implemented versions, never echoed blindly.
const toolsJSON = `[
 {"name":"brain_advise","description":"Compile a token-budgeted evidence pack, project profile and skill/MCP/CLI recommendations without invoking an LLM.","inputSchema":{"type":"object","properties":{"task":{"type":"string"},"budgetTokens":{"type":"integer","minimum":128,"maximum":16000}},"required":["task"],"additionalProperties":false}},
 {"name":"brain_profile","description":"Detect the active project's languages, stack and operational signals from bounded local metadata.","inputSchema":{"type":"object","properties":{},"additionalProperties":false}},
 {"name":"brain_feedback","description":"Report whether a context pack helped so future knowledge ranking can improve.","inputSchema":{"type":"object","properties":{"packId":{"type":"string"},"outcome":{"type":"string","enum":["helpful","partial","harmful"]},"tokensUsed":{"type":"integer","minimum":0},"notes":{"type":"string"}},"required":["packId","outcome"],"additionalProperties":false}},
 {"name":"brain_context","description":"Retrieve verified memories for a task in the active project.","inputSchema":{"type":"object","properties":{"task":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":30}},"required":["task"],"additionalProperties":false}},
 {"name":"brain_recall","description":"Search verified project and personal memories.","inputSchema":{"type":"object","properties":{"query":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":30}},"required":["query"],"additionalProperties":false}},
 {"name":"brain_remember","description":"Propose a project memory for explicit user approval.","inputSchema":{"type":"object","properties":{"content":{"type":"string","minLength":1,"maxLength":20000},"type":{"type":"string","enum":["identity","semantic","episodic","project","procedural","decision"]},"scope":{"type":"string","enum":["project"]}},"required":["content"],"additionalProperties":false}},
 {"name":"brain_finish_task","description":"Record an outcome without promoting it to verified memory.","inputSchema":{"type":"object","properties":{"summary":{"type":"string"},"status":{"type":"string","enum":["completed","blocked","failed"]},"evidence":{"type":"string"}},"required":["summary","status"],"additionalProperties":false}},
 {"name":"brain_handoff","description":"Record a structured handoff for another agent.","inputSchema":{"type":"object","properties":{"objective":{"type":"string"},"completed":{"type":"array","items":{"type":"string"}},"remaining":{"type":"array","items":{"type":"string"}},"evidence":{"type":"string"}},"required":["objective"],"additionalProperties":false}}
]`

func strictDecode(raw json.RawMessage, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(out)
}

var errUnknownTool = errors.New("unknown tool")

func callTool(s *memory.Store, project memory.Project, name string, raw json.RawMessage) (any, error) {
	switch name {
	case "brain_advise":
		var args struct {
			Task         string `json:"task"`
			BudgetTokens int    `json:"budgetTokens"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		return intelligence.Compile(s, project, args.Task, args.BudgetTokens)
	case "brain_profile":
		var args struct{}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		return intelligence.Profile(project.Root)
	case "brain_feedback":
		var args struct {
			PackID     string `json:"packId"`
			Outcome    string `json:"outcome"`
			TokensUsed int    `json:"tokensUsed"`
			Notes      string `json:"notes"`
		}
		if err := strictDecode(raw, &args); err != nil {
			return nil, err
		}
		return intelligence.Feedback(s, args.PackID, args.Outcome, args.TokensUsed, args.Notes)
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
			return s.Context(text, project.ID, limit)
		}
		return s.Recall(text, project.ID, limit)
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
		return s.Propose(memory.Input{Content: args.Content, Type: args.Type, Scope: args.Scope, ProjectID: project.ID})
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
		id, err := s.Log("task_finish", project.ID, args)
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
		id, err := s.Log("handoff", project.ID, args)
		return map[string]any{"eventId": id}, err
	}
	return nil, fmt.Errorf("%w: %s", errUnknownTool, name)
}
