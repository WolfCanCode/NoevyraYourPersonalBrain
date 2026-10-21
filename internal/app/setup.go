package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// skillGroup holds skills from a single keyword search, displayed as one section.
type skillGroup struct {
	label string
	caps  []intelligence.ProvisionableCapability
}

// projectRoot walks ancestor directories from cwd looking for a .git folder.
// It checks cwd first, then up to 10 parent directories.
// Returns cwd unchanged if no .git is found within that bound (Req 1.1, 1.7).
func projectRoot(cwd string) string {
	dir := cwd
	for i := 0; i <= 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

// displayProfile writes detected languages and signals to out.
// Stacks are truncated to 5 entries to avoid the wall-of-text problem
// when package.json has hundreds of dependencies.
func displayProfile(out io.Writer, profile intelligence.ProjectProfile) {
	fmt.Fprintln(out, "Detected profile:")
	if len(profile.Languages) > 0 {
		fmt.Fprintln(out, "  Languages:", strings.Join(profile.Languages, ", "))
	}
	if len(profile.Stacks) > 0 {
		shown := profile.Stacks
		extra := 0
		if len(shown) > 5 {
			extra = len(shown) - 5
			shown = shown[:5]
		}
		line := strings.Join(shown, ", ")
		if extra > 0 {
			line += fmt.Sprintf(" … +%d more", extra)
		}
		fmt.Fprintln(out, "  Stacks:   ", line)
	}
	if len(profile.Signals) > 0 {
		fmt.Fprintln(out, "  Signals:  ", strings.Join(profile.Signals, ", "))
	}
	if profile.Truncated {
		fmt.Fprintln(out, "  (scan truncated at 4000 files)")
	}
}

// promptConfirm reads a y/N answer. Returns true only for "y" or "Y".
func promptConfirm(in io.Reader, out io.Writer, msg string) (bool, error) {
	fmt.Fprintf(out, "%s [y/N]: ", msg)
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return line == "y" || line == "Y", nil
	}
	return false, scanner.Err()
}

