package intelligence

import (
	"strings"
	"testing"
)

func TestSelectContextCandidatesPreservesBudgetAndDeduplication(t *testing.T) {
	candidates := []ContextItem{
		{ID: "first", Content: strings.Repeat("a", 360), EstimatedTokens: 90},
		{ID: "duplicate", Content: strings.Repeat("A", 360), EstimatedTokens: 90},
		{ID: "stopped", Content: "unused", EstimatedTokens: 2},
		{ID: "unvisited-duplicate", Content: strings.Repeat("a", 360), EstimatedTokens: 90},
	}
	selected, used, duplicates := selectContextCandidates(candidates, 128)
	if len(selected) != 1 || selected[0].ID != "first" || used != 90 || duplicates != 1 {
		t.Fatalf("selection=%+v used=%d duplicates=%d", selected, used, duplicates)
	}
}

func TestSelectContextCandidatesTruncatesContentAndSkipsEmpty(t *testing.T) {
	selected, used, duplicates := selectContextCandidates([]ContextItem{
		{ID: "empty"},
		{ID: "large", Title: "heading", Content: strings.Repeat("界", 700), EstimatedTokens: 180},
	}, 128)
	if len(selected) != 1 || selected[0].ID != "large" || used != 128 || duplicates != 0 || !strings.HasSuffix(selected[0].Content, "…") {
		t.Fatalf("selection=%+v used=%d duplicates=%d", selected, used, duplicates)
	}
}
