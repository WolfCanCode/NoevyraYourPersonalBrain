package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPMergePreservesEnvelopeAndOtherEntries(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		write     func(string, string, MCPServerDef) error
	}{
		{"standard", "mcpServers", writeMCPEntry}, {"opencode", "mcp", writeOpenCodeMCPEntry},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			initial := `{"$schema":"custom","other":42,"` + tc.key + `":{"existing":{"command":"keep"}}}`
			if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
				t.Fatal(err)
			}
			if err := tc.write(path, "brain", MCPServerDef{Command: "brain", Args: []string{"mcp"}}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var top map[string]json.RawMessage
			if err := json.Unmarshal(data, &top); err != nil {
				t.Fatal(err)
			}
			var servers map[string]json.RawMessage
			if err := json.Unmarshal(top[tc.key], &servers); err != nil {
				t.Fatal(err)
			}
			if string(top["$schema"]) != `"custom"` || string(top["other"]) != "42" || servers["existing"] == nil || servers["brain"] == nil {
				t.Fatalf("lost config: %s", data)
			}
		})
	}
}

func TestInspectPackageManifestRetainsNodeForInvalidJSON(t *testing.T) {
	root := t.TempDir()
	for _, content := range []string{`{`, `null`, `{"dependencies":{"React":"1"},"devDependencies":{"@playwright/test":"2"}}`} {
		if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		stacks, signals := map[string]bool{}, map[string]bool{}
		inspectManifest(root, "package.json", stacks, signals)
		if !signals["node"] {
			t.Fatalf("missing node signal for %s", content)
		}
		if content[0] == '{' && len(content) > 1 && (!signals["react"] || !signals["e2e"] || !stacks["react"]) {
			t.Fatalf("missing dependency evidence: %v %v", stacks, signals)
		}
	}
}

func TestMCPMergeAcceptsNullObjects(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		write     func(string, string, MCPServerDef) error
	}{
		{"standard", "mcpServers", writeMCPEntry}, {"opencode", "mcp", writeOpenCodeMCPEntry},
	} {
		for _, initial := range []string{`null`, `{"` + tc.key + `":null}`} {
			t.Run(tc.name+initial, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "config.json")
				if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
					t.Fatal(err)
				}
				if err := tc.write(path, "brain", MCPServerDef{Command: "brain"}); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var top map[string]json.RawMessage
				if err := json.Unmarshal(data, &top); err != nil {
					t.Fatal(err)
				}
				var servers map[string]json.RawMessage
				if err := json.Unmarshal(top[tc.key], &servers); err != nil {
					t.Fatal(err)
				}
				if servers["brain"] == nil {
					t.Fatalf("missing server: %s", data)
				}
			})
		}
	}
}