// promptSelection prints a numbered capability list grouped by source and
// reads the user's choice: "all", "none", or comma-separated indices.
// catalog contains built-in MCP/skill entries; liveGroups contains per-keyword
// sections returned by searchCatalogSkills.
func promptSelection(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup) ([]intelligence.ProvisionableCapability, error) {
	all := make([]intelligence.ProvisionableCapability, 0, len(catalog)+10)
	all = append(all, catalog...)
	for _, g := range liveGroups {
		all = append(all, g.caps...)
	}
	if len(all) == 0 {
		return nil, nil
	}

	fmt.Fprintln(out, "\nAvailable capabilities:")
	if len(catalog) > 0 {
		fmt.Fprintln(out, "\n  Built-in recommendations:")
		for i, cap := range catalog {
			fmt.Fprintf(out, "    %2d. [%s][%s] %s\n        %s\n", i+1, cap.Kind, string(cap.Network), cap.Name, cap.Reason)
		}
	}
	offset := len(catalog)
	for _, g := range liveGroups {
		if len(g.caps) == 0 {
			continue
		}
		fmt.Fprintf(out, "\n  %s:\n", g.label)
		for i, cap := range g.caps {
			fmt.Fprintf(out, "    %2d. [skill][network] %s\n        %s\n", offset+i+1, cap.Name, cap.Reason)
		}
		offset += len(g.caps)
	}

	fmt.Fprint(out, "\nSelect capabilities to provision (all/none/1,2,3): ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}
	line := strings.TrimSpace(scanner.Text())
	if line == "" || line == "none" {
		return nil, nil
	}
	if line == "all" {
		return all, nil
	}
	selected := []intelligence.ProvisionableCapability{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil || idx < 1 || idx > len(all) {
			fmt.Fprintf(out, "  Skipping invalid selection: %q\n", part)
			continue
		}
		selected = append(selected, all[idx-1])
	}
	return selected, nil
}

// printProvisionSummary writes an aligned table of capability name → outcome.
func printProvisionSummary(out io.Writer, results []intelligence.ProvisionResult) {
	fmt.Fprintln(out, "\nProvisioning summary:")
	for _, r := range results {
		status := string(r.Outcome)
		if r.Error != "" {
			status += " (" + r.Error + ")"
		}
		fmt.Fprintf(out, "  %-36s %s\n", r.Capability.Name, status)
	}
}

// buildSessionEvent builds the setup_session event payload.
func buildSessionEvent(results []intelligence.ProvisionResult) map[string]any {
	names := make([]string, len(results))
	provisioned, total := 0, len(results)
	for i, r := range results {
		names[i] = r.Capability.Name
		if r.Outcome == intelligence.OutcomeProvisioned {
			provisioned++
		}
	}
	outcome := "failed"
	if provisioned == total && total > 0 {
		outcome = "completed"
	} else if provisioned > 0 {
		outcome = "partial"
	}
	return map[string]any{"capabilities": names, "outcome": outcome}
}

// ── Live skill search via skills.sh ─────────────────────────────────────────

// setupKeywords returns up to 3 search keywords for the project, ordered from
// most specific (framework) to most generic (language, then operational signal).
// It checks Signals + Languages + Stacks — but only returns terms that appear
// in keywordPriority, so raw dep names (zone.js, karma, ...) never leak through.
var keywordPriority = []string{
	// JS/TS framework signals (injected into Signals by profile.go)
	"angular", "react", "vue", "svelte", "nextjs",
	"graphql", "storybook", "tailwind", "redux",
	// JVM / server-side framework names (appear in Stacks from pom.xml / build.gradle)
	"spring", "nestjs", "gin", "fiber", "echo",
	// Python framework names (appear in Stacks from requirements.txt / pyproject.toml)
	"django", "fastapi", "flask", "pytorch", "tensorflow",
	// Ruby / PHP / .NET
	"rails", "laravel", "symfony",
	// Languages — specific ones first so java beats javascript for JVM projects
	"java", "kotlin", "scala",
	"go", "rust", "python", "ruby", "php", "swift",
	// Generic web languages (last resort for pure JS/TS projects)
	"typescript", "javascript",
	// Operational signals
	"api", "database", "security", "containers",
}

func setupKeywords(profile intelligence.ProjectProfile) []string {
	// Build a term set from ALL three profile fields.
	// keywordPriority acts as an allowlist so raw dep names never leak through.
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
	// mutualExclusions: if a more specific language/stack is already chosen,
	// skip generic alternatives that would produce irrelevant results.
	// e.g. a Java project with a few .js build scripts should not search "javascript".
	excludeIfPresent := map[string][]string{
		"java":       {"javascript", "typescript"},
		"kotlin":     {"javascript", "typescript"},
		"scala":      {"javascript", "typescript"},
		"python":     {"javascript", "typescript"},
		"ruby":       {"javascript", "typescript"},
		"php":        {"javascript", "typescript"},
		"typescript": {"javascript"},
	}
	// Collect ALL matched keywords in priority order — no hard cap.
	// The caller decides how many to use (catalog uses all; npx fallback limits to 3).
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

// skillRefRe matches owner/repo@skill-name patterns in skills.sh output.
var skillRefRe = regexp.MustCompile(`[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+`)

// ansiEscape matches ANSI terminal color/style sequences (e.g. \x1b[145m).
// npx skills find colorises its output; these must be stripped before parsing
// so that color codes like "145m" do not bleed into owner/repo names.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// parseSkillOutput extracts ProvisionableCapabilities from `npx skills find` output.
func parseSkillOutput(raw, keyword string) []intelligence.ProvisionableCapability {
	raw = ansiEscape.ReplaceAllString(raw, "") // strip terminal color codes
	seen := map[string]bool{}
	caps := []intelligence.ProvisionableCapability{}
	for _, line := range strings.Split(raw, "\n") {
		ref := skillRefRe.FindString(line)
		if ref == "" || seen[ref] {
			continue
		}
		seen[ref] = true
		// Extract description: the line without the ref itself.
		desc := strings.TrimSpace(skillRefRe.ReplaceAllString(line, ""))
		desc = strings.Trim(desc, "-–—:\"'|·•·\t ")
		if desc == "" {
			desc = "Community skill for " + keyword
		}
		// Use the part after @ as the trust keyword.
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
		})
	}
	return caps
}

