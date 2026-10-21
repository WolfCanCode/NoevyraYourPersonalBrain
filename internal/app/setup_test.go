package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
	"pgregory.net/rapid"
)

// ─── Task 6.2 ─────────────────────────────────────────────────────────────────
// Feature: auto-project-setup, Property 1: projectRoot traversal bound.
// Validates: Requirements 1.1, 1.7
func TestProjectRootTraversalBound(t *testing.T) {
	outer := t
	rapid.Check(t, func(t *rapid.T) {
		depth := rapid.IntRange(0, 12).Draw(t, "depth")
		// gitLevel is which level from root has .git (depth = invocation depth from root)
		gitLevel := rapid.IntRange(0, depth).Draw(t, "gitLevel")
		base := outer.TempDir()
		// Build a chain of directories: base/0/1/2/.../depth
		chain := make([]string, depth+1)
		chain[0] = base
		for i := 1; i <= depth; i++ {
			chain[i] = filepath.Join(chain[i-1], "sub")
			if err := os.Mkdir(chain[i], 0700); err != nil {
				t.Fatalf("mkdir chain[%d]: %v", i, err)
			}
		}
		// Place .git at chain[gitLevel]
		if err := os.Mkdir(filepath.Join(chain[gitLevel], ".git"), 0700); err != nil {
			t.Fatalf("mkdir .git at level %d: %v", gitLevel, err)
		}
		result := projectRoot(chain[depth])
		distanceFromCwd := depth - gitLevel
		if distanceFromCwd <= 10 {
			if result != chain[gitLevel] {
				t.Fatalf("depth=%d gitLevel=%d: got %q, want %q", depth, gitLevel, result, chain[gitLevel])
			}
		} else {
			// .git is beyond 10 ancestor levels — should fall back to cwd
			if result != chain[depth] {
				t.Fatalf("depth=%d gitLevel=%d (beyond 10): got %q, want %q (cwd fallback)", depth, gitLevel, result, chain[depth])
			}
		}
	})
}

