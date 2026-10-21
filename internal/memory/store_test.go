package memory

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t testing.TB) *Store {
	t.Helper()
	s, e := Open(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func TestScopeAndApproval(t *testing.T) {
	s := testStore(t)
	for _, p := range []string{"alpha", "beta"} {
		if _, e := s.Remember(Input{Content: "payment HMAC " + p, ProjectID: p}); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := s.Recall("payment", "alpha", 8)
	if e != nil || len(rows) != 1 || *rows[0].ProjectID != "alpha" {
		t.Fatalf("scope: %+v %v", rows, e)
	}
	item, e := s.Propose(Input{Content: "PostgreSQL migration", ProjectID: "alpha"})
	if e != nil {
		t.Fatal(e)
	}
	rows, e = s.Recall("PostgreSQL", "alpha", 8)
	if e != nil || len(rows) != 0 {
		t.Fatalf("candidate leaked: %+v %v", rows, e)
	}
	if ok, e := s.Approve(item.ID); e != nil || !ok {
		t.Fatalf("approve: %v %v", ok, e)
	}
	if ok, e := s.Approve(item.ID); e != nil || ok {
		t.Fatalf("second approval: %v %v", ok, e)
	}
	rows, e = s.Recall("PostgreSQL", "alpha", 8)
	if e != nil || len(rows) != 1 {
		t.Fatalf("approved: %+v %v", rows, e)
	}
	if _, e = s.Propose(Input{Content: "global", Scope: "personal", ProjectID: "alpha"}); e == nil {
		t.Fatal("agent global write allowed")
	}
	if _, e = s.Remember(Input{Content: "TypeScript preference", Scope: "personal"}); e != nil {
		t.Fatal(e)
	}
	rows, e = s.Recall("TypeScript", "other", 8)
	if e != nil || len(rows) != 1 || rows[0].ProjectID != nil {
		t.Fatalf("personal: %+v %v", rows, e)
	}
}
func TestExpiryAndSyncAtomicity(t *testing.T) {
	s := testStore(t)
	expiry := time.Now().Add(-time.Hour).Format(time.RFC3339Nano)
	if _, e := s.Remember(Input{Content: "expired token", ProjectID: "p", ExpiresAt: expiry}); e != nil {
		t.Fatal(e)
	}
	rows, e := s.Recall("token", "p", 8)
	if e != nil || len(rows) != 0 {
		t.Fatalf("expired recall: %+v %v", rows, e)
	}
	if _, e = s.Sync(); e != nil {
		t.Fatal(e)
	}
	stats, e := s.Stats()
	if e != nil || stats["verified"] != 0 || stats["events"] != 1 || stats["lastSync"] == nil {
		t.Fatalf("sync: %+v %v", stats, e)
	}
	if _, e = s.Digest(-1); e == nil {
		t.Fatal("negative days accepted")
	}
}
func TestLegacyMigrationRebuildsMissingIndex(t *testing.T) {
	home := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(home, "brain.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(schema); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`DROP TRIGGER memory_fts_ai; INSERT INTO memory(id,type,scope,project_id,content,source,status,created_at,updated_at) VALUES('old','decision','project','p','legacy architecture','cli:user','verified','2026-01-01T00:00:00.000Z','2026-01-01T00:00:00.000Z')`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	s, e := Open(home)
	if e != nil {
		t.Fatal(e)
	}
	rows, e := s.Recall("architecture", "p", 8)
	if e != nil || len(rows) != 1 {
		t.Fatalf("migration lost recall: %+v %v", rows, e)
	}
	s.Close()
	s, e = Open(home)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.Remember(Input{Content: "new architecture", ProjectID: "p"}); e != nil {
		t.Fatal(e)
	}
	rows, e = s.Recall("architecture", "p", 8)
	if e != nil || len(rows) != 2 {
		t.Fatalf("reopened index: %+v %v", rows, e)
	}
}
func TestFTSSyntaxAndUnicode(t *testing.T) {
	s := testStore(t)
	if _, e := s.Remember(Input{Content: "TypeScript: quyết định dùng bộ nhớ", ProjectID: "p"}); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{`TypeScript OR "*`, "quyết", "TypeScript - SQL", "!!!"} {
		if _, e := s.Recall(q, "p", 8); e != nil {
			t.Fatalf("%q: %v", q, e)
		}
	}
}
func BenchmarkRecall(b *testing.B) {
	s := testStore(b)
	for i := 0; i < 1000; i++ {
		if _, e := s.Remember(Input{Content: "payment webhook architecture uses idempotency HMAC", ProjectID: "p"}); e != nil {
			b.Fatal(e)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, e := s.Recall("payment HMAC", "p", 8); e != nil {
			b.Fatal(e)
		}
	}
}

// ─── Tasks 1.4 & 1.5: project_profiles persistence tests ──────────────────

// Feature: auto-project-setup, Property 2: Profile store count never exceeds three per project.
// Validates: Requirement 2.3
func TestUpsertProfileNeverExceedsThree(t *testing.T) {
	// Property test: for any project ID and N ≥ 1 UpsertProfile calls,
	// the row count for that project never exceeds 3.
	s := testStore(t)
	const projID = "prop-test-project"
	for n := 1; n <= 10; n++ {
		if _, err := s.UpsertProfile(projID, "/root", `{"root":"/root"}`, "[]"); err != nil {
			t.Fatalf("UpsertProfile call %d: %v", n, err)
		}
		var count int
		if err := s.DB.QueryRow("SELECT count(*) FROM project_profiles WHERE project_id=?", projID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count > 3 {
			t.Fatalf("after %d calls count=%d, want ≤3", n, count)
		}
	}
}

// Validates: Requirements 2.1, 2.3, 2.4, 2.8
func TestUpsertProfileAndLatest(t *testing.T) {
	s := testStore(t)
	const projID = "upsert-test"
	roots := []string{"/a", "/b", "/c"}
	for _, root := range roots {
		if _, err := s.UpsertProfile(projID, root, `{"root":"`+root+`"}`, "[]"); err != nil {
			t.Fatalf("UpsertProfile(%q): %v", root, err)
		}
	}
	latest, err := s.LatestProfile(projID)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil {
		t.Fatal("expected non-nil StoredProfile")
	}
	if latest.ProjectID != projID {
		t.Errorf("ProjectID=%q, want %q", latest.ProjectID, projID)
	}
	if latest.Root != "/c" {
		t.Errorf("Root=%q, want /c (the most recent)", latest.Root)
	}
}

func TestUpsertProfileEvictsOldest(t *testing.T) {
	s := testStore(t)
	const projID = "evict-test"
	var firstID string
	for i := 0; i < 4; i++ {
		sp, err := s.UpsertProfile(projID, "/root", `{}`, "[]")
		if err != nil {
			t.Fatalf("UpsertProfile %d: %v", i, err)
		}
		if i == 0 {
			firstID = sp.ID
		}
		// Ensure distinct created_at by inserting a tiny sleep
		time.Sleep(2 * time.Millisecond)
	}
	var count int
	if err := s.DB.QueryRow("SELECT count(*) FROM project_profiles WHERE project_id=?", projID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count=%d after 4 inserts, want 3", count)
	}
	var exists int
	if err := s.DB.QueryRow("SELECT count(*) FROM project_profiles WHERE id=?", firstID).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists != 0 {
		t.Error("first (oldest) profile was not evicted")
	}
}

func TestLatestProfileNilWhenMissing(t *testing.T) {
	s := testStore(t)
	p, err := s.LatestProfile("nonexistent-project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil, got %+v", p)
	}
}

func TestSchemaV3Migration(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	// Reopen to confirm table still accessible
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	var n int
	if err := s2.DB.QueryRow("SELECT count(*) FROM project_profiles").Scan(&n); err != nil {
		t.Fatalf("project_profiles not accessible after reopen: %v", err)
	}
}

func TestVersionGuardV3(t *testing.T) {
	s := testStore(t)
	path := s.Path
	s.Close()
	// Manually set user_version to 4 (future schema).
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("PRAGMA user_version=4"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	_, openErr := Open(filepath.Dir(path))
	if openErr == nil {
		t.Fatal("expected error opening newer schema, got nil")
	}
}
