package intelligence

import (
	_ "embed"
	"fmt"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// embeddedSkillsIndex is the AAS skills catalog bundled at build time.
// Updated automatically during CI builds from:
// https://raw.githubusercontent.com/sickn33/agentic-awesome-skills/main/skills_index.json
//
//go:embed skills_index.json
var embeddedSkillsIndex []byte

// minExpectedSkills is the minimum number of catalog items expected after a
// correct seed. If fewer exist the seed is considered malformed and re-runs.
// One giant "Overview" chunk from a failed JSON parse would be caught here.
const minExpectedSkills = 2000 // 2,737 skills expected; anything less triggers re-seed

// SeedEmbeddedCatalog imports the bundled AAS skills_index.json into the local
// SQLite store when no catalog entries exist, or when the existing count is
// suspiciously low (indicating the JSON chunker failed and the markdown chunker
// created one giant "Overview" chunk instead of individual skill entries).
//
// It registers the upstream URL so that brain sources sync can later refresh.
// Safe to call on every startup — cheap when already properly seeded.
func SeedEmbeddedCatalog(store *memory.Store) (int, error) {
	var count int
	err := store.DB.QueryRow(`
		SELECT count(*) FROM knowledge_items k
		JOIN knowledge_sources s ON s.id = k.source_id
		WHERE k.status = 'active' AND s.status = 'active'
		  AND (s.tags LIKE '%skills%' OR s.tags LIKE '%catalog%')`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("catalog count: %w", err)
	}
	if count >= minExpectedSkills {
		return 0, nil // already properly seeded — nothing to do
	}

	// Register (or re-register) the upstream source.
	source, err := AddSource(store,
		"AAS skills catalog",
		"https://raw.githubusercontent.com/sickn33/agentic-awesome-skills/main/skills_index.json",
		"skills,catalog,aas")
	if err != nil {
		return 0, fmt.Errorf("register catalog source: %w", err)
	}

	// Supersede any existing malformed chunks before re-importing.
	if count > 0 {
		_, _ = store.DB.Exec(
			`UPDATE knowledge_items SET status='superseded' WHERE source_id=?`, source.ID)
	}

	// Import through saveSource — chunksFromSkillsIndex handles the JSON array.
	n, _, err := saveSource(store, source, embeddedSkillsIndex, "", "")
	if err != nil {
		return 0, fmt.Errorf("import embedded catalog: %w", err)
	}

	// Optimise the FTS5 index after a bulk load so subsequent searches are fast.
	_, _ = store.DB.Exec(`INSERT INTO knowledge_fts(knowledge_fts) VALUES('optimize')`)

	return n, nil
}