func TestProjectRootFallbackNoGit(t *testing.T) {
	base := t.TempDir()
	dir := base
	// Build 12 levels, no .git anywhere
	for i := 0; i < 12; i++ {
		dir = filepath.Join(dir, "sub")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	result := projectRoot(dir)
	if result != dir {
		t.Errorf("got %q, want cwd fallback %q", result, dir)
	}
}

// ─── Task 6.4 ─────────────────────────────────────────────────────────────────

func testSetupStore(t testing.TB) (*memory.Store, string) {
	t.Helper()
	home := t.TempDir()
	s, err := memory.Open(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, home
}

func testProject(t testing.TB, home string) memory.Project {
	t.Helper()
	dir := t.TempDir()
	// Create a minimal project dir with .git so profile works
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	return memory.Project{ID: "test-proj-" + filepath.Base(dir), Name: "test", Root: dir}
}

func TestRunSetupStatusNoProfile(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	var out, errOut bytes.Buffer
	err := runSetup(store, proj, []string{"--status"}, strings.NewReader(""), &out, &errOut, home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "brain setup") {
		t.Errorf("expected instruction to run 'brain setup', got: %q", out.String())
	}
}

func TestRunSetupStatusWithProfile(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	_, err := store.UpsertProfile(proj.ID, proj.Root, `{"root":"/test","languages":["go"],"stacks":[],"signals":["git"],"manifests":[],"fileCounts":{},"scannedFiles":5,"truncated":false}`, "[]")
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err = runSetup(store, proj, []string{"--status"}, strings.NewReader(""), &out, &errOut, home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outStr := out.String()
	if !strings.Contains(outStr, proj.ID) {
		t.Errorf("expected project ID in output, got: %q", outStr)
	}
	if !strings.Contains(outStr, "Created") {
		t.Errorf("expected timestamp in output, got: %q", outStr)
	}
}

func TestRunSetupDryRun(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	var out, errOut bytes.Buffer
	if err := runSetup(store, proj, []string{"--dry-run"}, strings.NewReader(""), &out, &errOut, home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nothing written to project_profiles
	p, err := store.LatestProfile(proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Error("dry-run should not persist a profile")
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("expected dry-run label in output: %q", out.String())
	}
}

func TestRunSetupRefreshSkipsConfirmation(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	// Insert an existing profile.
	if _, err := store.UpsertProfile(proj.ID, proj.Root, `{"root":"/old"}`, "[]"); err != nil {
		t.Fatal(err)
	}
	// Provide "n" as stdin — if confirmation is shown this would cancel.
	// With --refresh it should skip confirmation and proceed.
	in := strings.NewReader("none\n") // "none" for promptSelection
	var out, errOut bytes.Buffer
	if err := runSetup(store, proj, []string{"--refresh"}, in, &out, &errOut, home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outStr := out.String()
	if strings.Contains(outStr, "cancelled") || strings.Contains(outStr, "Cancelled") {
		t.Errorf("--refresh should not show cancellation message; got: %q", outStr)
	}
}

func TestRunSetupAbortOnDenyConfirmation(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	if _, err := store.UpsertProfile(proj.ID, proj.Root, `{"root":"/existing"}`, "[]"); err != nil {
		t.Fatal(err)
	}
	in := strings.NewReader("n\n")
	var out, errOut bytes.Buffer
	if err := runSetup(store, proj, nil, in, &out, &errOut, home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should NOT have written a new profile (still 1, the original).
	var count int
	if err := store.DB.QueryRow("SELECT count(*) FROM project_profiles WHERE project_id=?", proj.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 profile after cancel, got %d", count)
	}
	if !strings.Contains(out.String(), "cancelled") && !strings.Contains(out.String(), "Cancelled") {
		t.Errorf("expected cancellation message, got: %q", out.String())
	}
}

func TestRunSetupSessionEventRecorded(t *testing.T) {
	store, home := testSetupStore(t)
	proj := testProject(t, home)
	in := strings.NewReader("none\n")
	var out, errOut bytes.Buffer
	if err := runSetup(store, proj, nil, in, &out, &errOut, home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows, err := store.DB.Query("SELECT kind FROM events WHERE project_id=? AND kind='setup_session'", proj.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var found bool
	for rows.Next() {
		found = true
	}
	if !found {
		t.Error("expected a setup_session event to be recorded")
	}
}

// ─── Task 7.4 ─────────────────────────────────────────────────────────────────

func TestDoctorIncludesKiro(t *testing.T) {
	store, _ := testSetupStore(t)
	_ = store
	var out bytes.Buffer
	err := Run([]string{"doctor"}, "test", "abc", time.Now().Format(time.RFC3339), strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}
	var result struct {
		Commands map[string]bool `json:"commands"`
	}
	if err = json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("parse doctor output: %v (raw: %q)", err, out.String())
	}
	if _, ok := result.Commands["kiro"]; !ok {
		t.Error("expected 'kiro' key in commands map")
	}
}

func TestRunAgentKiroNotFound(t *testing.T) {
	// Override PATH to an empty temp dir so no binaries are found.
	oldPath := os.Getenv("PATH")
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })
	os.Setenv("PATH", t.TempDir())

	var stdout, stderr bytes.Buffer
	err := Run([]string{"run", "kiro", "test task"}, "test", "abc", time.Now().Format(time.RFC3339), strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when kiro is not in PATH")
	}
	if !strings.Contains(stderr.String(), "kiro") {
		t.Errorf("expected kiro-specific message in stderr, got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "install") {
		t.Errorf("expected install instruction in stderr, got: %q", stderr.String())
	}
}

func TestMcpConfigIsKiroCompatible(t *testing.T) {
	var out bytes.Buffer
	err := Run([]string{"mcp-config"}, "test", "abc", time.Now().Format(time.RFC3339), strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatalf("mcp-config failed: %v", err)
	}
	var cfg struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	if err = json.Unmarshal(out.Bytes(), &cfg); err != nil {
		t.Fatalf("parse mcp-config: %v", err)
	}
	brain, ok := cfg.MCPServers["brain"]
	if !ok {
		t.Fatal("no 'brain' entry in mcpServers")
	}
	if brain.Command == "" {
		t.Error("brain.command is empty")
	}
	if len(brain.Args) != 1 || brain.Args[0] != "mcp" {
		t.Errorf("brain.args=%v, want [mcp]", brain.Args)
	}
}
