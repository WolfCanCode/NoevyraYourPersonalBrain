package intelligence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func testStore(t testing.TB) *memory.Store {
	t.Helper()
	store, err := memory.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestProfileUsesBoundedProjectEvidence(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/app\n\nrequire modernc.org/sqlite v1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte("name: CI"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}
	profile, err := Profile(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(append(append(profile.Languages, profile.Stacks...), profile.Signals...), " ")
	for _, expected := range []string{"go", "sqlite", "github-actions", "ci-cd"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s in %+v", expected, profile)
		}
	}
}

func TestSourceCompileBudgetAndFeedback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = w.Write([]byte("# Payment webhooks\nUse idempotency keys and verify HMAC signatures before processing retries.\n\n# Database\nStore delivery identifiers with a unique constraint."))
	}))
	defer server.Close()
	store := testStore(t)
	source, err := AddSource(store, "Security guide", server.URL, "security,payment")
	if err != nil {
		t.Fatal(err)
	}
	reports, err := SyncSources(context.Background(), store)
	if err != nil || len(reports) != 1 || reports[0].Error != "" || !reports[0].Changed {
		t.Fatalf("sync: %+v %v", reports, err)
	}
	if _, err = store.Remember(memory.Input{Content: "Payment handlers must be retry-safe", ProjectID: "project"}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err = os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}
	pack, err := Compile(store, memory.Project{ID: "project", Name: "project", Root: root}, "implement secure payment webhook", 128)
	if err != nil {
		t.Fatal(err)
	}
	if pack.Metrics.EstimatedTokens > 128 || len(pack.Context) < 2 || len(pack.Recommendations) == 0 {
		t.Fatalf("pack: %+v", pack)
	}
	result, err := Feedback(store, pack.ID, "helpful", 90, "saved repeated research")
	if err != nil || result["knowledgeUpdated"].(int64) < 1 {
		t.Fatalf("feedback: %+v %v", result, err)
	}
	var quality float64
	if err = store.DB.QueryRow("SELECT quality FROM knowledge_items WHERE source_id=? AND status='active' LIMIT 1", source.ID).Scan(&quality); err != nil || quality <= 0 {
		t.Fatalf("quality: %v %v", quality, err)
	}
	for _, tokens := range []int{160, 240} {
		next, compileErr := Compile(store, memory.Project{ID: "project", Name: "project", Root: root}, "implement secure payment webhook", 512)
		if compileErr != nil {
			t.Fatal(compileErr)
		}
		if _, compileErr = Feedback(store, next.ID, "helpful", tokens, "useful"); compileErr != nil {
			t.Fatal(compileErr)
		}
	}
	adaptive, err := Compile(store, memory.Project{ID: "project", Name: "project", Root: root}, "implement secure payment webhook", 0)
	if err != nil || adaptive.Metrics.BudgetMode != "adaptive" || adaptive.Metrics.BudgetTokens != 256 {
		t.Fatalf("adaptive budget: %+v %v", adaptive.Metrics, err)
	}
}

func TestSourceURLPolicyAndChunking(t *testing.T) {
	store := testStore(t)
	for _, raw := range []string{"http://example.com/guide", "file:///tmp/guide", "https://user:pass@example.com/guide"} {
		if _, err := AddSource(store, "bad", raw, ""); err == nil {
			t.Fatalf("accepted unsafe URL %s", raw)
		}
	}
	parts := chunks("# One\n" + strings.Repeat("use evidence, not assumptions. ", 400))
	if len(parts) < 2 {
		t.Fatalf("expected bounded chunks: %d", len(parts))
	}
}
