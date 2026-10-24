package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSteeringPreservesExistingContentAndIsIdempotent(t *testing.T) {
	root, bin := t.TempDir(), t.TempDir()
	t.Chdir(root)
	t.Setenv("PATH", bin)
	name := "claude"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	// Steering only checks installation with LookPath; it never executes agents.
	if err := os.WriteFile(filepath.Join(bin, name), []byte("test placeholder"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("CLAUDE.md", []byte("# Existing project instructions\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var output, diagnostics bytes.Buffer
	if err := runSteer(nil, &output, &diagnostics); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(first), "# Existing project instructions\n") || !strings.Contains(string(first), "brain_recall") {
		t.Fatalf("content: %s", first)
	}
	if err := runSteer(nil, &output, &diagnostics); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile("CLAUDE.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("second run changed configured file")
	}
	if diagnostics.Len() != 0 {
		t.Fatal(diagnostics.String())
	}
}

func TestSteeringDryRunWritesNoFiles(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("PATH", t.TempDir())
	var output bytes.Buffer
	if err := runSteer([]string{"--dry-run"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("dry run wrote files: %v %v", entries, err)
	}
	if !strings.Contains(output.String(), "not in PATH") {
		t.Fatal(output.String())
	}
}
