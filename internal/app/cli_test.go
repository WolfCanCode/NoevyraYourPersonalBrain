package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCommandValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"unknown", []string{"unknown"}, `unknown command "unknown"; run brain help`},
		{"approve missing ID", []string{"approve"}, "approve requires one memory ID"},
		{"approve unknown ID", []string{"approve", "missing"}, "candidate memory not found"},
		{"query unknown option", []string{"recall", "--limit", "2"}, "unknown option: --limit"},
		{"profile argument", []string{"profile", "extra"}, "profile accepts no arguments"},
		{"feedback missing ID", []string{"feedback"}, "feedback requires one context pack ID"},
		{"budget not integer", []string{"advise", "task", "--budget", "abc"}, "budget must be an integer"},
		{"digest not integer", []string{"digest", "--days", "abc"}, "days must be an integer"},
		{"digest extra argument", []string{"digest", "extra"}, "digest accepts only --days"},
		{"sync extra argument", []string{"sync", "extra"}, "sync accepts only --sources, --skills or --online"},
		{"sources missing command", []string{"sources"}, "use sources seed|add|list|sync|pause"},
		{"sources list argument", []string{"sources", "list", "extra"}, "sources list accepts no arguments"},
		{"skills missing command", []string{"skills"}, "use skills find|trust|list"},
		{"skills missing query", []string{"skills", "find"}, "search term required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BRAIN_HOME", t.TempDir())
			var stdout, stderr bytes.Buffer
			err := Run(tt.args, "test", "commit", "time", strings.NewReader(""), &stdout, &stderr)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("got %v, want %q", err, tt.want)
			}
			if stdout.Len() != 0 {
				t.Fatalf("validation wrote stdout: %s", stdout.String())
			}
		})
	}
}

func TestVersionAndConfigDoNotCreateStore(t *testing.T) {
	for _, command := range []string{"version", "--version", "mcp-config"} {
		t.Run(command, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "unused")
			t.Setenv("BRAIN_HOME", home)
			var output bytes.Buffer
			if err := Run([]string{command}, "test-version", "test-commit", "test-time", strings.NewReader(""), &output, &output); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if command != "mcp-config" && (result["version"] != "test-version" || result["commit"] != "test-commit" || result["builtAt"] != "test-time") {
				t.Fatalf("metadata: %v", result)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("store created by %s: %v", command, err)
			}
		})
	}
}

func TestFlagsPreservePositionsAndTerminator(t *testing.T) {
	pos, options, err := flags([]string{"first", "--scope", "personal", "--dry-run", "second", "--", "--literal"}, []string{"--scope"}, []string{"--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pos, []string{"first", "second", "--literal"}) {
		t.Fatal(pos)
	}
	if !reflect.DeepEqual(options, map[string]string{"--scope": "personal", "--dry-run": "true"}) {
		t.Fatal(options)
	}
}
