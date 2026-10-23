package intelligence

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// NetworkReq classifies whether provisioning requires outbound network access.
type NetworkReq string

const (
	NetworkLocal   NetworkReq = "local"   // no outbound requests
	NetworkNetwork NetworkReq = "network" // requires internet
)

// MCPServerDef is the MCP server entry written to agent config files.
type MCPServerDef struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// ProvisionableCapability is the unified type used by both brain setup
// (discovery + provisioning) and brain advise (Compile recommendations).
// It replaces the unexported capability type formerly in compiler.go.
type ProvisionableCapability struct {
	Kind           string        `json:"kind"`           // "mcp" | "skill" | "cli"
	Ref            string        `json:"ref"`            // stable short identifier
	Name           string        `json:"name"`           // human display name
	Reason         string        `json:"reason"`         // one-line reason (used by Compile)
	Description    string        `json:"description"`    // extended description (used by setup)
	Confidence     float64       `json:"confidence"`     // 0.0–1.0 base confidence score
	TriggerSignals []string      `json:"triggerSignals"` // matched against profile terms
	Network        NetworkReq    `json:"network"`
	MCPServer      *MCPServerDef `json:"mcpServer,omitempty"`    // kind == "mcp"
	SkillKeyword   string        `json:"skillKeyword,omitempty"` // kind == "skill"
	CLIBinary      string        `json:"cliBinary,omitempty"`    // kind == "cli"
	InstallCount   string        `json:"installCount,omitempty"` // e.g. "155.4K" from registry
}

// ProvisionOutcome classifies the result of a single Provision call.
type ProvisionOutcome string

const (
	OutcomeProvisioned ProvisionOutcome = "provisioned"
	OutcomeSkipped     ProvisionOutcome = "skipped"
	OutcomeFailed      ProvisionOutcome = "failed"
)

// ProvisionResult records the outcome of one Provision call.
type ProvisionResult struct {
	Capability ProvisionableCapability `json:"capability"`
	Outcome    ProvisionOutcome        `json:"outcome"`
	Error      string                  `json:"error,omitempty"`
}

// AgentConfigPaths maps each supported agent name to its MCP config file path.
type AgentConfigPaths struct {
	Paths map[string]string // e.g. "kiro" → "/Users/alice/.kiro/settings/mcp.json"
}