// searchCatalogSkills queries the locally cached AAS skills catalog per keyword,
// returning at most 5 skills per keyword as separate labeled groups.
// If the catalog has not been seeded it falls back to npx skills find
// (also capped at 5 per keyword) and prints a seed hint.
func searchCatalogSkills(profile intelligence.ProjectProfile, store *memory.Store, seen map[string]bool, out, errOut io.Writer, skipLive bool) []skillGroup {
	const perKeyword = 5
	keywords := setupKeywords(profile)
	if len(keywords) == 0 {
		return nil
	}

	seeded, err := intelligence.CatalogSeeded(store)
	if err != nil {
		fmt.Fprintf(errOut, "brain: catalog check failed: %v\n", err)
	}

	if seeded {
		groups := []skillGroup{}
		for _, kw := range keywords {
			results, searchErr := intelligence.SearchSkillsCatalog(store, kw, perKeyword)
			if searchErr != nil {
				fmt.Fprintf(errOut, "brain: catalog search for %q: %v\n", kw, searchErr)
				continue
			}
			caps := []intelligence.ProvisionableCapability{}
			for _, c := range results {
				key := "skill:" + c.Ref
				if seen[key] {
					continue
				}
				seen[key] = true
				caps = append(caps, c)
			}
			if len(caps) > 0 {
				groups = append(groups, skillGroup{
					label: fmt.Sprintf("From AAS catalog — %s (top %d)", kw, len(caps)),
					caps:  caps,
				})
			}
		}
		return groups
	}

	// Catalog not seeded — hint and optional npx fallback
	fmt.Fprintln(out, "  AAS skill catalog not found locally.")
	fmt.Fprintln(out, "  Run 'brain sources seed && brain sources sync' to index 2 700+ skills for offline search.")
	if skipLive {
		return nil
	}
	if _, lookErr := exec.LookPath("npx"); lookErr != nil {
		return nil
	}
	// Limit npx to 3 keywords — each is a separate network call
	npxKeywords := keywords
	if len(npxKeywords) > 3 {
		npxKeywords = npxKeywords[:3]
	}
	groups := []skillGroup{}
	for _, kw := range npxKeywords {
		fmt.Fprintf(out, "  Searching skills.sh for %q (npx fallback)…\n", kw)
		raw, npxErr := external("npx", []string{"--yes", "skills", "find", kw})
		if npxErr != nil {
			fmt.Fprintf(errOut, "brain: npx skills find %q: %v\n", kw, npxErr)
			continue
		}
		all := parseSkillOutput(raw, kw)
		if len(all) > perKeyword {
			all = all[:perKeyword]
		}
		caps := []intelligence.ProvisionableCapability{}
		for _, c := range all {
			key := "skill:" + c.Ref
			if seen[key] {
				continue
			}
			seen[key] = true
			caps = append(caps, c)
		}
		if len(caps) > 0 {
			groups = append(groups, skillGroup{
				label: fmt.Sprintf("From skills.sh — %s (top %d)", kw, len(caps)),
				caps:  caps,
			})
		}
	}
	return groups
}

// ── Main setup handler ───────────────────────────────────────────────────────

