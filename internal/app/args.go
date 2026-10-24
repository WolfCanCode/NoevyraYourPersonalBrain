package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func printJSON(out io.Writer, v any) error {
	e := json.NewEncoder(out)
	e.SetIndent("", "  ")
	return e.Encode(v)
}

// flags parses a flat args slice into positional arguments and named flags.
// values lists flags that consume the next token as their value (e.g. --budget 4).
// booleans lists presence-only flags (e.g. --dry-run).
func flags(args []string, values, booleans []string) ([]string, map[string]string, error) {
	known := map[string]bool{}
	for _, v := range values {
		known[v] = true
	}
	for _, v := range booleans {
		known[v] = false
	}
	result := map[string]string{}
	pos := []string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		value, ok := known[a]
		if !ok {
			return nil, nil, fmt.Errorf("unknown option: %s", a)
		}
		if _, ok = result[a]; ok {
			return nil, nil, fmt.Errorf("duplicate option: %s", a)
		}
		if !value {
			result[a] = "true"
			continue
		}
		i++
		if i >= len(args) || strings.HasPrefix(args[i], "--") {
			return nil, nil, fmt.Errorf("%s requires a value", a)
		}
		result[a] = args[i]

	}
	return pos, result, nil
}

func mcpConfig() (any, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"mcpServers": map[string]any{
			"brain": map[string]any{"command": exe, "args": []string{"mcp"}},
		},
	}, nil
}
