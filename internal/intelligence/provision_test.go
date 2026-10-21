package intelligence

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
	"pgregory.net/rapid"
)

// ─── Task 2.5 ─────────────────────────────────────────────────────────────────
// Feature: auto-project-setup, Property 3: ProjectProfile serialization round-trip.
// Validates: Requirement 2.6
func TestProfileSerializationRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		profile := ProjectProfile{
			Root:      rapid.String().Draw(t, "root"),
			Languages: rapid.SliceOf(rapid.String()).Draw(t, "languages"),
			Stacks:    rapid.SliceOf(rapid.String()).Draw(t, "stacks"),
			Signals:   rapid.SliceOf(rapid.String()).Draw(t, "signals"),
			Manifests: rapid.SliceOf(rapid.String()).Draw(t, "manifests"),
			Counts:    rapid.MapOf(rapid.String(), rapid.IntRange(0, 10000)).Draw(t, "counts"),
			Scanned:   rapid.IntRange(0, 4000).Draw(t, "scanned"),
			Truncated: rapid.Bool().Draw(t, "truncated"),
		}
		data, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got ProjectProfile
		if err = json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		// Normalize nil vs empty slices before comparison.
		normalizeProfile := func(p *ProjectProfile) {
			if p.Languages == nil {
				p.Languages = []string{}
			}
			if p.Stacks == nil {
				p.Stacks = []string{}
			}
			if p.Signals == nil {
				p.Signals = []string{}
			}
			if p.Manifests == nil {
				p.Manifests = []string{}
			}
			if p.Counts == nil {
				p.Counts = map[string]int{}
			}
		}
		normalizeProfile(&profile)
		normalizeProfile(&got)
		profileJSON, _ := json.Marshal(profile)
		gotJSON, _ := json.Marshal(got)
		if string(profileJSON) != string(gotJSON) {
			t.Fatalf("round-trip mismatch:\n  want: %s\n  got:  %s", profileJSON, gotJSON)
		}
	})
}

// ─── Task 2.4 ─────────────────────────────────────────────────────────────────
// Feature: auto-project-setup, Property 4: Capability discovery is relevant, sorted, bounded.
// Validates: Requirements 3.1, 3.3
func TestDiscoverCapabilitiesRelevanceSortedBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		langs := rapid.SliceOf(rapid.StringN(1, 20, -1)).Draw(t, "langs")
		stacks := rapid.SliceOf(rapid.StringN(1, 20, -1)).Draw(t, "stacks")
		signals := rapid.SliceOf(rapid.StringN(1, 20, -1)).Draw(t, "signals")
		profile := ProjectProfile{Languages: langs, Stacks: stacks, Signals: signals}
		caps := DiscoverCapabilities(profile, nil)

		// Bounded
		if len(caps) > 10 {
			t.Fatalf("len=%d, want ≤10", len(caps))
		}
		// Sorted non-increasing by confidence
		for i := 1; i < len(caps); i++ {
			if caps[i].Confidence > caps[i-1].Confidence {
				t.Fatalf("not sorted: caps[%d].Confidence=%f > caps[%d].Confidence=%f",
					i, caps[i].Confidence, i-1, caps[i-1].Confidence)
			}
		}
		// Relevance: every returned entry has at least one matching trigger signal
		termSet := buildTermSet(profile)
		for _, cap := range caps {
			matched := false
			for _, sig := range cap.TriggerSignals {
				if termSet[strings.ToLower(sig)] {
					matched = true
					break
				}
			}
			if !matched && cap.Confidence != 0.95 { // 0.95 = trusted skill appended
				t.Fatalf("non-matching capability in results: %s (triggers=%v, terms=%v)",
					cap.Ref, cap.TriggerSignals, termSet)
			}
		}
	})
}

