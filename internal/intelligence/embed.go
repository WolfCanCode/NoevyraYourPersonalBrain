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

// SeedEmbeddedCatalog imports the bundled AAS skills_index.json into the local
// SQLite store if no skill catalog entries exist yet. It is safe to call on
// every startup — it exits immediately when the catalog is already populated.
//
// It registers the live upstream URL as the source so that brain sources sync
// can later refresh from the internet when the user wants fresher data.
func SeedEmbeddedCatalog(store *memory.Store) (int, error) {
	seeded, err := CatalogSeeded(store)
	if err != nil {
		return 0, fmt.Errorf("catalog check: %w", err)
	}
	if seeded {
		return 0, nil // already populated — nothing to do
	}

	// Register the upstream URL so brain sources sync can keep it fresh.
	source, err := AddSource(store,
		"AAS skills catalog",
		"https://raw.githubusercontent.com/sickn33/agentic-awesome-skills/main/skills_index.json",
		"skills,catalog,aas")
	if err != nil {
		return 0, fmt.Errorf("register catalog source: %w", err)
	}

	// Import the embedded bytes through the existing saveSource pipeline.
	// chunksFromSkillsIndex handles the JSON array format automatically.
	n, _, err := saveSource(store, source, embeddedSkillsIndex, "", "")
	if err != nil {
		return 0, fmt.Errorf("import embedded catalog: %w", err)
	}
	return n, nil
}