// ProvisionCatalog is the authoritative ordered list of provisionable capabilities,
// also read by compiler.go for context-pack recommendations.
// Covers all 11 required signal values: go, rust, python, typescript, javascript,
// database, web, api, security, containers, ci-cd.
var ProvisionCatalog = []ProvisionableCapability{
	{
		Kind: "mcp", Ref: "github", Name: "GitHub MCP", Confidence: 0.88, Network: NetworkNetwork,
		Reason:         "Inspect issues, pull requests, commits and CI state without pasting repository history.",
		Description:    "Access GitHub issues, PRs, commits, and CI results directly from your agent.",
		TriggerSignals: []string{"github", "github-actions", "ci-cd"},
	},
	{
		Kind: "mcp", Ref: "documentation", Name: "Documentation MCP", Confidence: 0.85, Network: NetworkNetwork,
		Reason:         "Retrieve current primary documentation instead of loading broad or stale framework knowledge.",
		Description:    "Fetch live primary-source docs for your detected stack — Go, Rust, Python, Angular, React, Vue, and more.",
		TriggerSignals: []string{"go", "rust", "python", "typescript", "javascript", "api", "angular", "react", "vue", "nextjs", "nestjs", "graphql"},
	},
	{
		Kind: "mcp", Ref: "database", Name: "Database MCP", Confidence: 0.85, Network: NetworkNetwork,
		Reason:         "Inspect schema and query results directly when database evidence is needed.",
		Description:    "Connect to your database to inspect schema, run queries, and verify migrations.",
		TriggerSignals: []string{"database", "sql", "postgres", "mysql", "sqlite", "migration"},
	},
	{
		Kind: "mcp", Ref: "browser", Name: "Browser MCP", Confidence: 0.82, Network: NetworkNetwork,
		Reason:         "Verify browser behavior and UI state instead of reasoning from screenshots or assumptions.",
		Description:    "Control a real browser to verify frontend behavior, forms, and navigation.",
		TriggerSignals: []string{"web", "frontend", "ui", "e2e", "playwright", "angular", "react", "vue", "nextjs", "svelte"},
	},
	{
		Kind: "skill", Ref: "security-review", Name: "Security Review Skill", Confidence: 0.82, Network: NetworkNetwork,
		Reason:         "Apply a focused threat-model and unsafe-input checklist before shipping security-sensitive changes.",
		Description:    "Structured checklist for auth, secrets, injection, and webhook security.",
		TriggerSignals: []string{"security", "auth", "secret", "token", "permission", "webhook", "payment"},
		SkillKeyword:   "security",
	},
	{
		Kind: "mcp", Ref: "figma", Name: "Figma MCP", Confidence: 0.78, Network: NetworkNetwork,
		Reason:         "Read design structure and tokens when implementation depends on a Figma source of truth.",
		Description:    "Access Figma files, components, and design tokens directly.",
		TriggerSignals: []string{"figma", "design", "ui", "component", "frontend", "storybook", "angular", "react", "vue"},
	},
	{
		Kind: "skill", Ref: "api-contract", Name: "API Contract Skill", Confidence: 0.75, Network: NetworkLocal,
		Reason:         "Keep request, response and compatibility constraints explicit across services.",
		Description:    "Define and enforce API contracts before changing adapters.",
		TriggerSignals: []string{"api", "backend", "schema", "contract", "mcp", "grpc", "rest"},
		SkillKeyword:   "api",
	},
	{
		Kind: "skill", Ref: "container-ops", Name: "Container Ops Skill", Confidence: 0.75, Network: NetworkNetwork,
		Reason:         "Manage container build, run, and compose workflows with validated patterns.",
		Description:    "Docker and Compose best practices for build, run, and deployment.",
		TriggerSignals: []string{"containers", "docker", "compose"},
		SkillKeyword:   "containers",
	},
	{
		Kind: "skill", Ref: "test-strategy", Name: "Test Strategy Skill", Confidence: 0.70, Network: NetworkLocal,
		Reason:         "Select the smallest useful unit, integration and end-to-end checks for the change.",
		Description:    "Choose and structure unit, integration, and E2E tests for the task.",
		TriggerSignals: []string{"test", "bug", "refactor", "ci", "release", "ci-cd", "e2e"},
		SkillKeyword:   "test",
	},
	{
		Kind: "cli", Ref: "ripgrep", Name: "ripgrep", Confidence: 0.65, Network: NetworkLocal,
		Reason:         "Locate project evidence with low latency and low context overhead.",
		Description:    "Fast code search — lower latency and context cost than grep.",
		TriggerSignals: []string{"code", "refactor", "bug", "project"},
		CLIBinary:      "rg",
	},
}

// DefaultAgentConfigPaths returns the platform-standard MCP config file paths
// for all four supported agents with ~ expanded (Req 4.3, 6.6).
func DefaultAgentConfigPaths() (AgentConfigPaths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return AgentConfigPaths{}, err
	}
	return AgentConfigPaths{Paths: map[string]string{
		"claude": filepath.Join(home, ".claude.json"),
		"codex":  filepath.Join(home, ".codex", "config.json"),
		"cursor": filepath.Join(home, ".cursor", "mcp.json"),
		"kiro":   filepath.Join(home, ".kiro", "settings", "mcp.json"),
		// OpenCode uses opencode.json with a different envelope: {"mcp":{"brain":{...}}}
		// Pi (pi.dev) follows the same mcpServers convention as Claude/Codex.
		"opencode": filepath.Join(home, ".config", "opencode", "opencode.json"),
		"pi":       filepath.Join(home, ".config", "pi", "mcp.json"),
	}}, nil
}

