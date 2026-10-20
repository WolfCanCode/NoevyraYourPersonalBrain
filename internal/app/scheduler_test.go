package app

import (
	"strings"
	"testing"
)

func TestSchedulerRefreshesTrustedSourcesWithoutExecutingSkills(t *testing.T) {
	for _, platform := range []string{"darwin", "linux"} {
		service, timer, err := schedulerDefinitions(platform, "/opt/brain", "/tmp/brain-home")
		if err != nil {
			t.Fatalf("%s definition: %v", platform, err)
		}
		definition := service + timer
		if !strings.Contains(definition, "sync") || !strings.Contains(definition, "--sources") {
			t.Fatalf("%s schedule does not refresh sources: %s", platform, definition)
		}
		if strings.Contains(definition, "--skills") || strings.Contains(definition, "--online") || strings.Contains(definition, "npx") {
			t.Fatalf("%s schedule may execute third-party skill tooling: %s", platform, definition)
		}
	}
}

func TestSchedulerDefinitionsEscapePaths(t *testing.T) {
	plist, _, err := schedulerDefinitions("darwin", "/Applications/A&B/brain", "/Users/me/A<B")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plist, "/Applications/A&amp;B/brain") || !strings.Contains(plist, "/Users/me/A&lt;B") {
		t.Fatalf("launchd XML paths were not escaped: %s", plist)
	}
	service, _, err := schedulerDefinitions("linux", "/opt/brain path/$brain", "/tmp/brain%home")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(service, `"/opt/brain path/$$brain"`) || !strings.Contains(service, "BRAIN_HOME=/tmp/brain%%home") {
		t.Fatalf("systemd paths were not escaped: %s", service)
	}
}
