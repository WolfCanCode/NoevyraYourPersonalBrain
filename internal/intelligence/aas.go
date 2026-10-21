package intelligence

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// SearchSkillsCatalog searches knowledge items from locally cached skill catalog
// sources (sources tagged "skills" or "catalog") and returns at most limit
// ProvisionableCapability values ranked by FTS5 relevance.
//
// It is the backend for both brain setup skill discovery and brain skills find.
// An empty result means the catalog has not been seeded yet; prompt the user to
// run: brain sources seed && brain sources sync
func SearchSkillsCatalog(store *memory.Store, query string, limit int) ([]ProvisionableCapability, error) {
	if limit <= 0 {
		limit = 10
	}
	tokens := catalogTokens(query)
	if len(tokens) == 0 {
		return nil, nil
	}
	matchExpr := strings.Join(tokens, " OR ")
	rows, err := store.DB.Query(`
		SELECT k.id, k.title, k.content
		FROM knowledge_fts
		JOIN knowledge_items k ON k.rowid = knowledge_fts.rowid
		JOIN knowledge_sources s ON s.id = k.source_id
		WHERE knowledge_fts MATCH ?
		  AND k.status  = 'active'
		  AND s.status  = 'active'
		  AND (s.tags LIKE '%skills%' OR s.tags LIKE '%catalog%')
		ORDER BY bm25(knowledge_fts)
		LIMIT ?`, matchExpr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	caps := []ProvisionableCapability{}
	for rows.Next() {
		var id, title, content string
		if err = rows.Scan(&id, &title, &content); err != nil {
			return nil, err
		}
		// Construct an owner/repo@skill ref so the existing trust store can store it.
		ref := "sickn33/agentic-awesome-skills@" + strings.ToLower(strings.ReplaceAll(title, " ", "-"))
		caps = append(caps, ProvisionableCapability{
			Kind:           "skill",
			Ref:            ref,
			Name:           title,
			Reason:         descriptionLine(content),
			Description:    content,
			Confidence:     0.72,
			Network:        NetworkNetwork,
			TriggerSignals: []string{},
			SkillKeyword:   title,
		})
	}
	return caps, rows.Err()
}

// CatalogSeeded returns true if at least one skill knowledge item exists.
// Used to decide whether to fall back to npx skills find.
func CatalogSeeded(store *memory.Store) (bool, error) {
	var n int
	err := store.DB.QueryRow(`
		SELECT count(*) FROM knowledge_items k
		JOIN knowledge_sources s ON s.id = k.source_id
		WHERE k.status = 'active' AND s.status = 'active'
		  AND (s.tags LIKE '%skills%' OR s.tags LIKE '%catalog%')`).Scan(&n)
	return n > 0, err
}

var catalogWordRe = regexp.MustCompile(`[\p{L}\p{N}_-]+`)

// catalogTokens tokenises a free-form query into FTS5 quoted terms.
func catalogTokens(query string) []string {
	tokens := []string{}
	seen := map[string]bool{}
	for _, word := range catalogWordRe.FindAllString(strings.ToLower(query), -1) {
		if utf8.RuneCountInString(word) < 2 || seen[word] || len(tokens) >= 8 {
			continue
		}
		seen[word] = true
		tokens = append(tokens, `"`+strings.ReplaceAll(word, `"`, "")+`"`)
	}
	return tokens
}

// descriptionLine returns the first non-empty line of a skill content block.
func descriptionLine(content string) string {
	for _, line := range strings.SplitN(content, "\n", 5) {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return content
}