// DiscoverCapabilities returns at most 10 ProvisionableCapabilities from
// ProvisionCatalog sorted by confidence descending. A catalog entry is included
// when at least one of its TriggerSignals matches—by case-insensitive exact
// comparison—any term in the union of profile.Languages, profile.Stacks, and
// profile.Signals. Trusted skills that match on keyword are appended with
// confidence 0.95 before the cap of 10 is applied (Req 3.1, 3.3, 3.6, 3.7).
func DiscoverCapabilities(profile ProjectProfile, trusted []memory.Skill) []ProvisionableCapability {
	termSet := buildTermSet(profile)
	result := []ProvisionableCapability{}
	seen := map[string]bool{}
	for _, cap := range ProvisionCatalog {
		for _, sig := range cap.TriggerSignals {
			if termSet[strings.ToLower(sig)] {
				result = append(result, cap)
				seen[cap.Kind+":"+cap.Ref] = true
				break
			}
		}
	}
	for _, skill := range trusted {
		kw := strings.ToLower(strings.TrimSpace(skill.Keyword))
		if kw == "" || !termSet[kw] {
			continue
		}
		key := "skill:" + skill.Ref
		if seen[key] {
			continue
		}
		result = append(result, ProvisionableCapability{
			Kind: "skill", Ref: skill.Ref, Name: skill.Ref, Confidence: 0.95, Network: NetworkLocal,
			Reason:         "Previously reviewed skill matches keyword " + kw + ".",
			Description:    "Trusted skill from approved_skills.",
			TriggerSignals: []string{kw},
			SkillKeyword:   kw,
		})
		seen[key] = true
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Confidence > result[j].Confidence })
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// buildTermSet builds a match set from profile Signals and Languages only.
// Stacks (raw dependency names from package.json, go.mod, etc.) are excluded
// because they contain hundreds of entries and cause generic signals like
// "typescript" to match every web project identically.
// Framework-level signals (angular, react, vue, e2e, ...) are injected into
// profile.Signals by the profiler when it detects known framework packages.
func buildTermSet(profile ProjectProfile) map[string]bool {
	set := map[string]bool{}
	for _, v := range profile.Languages {
		set[strings.ToLower(v)] = true
	}
	for _, v := range profile.Signals {
		set[strings.ToLower(v)] = true
	}
	return set
}

// Provision installs or configures a single capability and returns the result.
// When dryRun is true, no files or DB records are modified (Req 3.4, 4.7).
func Provision(cap ProvisionableCapability, agentPaths AgentConfigPaths, store *memory.Store, dryRun bool, in io.Reader, out io.Writer) ProvisionResult {
	switch cap.Kind {
	case "mcp":
		return provisionMCP(cap, agentPaths, dryRun, out)
	case "skill":
		return provisionSkill(cap, store, dryRun, in, out)
	case "cli":
		return provisionCLI(cap, dryRun)
	default:
		return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: "unknown capability kind: " + cap.Kind}
	}
}

func provisionMCP(cap ProvisionableCapability, agentPaths AgentConfigPaths, dryRun bool, out io.Writer) ProvisionResult {
	brainBin, err := os.Executable()
	if err != nil {
		return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: "could not determine brain binary path: " + err.Error()}
	}
	entry := MCPServerDef{Command: brainBin, Args: []string{"mcp"}}
	provisioned := 0
	for agent, cfgPath := range agentPaths.Paths {
		binary := agentBinary(agent)
		if _, lookErr := exec.LookPath(binary); lookErr != nil {
			continue
		}
		if dryRun {
			io.WriteString(out, "  [dry-run] would write mcp entry for "+cap.Ref+" to "+agent+" config: "+cfgPath+"\n")
			provisioned++
			continue
		}
		var wErr error
		if isOpenCodeAgent(agent) {
			wErr = writeOpenCodeMCPEntry(cfgPath, "brain", entry)
		} else {
			wErr = writeMCPEntry(cfgPath, "brain", entry)
		}
		if wErr != nil {
			return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: "failed to write MCP config for " + agent + ": " + wErr.Error()}
		}
		provisioned++
	}
	if provisioned == 0 {
		return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: "no supported agent found in PATH (tried claude, codex, cursor-agent, kiro)"}
	}
	return ProvisionResult{Capability: cap, Outcome: OutcomeProvisioned}
}

