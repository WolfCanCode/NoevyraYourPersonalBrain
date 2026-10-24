package memory

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsFutureSchemaWithoutModifyingData(t *testing.T) {
	home := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(home, "brain.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE marker(value TEXT); INSERT INTO marker VALUES('keep'); PRAGMA user_version=99"); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(home); err == nil || !strings.Contains(err.Error(), "newer than supported") {
		t.Fatalf("future schema: %v", err)
	}
	var value string
	if err := db.QueryRow("SELECT value FROM marker").Scan(&value); err != nil || value != "keep" {
		t.Fatalf("data modified: %q %v", value, err)
	}
	var count int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='memory'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("schema changed: %d %v", count, err)
	}
}

func TestDigestIncludesEventsAndMemories(t *testing.T) {
	s := testStore(t)
	if _, err := s.Remember(Input{Content: "payment architecture", ProjectID: "project"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Log("handoff", "project", map[string]string{"objective": "finish API"}); err != nil {
		t.Fatal(err)
	}
	digest, err := s.Digest(1)
	if err != nil {
		t.Fatal(err)
	}
	events := digest["events"].([]Event)
	memories := digest["memories"].([]Memory)
	if len(events) != 1 || len(memories) != 1 {
		t.Fatalf("digest: %+v", digest)
	}
	var payload map[string]string
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil || payload["objective"] != "finish API" {
		t.Fatalf("payload: %+v %v", payload, err)
	}
	pack, err := s.Context("payment", "project", 8)
	if err != nil || len(pack["memories"].([]Memory)) != 1 {
		t.Fatalf("context: %+v %v", pack, err)
	}
}

func TestTrustedSkillsUpdateAndSort(t *testing.T) {
	s := testStore(t)
	for ref, keyword := range map[string]string{"owner/repo@one": "zebra", "owner/repo@two": "alpha"} {
		if err := s.Trust(ref, keyword); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Trust("owner/repo@one", "beta"); err != nil {
		t.Fatal(err)
	}
	items, err := s.Skills()
	if err != nil || len(items) != 2 {
		t.Fatalf("skills: %+v %v", items, err)
	}
	if items[0].Keyword != "alpha" || items[1].Keyword != "beta" {
		t.Fatalf("order/update: %+v", items)
	}
}
