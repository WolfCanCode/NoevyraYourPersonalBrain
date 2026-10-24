package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
	"strings"
	"testing"
)

func TestMCPProtocolAndCandidatePolicy(t *testing.T) {
	s, e := memory.Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	input := strings.Join([]string{
		`null`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"future","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"brain_remember","arguments":{"content":"architecture candidate"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"brain_remember","arguments":{"content":"global","scope":"personal"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"brain_recall","arguments":{"query":"architecture","limit":0}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"missing"}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/list"}`,
		`{`,
	}, "\n")
	var output bytes.Buffer
	if e = ServeMCP(s, memory.Project{ID: "p", Name: "test", Root: t.TempDir()}, "test", strings.NewReader(input), &output); e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 9 {
		t.Fatalf("responses %d: %s", len(lines), output.String())
	}
	results := []map[string]any{}
	for _, line := range lines {
		var v map[string]any
		if e = json.Unmarshal([]byte(line), &v); e != nil {
			t.Fatal(e)
		}
		results = append(results, v)
	}
	if results[0]["error"].(map[string]any)["code"] != float64(-32600) {
		t.Fatal("null must be invalid")
	}
	if results[1]["error"] == nil {
		t.Fatal("tools before initialization accepted")
	}
	if results[2]["result"].(map[string]any)["protocolVersion"] != "2025-06-18" {
		t.Fatal("unknown protocol blindly accepted")
	}
	if results[3]["result"].(map[string]any)["isError"] != false {
		t.Fatal("candidate write failed")
	}
	for _, i := range []int{4, 5} {
		if results[i]["result"].(map[string]any)["isError"] != true {
			t.Fatal("invalid tool call allowed")
		}
	}
	rows, e := s.Recall("architecture", "p", 8)
	if e != nil || len(rows) != 0 {
		t.Fatal("unapproved memory leaked")
	}
	if results[8]["error"].(map[string]any)["code"] != float64(-32700) {
		t.Fatal("parse error not reported")
	}
}

func TestMCPLifecycleErrors(t *testing.T) {
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	ready := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	for _, tt := range []struct {
		name, prefix, request string
		code                  float64
	}{
		{"not ready", init, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, -32000},
		{"duplicate initialization", init, init, -32600},
		{"missing version", "", `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{}}`, -32602},
		{"unknown tool", init + "\n" + ready, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"unknown"}}`, -32602},
		{"null arguments", init + "\n" + ready, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"brain_recall","arguments":null}}`, -32602},
		{"missing method", "", `{"jsonrpc":"2.0","id":2,"method":"unknown"}`, -32601},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, err := memory.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			var output bytes.Buffer
			input := strings.TrimSpace(tt.prefix + "\n" + tt.request)
			if err := ServeMCP(s, memory.Project{ID: "p"}, "test", strings.NewReader(input), &output); err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			var response struct{ Error struct{ Code float64 } }
			if err := json.Unmarshal([]byte(lines[len(lines)-1]), &response); err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != tt.code {
				t.Fatalf("got %s, want code %v", output.String(), tt.code)
			}
		})
	}
}

type failingMCPWriter struct{ err error }

func (w failingMCPWriter) Write([]byte) (int, error) { return 0, w.err }

func TestMCPTransportErrors(t *testing.T) {
	want := errors.New("output closed")
	err := ServeMCP(nil, memory.Project{}, "test", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`), failingMCPWriter{want})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
	err = ServeMCP(nil, memory.Project{}, "test", strings.NewReader(strings.Repeat("x", 1024*1024+1)), &bytes.Buffer{})
	if err == nil {
		t.Fatal("oversized message accepted")
	}
}
func TestCLIFlagsDoNotSilentlyChangeMeaning(t *testing.T) {
	for _, args := range [][]string{{"--scope"}, {"--bad"}, {"--scope", "personal", "--scope", "project"}} {
		if _, _, e := flags(args, []string{"--scope"}, nil); e == nil {
			t.Fatalf("accepted %+v", args)
		}
	}
}