func agentBinary(agent string) string {
	if agent == "cursor" {
		return "cursor-agent"
	}
	return agent
}

// isOpenCodeAgent returns true for agents that use the opencode.json MCP envelope
// {"mcp": {"name": {...}}} instead of the standard {"mcpServers": {"name": {...}}}.
func isOpenCodeAgent(agent string) bool {
	return agent == "opencode"
}

func provisionSkill(cap ProvisionableCapability, store *memory.Store, dryRun bool, in io.Reader, out io.Writer) ProvisionResult {
	if dryRun {
		io.WriteString(out, "  [dry-run] would prompt approval for skill: "+cap.Ref+"\n")
		return ProvisionResult{Capability: cap, Outcome: OutcomeSkipped}
	}
	io.WriteString(out, "  Skill: "+cap.Name+"\n  "+cap.Description+"\n  Approve? [y/N]: ")
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "y" || line == "Y" {
			if err := store.Trust(cap.Ref, cap.SkillKeyword); err != nil {
				return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: err.Error()}
			}
			return ProvisionResult{Capability: cap, Outcome: OutcomeProvisioned}
		}
	}
	return ProvisionResult{Capability: cap, Outcome: OutcomeSkipped}
}

func provisionCLI(cap ProvisionableCapability, dryRun bool) ProvisionResult {
	if dryRun {
		return ProvisionResult{Capability: cap, Outcome: OutcomeProvisioned}
	}
	if _, err := exec.LookPath(cap.CLIBinary); err != nil {
		return ProvisionResult{Capability: cap, Outcome: OutcomeFailed, Error: cap.CLIBinary + " not found in PATH; install it manually"}
	}
	return ProvisionResult{Capability: cap, Outcome: OutcomeProvisioned}
}

// atomicWriteJSON writes data to path using a temp-file-then-rename pattern.
// This is atomic on POSIX and best-effort atomic on Windows.
// Parent directories are created as needed.
func atomicWriteJSON(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".brain-mcp-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if _, e := os.Stat(tmpName); e == nil {
			os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// writeMCPEntry atomically merges entry under serverName into the mcpServers object
// of the JSON file at path, creating the file and any missing parent directories.
// Preserves all other top-level keys. Idempotent — calling N times yields the same
// result as calling once (Req 4.6).
func writeMCPEntry(path, serverName string, entry MCPServerDef) error {
	top := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &top)
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := top["mcpServers"]; ok {
		_ = json.Unmarshal(raw, &servers)
	}
	entryBytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	servers[serverName] = entryBytes
	serversBytes, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	top["mcpServers"] = serversBytes
	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteJSON(path, out)
}

// WriteMCPEntry is the exported variant used by tests.
func WriteMCPEntry(path, serverName string, entry MCPServerDef) error {
	return writeMCPEntry(path, serverName, entry)
}

// writeOpenCodeMCPEntry writes into the opencode.json "mcp" envelope,
// which uses {"type":"local","command":[...]} instead of {"command":"...","args":[...]}.
func writeOpenCodeMCPEntry(path, serverName string, entry MCPServerDef) error {
	top := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &top)
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := top["mcp"]; ok {
		_ = json.Unmarshal(raw, &servers)
	}
	// OpenCode format: {"type":"local","command":["binary","arg"],"enabled":true}
	cmd := append([]string{entry.Command}, entry.Args...)
	entryBytes, err := json.Marshal(map[string]any{
		"type":    "local",
		"command": cmd,
		"enabled": true,
	})
	if err != nil {
		return err
	}
	servers[serverName] = entryBytes
	serversBytes, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	top["mcp"] = serversBytes
	// Preserve $schema if already set, or add it.
	if _, ok := top["$schema"]; !ok {
		schema, _ := json.Marshal("https://opencode.ai/config.json")
		top["$schema"] = schema
	}
	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteJSON(path, out)
}
