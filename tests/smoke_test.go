package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// TestNativeBinary exercises the exact executable CI will package. Without an
// override, go test builds a temporary executable for local development.
func TestNativeBinary(t *testing.T) {
	binary := os.Getenv("BRAIN_TEST_BINARY")
	if binary == "" {
		binary = filepath.Join(t.TempDir(), "brain")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		goBinary, err := exec.LookPath("go")
		if err != nil {
			t.Fatalf("Go toolchain must be on PATH to build the smoke-test executable: %v", err)
		}
		build := exec.CommandContext(ctx, goBinary, "build", "-o", binary, "./cmd/brain")
		build.Dir = ".."
		if output, err := build.CombinedOutput(); err != nil {
			t.Fatalf("build: %v\n%s", err, output)
		}
	} else if !filepath.IsAbs(binary) {
		// Relative overrides are rooted at the repository, independent of the
		// test package directory and Git Bash's Windows path translation.
		binary = filepath.Join("..", binary)
	}
	var err error
	binary, err = filepath.Abs(binary)
	if err != nil {
		t.Fatal(err)
	}
	home, project, other := t.TempDir(), t.TempDir(), t.TempDir()
	// Prevent external git environment from making isolated directories share an identity.
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR"} {
		t.Setenv(key, "")
	}
	identity, err := memory.ProjectFor(project)
	if err != nil {
		t.Fatal(err)
	}
	// Migrate a real schema snapshot rather than using today's Go schema as a fixture.
	legacy, err := os.ReadFile("testdata/legacy.sql")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "brain.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(string(legacy)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO memory(id,type,scope,project_id,content,source,status,created_at,updated_at)
	 VALUES('legacy','decision','project',?,'legacy payment webhook','cli:user','verified','2026-01-01T00:00:00.000Z','2026-01-01T00:00:00.000Z')`, identity.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BRAIN_HOME", home)
	run := func(dir, input string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir, cmd.Stdin = dir, strings.NewReader(input)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("brain %v: %v\n%s\n%s", args, err, output, stderr.String())
		}
		return output
	}
	decode := func(data []byte, out any) {
		t.Helper()
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("decode: %v\n%s", err, data)
		}
	}
	recall := func(dir, query string, want int) {
		t.Helper()
		var rows []memory.Memory
		decode(run(dir, "", "recall", query), &rows)
		if len(rows) != want {
			t.Fatalf("recall %q: got %d, want %d: %+v", query, len(rows), want, rows)
		}
	}
	var version struct{ Version, Commit, BuiltAt, Platform string }
	decode(run(project, "", "version"), &version)
	if version.Version == "" || version.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("version: %+v", version)
	}
	if expected := os.Getenv("BRAIN_EXPECT_VERSION"); expected != "" && (version.Version != expected || version.Commit == "" || version.BuiltAt == "") {
		t.Fatalf("release metadata: %+v, want %s", version, expected)
	}
	var init struct{ CatalogSeeded int }
	decode(run(project, "", "init"), &init)
	if init.CatalogSeeded == 0 {
		t.Fatal("embedded catalog was not seeded")
	}
	recall(project, "payment", 1)
	run(project, "", "remember", "native payment architecture", "--type", "decision")
	recall(project, "payment", 2)
	recall(other, "payment", 0)
	run(project, "", "remember", "concise TypeScript preference", "--scope", "personal")
	recall(other, "TypeScript", 1)
	var pack struct{ Memories []memory.Memory }
	decode(run(other, "", "context", "TypeScript"), &pack)
	if len(pack.Memories) != 1 || pack.Memories[0].Scope != "personal" {
		t.Fatalf("context: %+v", pack)
	}

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"brain_remember","arguments":{"content":"candidate-only nebula"}}}
`
	lines := bytes.Split(bytes.TrimSpace(run(project, input, "mcp")), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("MCP response count: %d", len(lines))
	}
	var response struct {
		ID     int
		Error  json.RawMessage
		Result struct {
			IsError bool
			Content []struct{ Text string }
		}
	}
	decode(lines[1], &response)
	if response.ID != 2 || len(response.Error) != 0 || response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("MCP response: %s", lines[1])
	}
	var candidate memory.Memory
	decode([]byte(response.Result.Content[0].Text), &candidate)
	if candidate.Status != "candidate" || candidate.ID == "" {
		t.Fatalf("candidate: %+v", candidate)
	}
	recall(project, "nebula", 0)
	run(project, "", "approve", candidate.ID)
	recall(project, "nebula", 1)
	recall(other, "nebula", 0)
	run(project, "", "sync")
	var digest struct{ Events []struct{ Kind string } }
	decode(run(project, "", "digest", "--days", "1"), &digest)
	found := false
	for _, e := range digest.Events {
		if e.Kind == "sync" {
			found = true
		}
	}
	if !found {
		t.Fatal("sync event missing")
	}
	var config struct {
		MCPServers map[string]struct {
			Command string
			Args    []string
		}
	}
	decode(run(project, "", "mcp-config"), &config)
	brain := config.MCPServers["brain"]
	got, err := filepath.EvalSymlinks(brain.Command)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(binary)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || len(brain.Args) != 1 || brain.Args[0] != "mcp" {
		t.Fatalf("mcp-config: %+v", brain)
	}
}
