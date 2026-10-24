package app

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

type skillGroup struct {
	label string
	caps  []intelligence.ProvisionableCapability
}

var keywordPriority = []string{
	"angular", "react", "vue", "svelte", "nextjs",
	"graphql", "storybook", "tailwind", "redux",
	"spring", "nestjs", "gin", "fiber", "echo",
	"django", "fastapi", "flask", "pytorch", "tensorflow",
	"rails", "laravel", "symfony",
	"java", "kotlin", "scala",
	"go", "rust", "python", "ruby", "php", "swift",
	"typescript", "javascript",
	"api", "database", "security", "containers",
}

func setupKeywords(profile intelligence.ProjectProfile) []string {
	sigSet := map[string]bool{}
	for _, v := range profile.Signals {
		sigSet[strings.ToLower(v)] = true
	}
	for _, v := range profile.Languages {
		sigSet[strings.ToLower(v)] = true
	}
	for _, v := range profile.Stacks {
		sigSet[strings.ToLower(v)] = true
	}
	excludeIfPresent := map[string][]string{
		"java":       {"javascript", "typescript"},
		"kotlin":     {"javascript", "typescript"},
		"scala":      {"javascript", "typescript"},
		"python":     {"javascript", "typescript"},
		"ruby":       {"javascript", "typescript"},
		"php":        {"javascript", "typescript"},
		"typescript": {"javascript"},
	}
	result := []string{}
	excluded := map[string]bool{}
	for _, kw := range keywordPriority {
		if excluded[kw] || !sigSet[kw] {
			continue
		}
		result = append(result, kw)
		for _, ex := range excludeIfPresent[kw] {
			excluded[ex] = true
		}
	}
	return result
}

const skillsPerKeyword = 8

func searchCatalogSkills(profile intelligence.ProjectProfile, store *memory.Store, seen map[string]bool, out, errOut io.Writer, skipLive bool) []skillGroup {
	keywords := setupKeywords(profile)
	if len(keywords) == 0 {
		return nil
	}
	seeded, err := intelligence.CatalogSeeded(store)
	if err != nil {
		fmt.Fprintf(errOut, "brain: catalog check failed: %v\n", err)
	}
	if seeded {
		return searchEmbeddedSkills(store, keywords, seen, errOut)
	}
	if isTTY() {
		fmt.Println(styleMuted.Render("  ⚠  AAS skill catalog not found."))
		fmt.Println(styleMuted.Render("     Run 'brain sources seed && brain sources sync' to index 2 700+ skills."))
	} else {
		fmt.Fprintln(out, "  AAS skill catalog not found locally.")
		fmt.Fprintln(out, "  Run 'brain sources seed && brain sources sync' to index 2 700+ skills for offline search.")
	}
	if skipLive {
		return nil
	}
	if _, err := exec.LookPath("npx"); err != nil {
		return nil
	}
	return searchLiveSkills(keywords[:min(len(keywords), 3)], seen, out, errOut)
}

func searchEmbeddedSkills(store *memory.Store, keywords []string, seen map[string]bool, errOut io.Writer) []skillGroup {
	groups := []skillGroup{}
	for _, keyword := range keywords {
		results, err := intelligence.SearchSkillsCatalog(store, keyword, skillsPerKeyword)
		if err != nil {
			fmt.Fprintf(errOut, "brain: catalog search for %q: %v\n", keyword, err)
			continue
		}
		caps := unseenSkills(results, seen)
		if len(caps) == 0 {
			continue
		}
		groups = append(groups, skillGroup{label: fmt.Sprintf("From AAS catalog — %s (top %d)", keyword, len(caps)), caps: caps})
	}
	return groups
}

func searchLiveSkills(keywords []string, seen map[string]bool, out, errOut io.Writer) []skillGroup {
	groups := []skillGroup{}
	for _, keyword := range keywords {
		if isTTY() {
			fmt.Printf("  %s Searching skills.sh for %s…\n", styleMuted.Render("→"), styleAccent.Render(keyword))
		} else {
			fmt.Fprintf(out, "  Searching skills.sh for %q (npx fallback)…\n", keyword)
		}
		raw, err := external("npx", []string{"--yes", "skills", "find", keyword})
		if err != nil {
			fmt.Fprintf(errOut, "brain: npx skills find %q: %v\n", keyword, err)
			continue
		}
		results := parseSkillOutput(raw, keyword)
		results = results[:min(len(results), skillsPerKeyword)]
		caps := unseenSkills(results, seen)
		if len(caps) == 0 {
			continue
		}
		groups = append(groups, skillGroup{label: fmt.Sprintf("From skills.sh — %s (top %d)", keyword, len(caps)), caps: caps})
	}
	return groups
}

func unseenSkills(candidates []intelligence.ProvisionableCapability, seen map[string]bool) []intelligence.ProvisionableCapability {
	result := make([]intelligence.ProvisionableCapability, 0, len(candidates))
	for _, candidate := range candidates {
		key := "skill:" + candidate.Ref
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}

var (
	skillRefRe     = regexp.MustCompile(`[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+`)
	ansiEscape     = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	installCountRe = regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?[KMB]?)\s+installs?`)
)

func parseSkillOutput(raw, keyword string) []intelligence.ProvisionableCapability {
	raw = ansiEscape.ReplaceAllString(raw, "")
	lines := strings.Split(raw, "\n")
	seen := map[string]bool{}
	caps := []intelligence.ProvisionableCapability{}

	for i, line := range lines {
		ref := skillRefRe.FindString(line)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true

		desc := strings.TrimSpace(skillRefRe.ReplaceAllString(line, ""))
		desc = strings.Trim(desc, "-–—:\"'|·•\t ")

		// Capture install count from the next 1-2 lines
		installCount := ""
		for j := i + 1; j <= i+2 && j < len(lines); j++ {
			if m := installCountRe.FindStringSubmatch(lines[j]); len(m) >= 2 {
				installCount = m[1]
				break
			}
		}

		if desc == "" {
			desc = "Community skill for " + keyword
		}
		skillKw := keyword
		if parts := strings.SplitN(ref, "@", 2); len(parts) == 2 {
			skillKw = strings.ReplaceAll(parts[1], "-", " ")
		}
		caps = append(caps, intelligence.ProvisionableCapability{
			Kind:           "skill",
			Ref:            ref,
			Name:           ref,
			Reason:         desc,
			Description:    desc,
			Confidence:     0.72,
			Network:        intelligence.NetworkNetwork,
			TriggerSignals: []string{keyword},
			SkillKeyword:   skillKw,
			InstallCount:   installCount,
		})
	}
	return caps
}
