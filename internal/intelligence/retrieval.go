package intelligence

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func searchTokens(primary, secondary string, limit int) []string {
	tokens := []string{}
	seen := map[string]bool{}
	appendTerms := func(value string) {
		for _, term := range termPattern.FindAllString(strings.ToLower(value), -1) {
			if utf8.RuneCountInString(term) < 2 || seen[term] || len(tokens) >= limit {
				continue
			}
			seen[term] = true
			tokens = append(tokens, `"`+strings.ReplaceAll(term, `"`, ``)+`"`)
		}
	}
	// Task terms must survive the cap; project metadata only broadens retrieval.
	appendTerms(primary)
	appendTerms(secondary)
	return tokens
}

func knowledgeCandidates(store *memory.Store, task, projectSignals string, limit int) ([]ContextItem, error) {
	tokens := searchTokens(task, projectSignals, 20)
	if len(tokens) == 0 {
		return []ContextItem{}, nil
	}
	rows, err := store.DB.Query(`SELECT k.id,k.title,k.content,s.url,k.quality,bm25(knowledge_fts)
		FROM knowledge_fts JOIN knowledge_items k ON k.rowid=knowledge_fts.rowid
		JOIN knowledge_sources s ON s.id=k.source_id
		WHERE knowledge_fts MATCH ? AND k.status='active' AND s.status='active'
		ORDER BY bm25(knowledge_fts),k.quality DESC,k.updated_at DESC LIMIT ?`, strings.Join(tokens, " OR "), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ContextItem{}
	position := 0
	for rows.Next() {
		var item ContextItem
		var quality, rank float64
		if err = rows.Scan(&item.ID, &item.Title, &item.Content, &item.Source, &quality, &rank); err != nil {
			return nil, err
		}
		item.Kind = "knowledge"
		item.Score = 80 - float64(position) + quality*10 - rank
		item.EstimatedTokens = estimateTokens(item.Title + " " + item.Content)
		items = append(items, item)
		position++
	}
	return items, rows.Err()
}

func contextCandidates(store *memory.Store, projectID, task string, profile ProjectProfile) ([]ContextItem, error) {
	projectParts := append([]string{}, profile.Languages...)
	projectParts = append(projectParts, profile.Stacks...)
	projectParts = append(projectParts, profile.Signals...)
	projectSignals := strings.Join(projectParts, " ")
	query := strings.TrimSpace(task + " " + projectSignals)
	memories, err := store.Recall(query, projectID, 24)
	if err != nil {
		return nil, err
	}
	candidates := make([]ContextItem, 0, len(memories)+32)
	for index, item := range memories {
		candidates = append(candidates, ContextItem{Kind: "memory", ID: item.ID, Content: item.Content, Source: item.Source, Score: 100 - float64(index), EstimatedTokens: estimateTokens(item.Content)})
	}
	knowledge, err := knowledgeCandidates(store, task, projectSignals, 40)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, knowledge...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return candidates, nil
}
