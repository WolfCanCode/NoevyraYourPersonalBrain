package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcProtocolError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcProtocolError) Error() string { return e.Message }

func validID(id json.RawMessage) bool {
	if len(id) == 0 {
		return true
	}
	var value any
	if json.Unmarshal(id, &value) != nil {
		return false
	}
	switch value.(type) {
	case string, float64:
		return true
	}
	return false
}

func decodeRPC(line []byte) (rpcRequest, *rpcProtocolError) {
	var request rpcRequest
	if !json.Valid(line) {
		return request, &rpcProtocolError{-32700, "Parse error"}
	}
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' || json.Unmarshal(line, &request) != nil || request.JSONRPC != "2.0" || request.Method == "" || !validID(request.ID) {
		return rpcRequest{}, &rpcProtocolError{-32600, "Invalid Request"}
	}
	return request, nil
}

func writeRPC(encoder *json.Encoder, id json.RawMessage, result any, protocolErr *rpcProtocolError) error {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	response := map[string]any{"jsonrpc": "2.0", "id": id}
	if protocolErr != nil {
		response["error"] = protocolErr
	} else {
		response["result"] = result
	}
	return encoder.Encode(response)
}

// ServeMCP reads bounded newline-delimited JSON-RPC and keeps protocol failures
// separate from transport failures and tool execution errors.
func ServeMCP(store *memory.Store, project memory.Project, version string, input io.Reader, output io.Writer) error {
	scan := bufio.NewScanner(input)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	encoder := json.NewEncoder(output)
	session := mcpSession{store: store, project: project, version: version}
	for scan.Scan() {
		request, protocolErr := decodeRPC(scan.Bytes())
		if protocolErr != nil {
			if err := writeRPC(encoder, nil, nil, protocolErr); err != nil {
				return err
			}
			continue
		}
		if len(request.ID) == 0 {
			if request.Method == "notifications/initialized" && session.initialized {
				session.ready = true
			}
			continue
		}
		result, err := session.handle(request)
		if err != nil && !errors.As(err, &protocolErr) {
			return err
		}
		if err := writeRPC(encoder, request.ID, result, protocolErr); err != nil {
			return err
		}
	}
	return scan.Err()
}
