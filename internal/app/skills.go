package app

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func skills(s *memory.Store, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("use skills find|trust|list")
	}
	sub := args[0]
	args = args[1:]
	switch sub {
	case "list":
		v, err := s.Skills()
		if err != nil {
			return err
		}
		return printJSON(out, v)
	case "trust":
		pos, f, err := flags(args, []string{"--keyword"}, nil)
		if err != nil {
			return err
		}
		if len(pos) != 1 {
			return errors.New("skill reference required")
		}
		if err = s.Trust(pos[0], f["--keyword"]); err != nil {
			return err
		}
		return printJSON(out, map[string]any{"trusted": pos[0], "keyword": f["--keyword"]})
	case "find":
		if len(args) == 0 {
			return errors.New("search term required")
		}
		return findSkills(s, args, out)
	}
	return errors.New("use skills find|trust|list")
}

// findSkills searches the local AAS catalog, falling back to npx.
func findSkills(s *memory.Store, args []string, out io.Writer) error {
	query := strings.Join(args, " ")
	// Silently seed from embedded catalog if not yet populated.
	_, _ = intelligence.SeedEmbeddedCatalog(s)
	seeded, seedErr := intelligence.CatalogSeeded(s)
	if seedErr == nil && seeded {
		return findCatalogSkills(s, query, out)
	}
	return findExternalSkills(query, out)
}

func findCatalogSkills(store *memory.Store, query string, out io.Writer) error {
	caps, searchErr := intelligence.SearchSkillsCatalog(store, query, 10)
	if searchErr != nil {
		return fmt.Errorf("catalog search: %w", searchErr)
	}
	if len(caps) == 0 {
		if isTTY() {
			fmt.Println(styleMuted.Render("  No skills matched " + query + " in the local catalog."))
		} else {
			_, e := fmt.Fprintf(out, "No skills matched %q in the local catalog.\n", query)
			return e
		}
		return nil
	}
	if isTTY() {
		printHeader("brain skills find · " + query)
		fmt.Println()
		for i, c := range caps {
			e := capEntry{Ref: c.Ref, Kind: c.Kind, Name: c.Name, Network: string(c.Network), Reason: c.Reason, InstallCount: c.InstallCount}
			fmt.Printf("  %2d. %s\n%s\n\n", i+1, capOptionLabel(e), capDetailLine(c.Reason))
		}
		fmt.Println(styleMuted.Render("  Use 'brain skills trust <ref> --keyword <term>' to approve."))
		fmt.Println()
	} else {
		fmt.Fprintf(out, "Top %d skills matching %q:\n\n", len(caps), query)
		for i, c := range caps {
			fmt.Fprintf(out, "  %2d. %s\n      %s\n\n", i+1, c.Ref, c.Reason)
		}
		fmt.Fprintf(out, "Use 'brain skills trust <ref> --keyword <term>' to approve a skill.\n")
	}
	return nil
}

func findExternalSkills(query string, out io.Writer) error {
	// Catalog not yet seeded — fall back to npx and cap at 10 results.
	fmt.Fprintln(out, "Local AAS catalog not found. Run 'brain sources seed && brain sources sync' to enable fast offline search.")
	fmt.Fprintln(out, "Falling back to npx…")
	r, err := external("npx", []string{"--yes", "skills", "find", query})
	if err != nil {
		return fmt.Errorf("skills discovery: %w: %s", err, r)
	}
	// Print only first 10 non-empty lines to avoid flooding the terminal.
	lines, shown := strings.Split(r, "\n"), 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if shown >= 10 {
			fmt.Fprintln(out, "  … (use local catalog for full results)")
			break
		}
		fmt.Fprintln(out, " ", line)
		shown++
	}
	return nil
}