// ─── Task 3.2 ─────────────────────────────────────────────────────────────────
// Feature: auto-project-setup, Property 5: writeMCPEntry is idempotent.
// Validates: Requirement 4.6
func TestWriteMCPEntryIdempotent(t *testing.T) {
	outer := t
	rapid.Check(t, func(t *rapid.T) {
		serverName := rapid.StringMatching("[a-z][a-z0-9-_]{0,39}").Draw(t, "name")
		n := rapid.IntRange(1, 8).Draw(t, "n")
		dir := outer.TempDir()
		path := filepath.Join(dir, "mcp.json")
		entry := MCPServerDef{Command: "/bin/brain", Args: []string{"mcp"}}
		for i := 0; i < n; i++ {
			if err := WriteMCPEntry(path, serverName, entry); err != nil {
				t.Fatalf("call %d: %v", i+1, err)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]map[string]json.RawMessage
		if err = json.Unmarshal(data, &cfg); err != nil {
			t.Fatalf("parse final file: %v", err)
		}
		if _, ok := cfg["mcpServers"][serverName]; !ok {
			t.Fatalf("entry for %q not found after %d calls", serverName, n)
		}
		if len(cfg["mcpServers"]) != 1 {
			t.Fatalf("expected exactly 1 entry, got %d", len(cfg["mcpServers"]))
		}
	})
}

// ─── Task 3.4 ─────────────────────────────────────────────────────────────────

func TestProvisionCatalogCoversAllRequiredSignals(t *testing.T) {
	required := []string{"go", "rust", "python", "typescript", "javascript", "database", "web", "api", "security", "containers", "ci-cd"}
	for _, sig := range required {
		found := false
		for _, cap := range ProvisionCatalog {
			for _, trigger := range cap.TriggerSignals {
				if strings.ToLower(trigger) == sig {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("required signal %q has no covering ProvisionCatalog entry", sig)
		}
	}
}

func TestProvisionCatalogNetworkLabels(t *testing.T) {
	for _, cap := range ProvisionCatalog {
		if cap.Network != NetworkLocal && cap.Network != NetworkNetwork {
			t.Errorf("cap %q has invalid Network=%q", cap.Ref, cap.Network)
		}
	}
}

func TestDiscoverCapabilitiesIncludesTrustedSkills(t *testing.T) {
	profile := ProjectProfile{Signals: []string{"security"}}
	trusted := []memory.Skill{{Ref: "owner/repo@sec", Keyword: "security", ApprovedAt: time.Now().Format(time.RFC3339)}}
	caps := DiscoverCapabilities(profile, trusted)
	found := false
	for _, c := range caps {
		if c.Ref == "owner/repo@sec" {
			found = true
			break
		}
	}
	if !found {
		t.Error("trusted skill with matching keyword not in results")
	}
}

func TestDiscoverCapabilitiesEmptyProfile(t *testing.T) {
	caps := DiscoverCapabilities(ProjectProfile{}, nil)
	if caps == nil {
		t.Error("expected non-nil empty slice for empty profile")
	}
	if len(caps) != 0 {
		t.Errorf("expected 0 caps for empty profile, got %d", len(caps))
	}
}

func TestDiscoverCapabilitiesSortedByConfidence(t *testing.T) {
	profile := ProjectProfile{
		Languages: []string{"go", "typescript"},
		Signals:   []string{"ci-cd", "database", "github", "security", "web", "containers"},
	}
	caps := DiscoverCapabilities(profile, nil)
	if len(caps) == 0 {
		t.Skip("no caps matched")
	}
	confidences := make([]float64, len(caps))
	for i, c := range caps {
		confidences[i] = c.Confidence
	}
	sorted := make([]float64, len(confidences))
	copy(sorted, confidences)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	for i := range confidences {
		if confidences[i] != sorted[i] {
			t.Errorf("not sorted at index %d: got %v, want %v", i, confidences[i], sorted[i])
		}
	}
}

func TestWriteMCPEntryCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "mcp.json")
	entry := MCPServerDef{Command: "/bin/brain", Args: []string{"mcp"}}
	if err := WriteMCPEntry(path, "brain", entry); err != nil {
		t.Fatalf("writeMCPEntry: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestWriteMCPEntryPreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	// Seed with an existing key.
	initial := []byte(`{"other":"value","mcpServers":{}}`)
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatal(err)
	}
	entry := MCPServerDef{Command: "/bin/brain", Args: []string{"mcp"}}
	if err := WriteMCPEntry(path, "brain", entry); err != nil {
		t.Fatalf("writeMCPEntry: %v", err)
	}
	data, _ := os.ReadFile(path)
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, ok := top["other"]; !ok {
		t.Error("'other' key was lost after writeMCPEntry")
	}
}

func TestProvisionSkillApproved(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cap := ProvisionableCapability{
		Kind: "skill", Ref: "owner/repo@test-skill", Name: "Test Skill",
		Description: "A test skill", SkillKeyword: "test",
	}
	in := bytes.NewBufferString("y\n")
	out := &bytes.Buffer{}
	result := Provision(cap, AgentConfigPaths{Paths: map[string]string{}}, store, false, in, out)
	if result.Outcome != OutcomeProvisioned {
		t.Errorf("outcome=%q, want provisioned; error=%q", result.Outcome, result.Error)
	}
}

func TestProvisionSkillDenied(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cap := ProvisionableCapability{
		Kind: "skill", Ref: "owner/repo@test-skill", Name: "Test Skill",
		Description: "A test skill", SkillKeyword: "test",
	}
	in := bytes.NewBufferString("n\n")
	out := &bytes.Buffer{}
	result := Provision(cap, AgentConfigPaths{Paths: map[string]string{}}, store, false, in, out)
	if result.Outcome != OutcomeSkipped {
		t.Errorf("outcome=%q, want skipped", result.Outcome)
	}
}

func TestProvisionMCPNoAgentsInstalled(t *testing.T) {
	dir := t.TempDir()
	store, err := memory.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	cap := ProvisionableCapability{
		Kind: "mcp", Ref: "github", Name: "GitHub MCP",
		TriggerSignals: []string{"github"},
	}
	// Use paths for agents that definitely don't exist
	paths := AgentConfigPaths{Paths: map[string]string{
		"__nonexistent_agent__": filepath.Join(dir, "mcp.json"),
	}}
	out := &bytes.Buffer{}
	result := Provision(cap, paths, store, false, bytes.NewBufferString(""), out)
	if result.Outcome != OutcomeFailed {
		t.Errorf("outcome=%q, want failed", result.Outcome)
	}
}
