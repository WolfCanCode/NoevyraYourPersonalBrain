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