// runSetup is the handler for "brain setup".
// Flags: --refresh, --dry-run, --status, --no-live-skills
func runSetup(store *memory.Store, project memory.Project, args []string, in io.Reader, out, errOut io.Writer, home string) error {
	pos, f, err := flags(args, nil, []string{"--refresh", "--dry-run", "--status", "--no-live-skills"})
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("setup accepts no positional arguments; got %v", pos)
	}
	dryRun := f["--dry-run"] != ""
	skipLive := f["--no-live-skills"] != ""

	// --status path
	if f["--status"] != "" {
		stored, err := store.LatestProfile(project.ID)
		if err != nil {
			return err
		}
		if stored == nil {
			fmt.Fprintln(out, "No profile found for the current project. Run 'brain setup' to create one.")
			return nil
		}
		fmt.Fprintln(out, "Stored profile for project:", stored.ProjectID)
		fmt.Fprintln(out, "Root:      ", stored.Root)
		fmt.Fprintln(out, "Created:   ", stored.CreatedAt)
		var profile intelligence.ProjectProfile
		if json.Unmarshal([]byte(stored.ProfileJSON), &profile) == nil {
			displayProfile(out, profile)
		}
		if stored.CapabilitiesJSON != "" && stored.CapabilitiesJSON != "[]" {
			var caps []intelligence.ProvisionResult
			if json.Unmarshal([]byte(stored.CapabilitiesJSON), &caps) == nil && len(caps) > 0 {
				fmt.Fprintln(out, "  Provisioned capabilities:")
				for _, c := range caps {
					if c.Outcome == intelligence.OutcomeProvisioned {
						fmt.Fprintln(out, "    -", c.Capability.Name)
					}
				}
			}
		}
		return nil
	}

	// Detection phase
	root := projectRoot(project.Root)
	profile, err := intelligence.Profile(root)
	if err != nil {
		return err
	}
	displayProfile(out, profile)

	// Overwrite check
	if f["--refresh"] == "" {
		stored, err := store.LatestProfile(project.ID)
		if err != nil {
			return err
		}
		if stored != nil {
			fmt.Fprintf(out, "\nExisting profile (created %s): root=%s\n", stored.CreatedAt, stored.Root)
			ok, err := promptConfirm(in, out, "Overwrite existing profile?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(out, "Setup cancelled.")
				return nil
			}
		}
	}

	// Built-in capability discovery
	trusted, err := store.Skills()
	if err != nil {
		return err
	}
	catalogCaps := intelligence.DiscoverCapabilities(profile, trusted)

	// Live skill search via skills.sh
	seen := map[string]bool{}
	for _, c := range catalogCaps {
		seen[c.Kind+":"+c.Ref] = true
	}
	var liveGroups []skillGroup
	if len(setupKeywords(profile)) > 0 {
		fmt.Fprintln(out, "\nSearching AAS skill catalog…")
		liveGroups = searchCatalogSkills(profile, store, seen, out, errOut, skipLive)
	}

	totalLive := 0
	for _, g := range liveGroups {
		totalLive += len(g.caps)
	}
	if len(catalogCaps) == 0 && totalLive == 0 {
		fmt.Fprintln(out, "\nNo capabilities matched the detected project profile.")
	}

	if dryRun {
		fmt.Fprintln(out, "\n[dry-run] built-in capabilities:")
		for i, cap := range catalogCaps {
			fmt.Fprintf(out, "  %2d. [%s][%s] %s\n", i+1, cap.Kind, string(cap.Network), cap.Name)
		}
		offset := len(catalogCaps)
		for _, g := range liveGroups {
			if len(g.caps) == 0 {
				continue
			}
			fmt.Fprintf(out, "\n[dry-run] %s:\n", g.label)
			for i, cap := range g.caps {
				fmt.Fprintf(out, "  %2d. [skill][network] %s\n", offset+i+1, cap.Name)
			}
			offset += len(g.caps)
		}
		fmt.Fprintln(out, "\n[dry-run] no files or database records were written.")
		return nil
	}

	// Selection
	agentPaths, err := intelligence.DefaultAgentConfigPaths()
	if err != nil {
		return err
	}
	selected, err := promptSelection(in, out, catalogCaps, liveGroups)
	if err != nil {
		return err
	}

	// Provisioning
	results := []intelligence.ProvisionResult{}
	for _, cap := range selected {
		result := intelligence.Provision(cap, agentPaths, store, false, in, out)
		results = append(results, result)
		if result.Outcome == intelligence.OutcomeFailed && result.Error != "" {
			payload, _ := json.Marshal(map[string]any{"capability": cap.Name, "error": result.Error})
			_, logErr := store.Log("setup_session", project.ID, json.RawMessage(payload))
			if logErr != nil {
				fmt.Fprintln(errOut, "brain: setup event log failed:", logErr)
			}
		}
	}

	// Persist profile
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	capsJSON, err := json.Marshal(results)
	if err != nil {
		return err
	}
	if _, err = store.UpsertProfile(project.ID, root, string(profileJSON), string(capsJSON)); err != nil {
		return fmt.Errorf("failed to persist project profile: %w", err)
	}

	if _, logErr := store.Log("setup_session", project.ID, buildSessionEvent(results)); logErr != nil {
		fmt.Fprintln(errOut, "brain: setup session log failed:", logErr)
	}

	printProvisionSummary(out, results)
	fmt.Fprintln(out, "\nSetup complete. Run 'brain setup --status' to inspect the stored profile.")
	return nil
}
