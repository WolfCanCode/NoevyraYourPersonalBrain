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
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ── Project root detection ────────────────────────────────────────────────

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

// ── Profile display ───────────────────────────────────────────────────────

func displayProfile(out io.Writer, profile intelligence.ProjectProfile) {
	if isTTY() {
		fmt.Println()
		printSectionLabel("Detected profile")
		fmt.Println()
		if len(profile.Languages) > 0 {
			printKeyValue("Languages", strings.Join(profile.Languages, "  "+styleMuted.Render("·")+"  "))
		}
		if len(profile.Stacks) > 0 {
			shown := profile.Stacks
			extra := 0
			if len(shown) > 5 {
				extra = len(shown) - 5
				shown = shown[:5]
			}
			line := strings.Join(shown, "  "+styleMuted.Render("·")+"  ")
			if extra > 0 {
				line += styleMuted.Render(fmt.Sprintf("  +%d more", extra))
			}
			printKeyValue("Stacks", line)
		}
		if len(profile.Signals) > 0 {
			signals := make([]string, len(profile.Signals))
			for i, s := range profile.Signals {
				signals[i] = lipgloss.NewStyle().Foreground(clrPurple).Render(s)
			}
			printKeyValue("Signals", strings.Join(signals, "  "))
		}
		if profile.Truncated {
			fmt.Println(styleMuted.Render("  ⚠  scan truncated at 4 000 files"))
		}
		fmt.Println()
		return
	}
	// Non-TTY fallback
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

// ── Confirm prompt ────────────────────────────────────────────────────────

func promptConfirm(in io.Reader, out io.Writer, msg string) (bool, error) {
	if isTTY() {
		var ok bool
		err := huh.NewConfirm().
			Title(msg).
			Affirmative("Yes, overwrite").
			Negative("No, keep existing").
			Value(&ok).
			Run()
		return ok, err
	}
	// Non-TTY fallback
	fmt.Fprintf(out, "%s [y/N]: ", msg)
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return line == "y" || line == "Y", nil
	}
	return false, scanner.Err()
}

// ── skillGroup holds per-keyword results with a display label ─────────────

type skillGroup struct {
	label string
	caps  []intelligence.ProvisionableCapability
}

// ── Interactive multi-select with huh ─────────────────────────────────────

func toCapEntries(caps []intelligence.ProvisionableCapability) []capEntry {
	out := make([]capEntry, len(caps))
	for i, c := range caps {
		out[i] = capEntry{
			Ref:          c.Ref,
			Kind:         c.Kind,
			Name:         c.Name,
			Network:      string(c.Network),
			Reason:       c.Reason,
			InstallCount: c.InstallCount,
		}
	}
	return out
}

func promptSelection(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup) ([]intelligence.ProvisionableCapability, error) {
	all := make([]intelligence.ProvisionableCapability, 0, len(catalog)+20)
	all = append(all, catalog...)
	for _, g := range liveGroups {
		all = append(all, g.caps...)
	}
	if len(all) == 0 {
		return nil, nil
	}

	if isTTY() {
		return promptSelectionInteractive(catalog, liveGroups, all)
	}
	return promptSelectionText(in, out, catalog, liveGroups, all)
}

func promptSelectionInteractive(catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup, all []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
	// Build huh options with badge labels
	opts := make([]huh.Option[string], 0, len(all))

	// Section: built-in catalog
	if len(catalog) > 0 {
		for _, cap := range catalog {
			e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason}
			label := capOptionLabel(e)
			opts = append(opts, huh.NewOption(label, cap.Ref))
		}
	}
	// Sections: per-keyword groups
	for _, g := range liveGroups {
		for _, cap := range g.caps {
			e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason, InstallCount: cap.InstallCount}
			label := capOptionLabel(e)
			opts = append(opts, huh.NewOption(label, cap.Ref))
		}
	}

	var selectedRefs []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(styleAccent.Render("Select capabilities to provision")).
				Description(styleMuted.Render("↑↓ navigate  ·  Space select  ·  Enter confirm  ·  a toggle all")).
				Options(opts...).
				Value(&selectedRefs),
		),
	).WithTheme(huh.ThemeDracula())

	if err := form.Run(); err != nil {
		return nil, err
	}

	refMap := map[string]intelligence.ProvisionableCapability{}
	for _, cap := range all {
		refMap[cap.Ref] = cap
	}
	selected := make([]intelligence.ProvisionableCapability, 0, len(selectedRefs))
	for _, ref := range selectedRefs {
		if cap, ok := refMap[ref]; ok {
			selected = append(selected, cap)
		}
	}
	return selected, nil
}

func promptSelectionText(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, liveGroups []skillGroup, all []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
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
			line := fmt.Sprintf("    %2d. [skill][network] %s", offset+i+1, cap.Name)
			if cap.InstallCount != "" {
				line += fmt.Sprintf(" · 📦 %s", cap.InstallCount)
			}
			fmt.Fprintln(out, line)
			fmt.Fprintf(out, "        %s\n", cap.Reason)
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

// ── Provisioning summary ──────────────────────────────────────────────────

func printProvisionSummary(out io.Writer, results []intelligence.ProvisionResult) {
	if isTTY() {
		fmt.Println()
		printSectionLabel("Provisioning summary")
		fmt.Println()
		for _, r := range results {
			badge := outcomeBadge(string(r.Outcome))
			name := styleWhite.Render(r.Capability.Name)
			if r.Error != "" {
				name += styleMuted.Render("  " + r.Error)
			}
			fmt.Printf("  %s  %s\n", badge, name)
		}
		fmt.Println()
		return
	}
	fmt.Fprintln(out, "\nProvisioning summary:")
	for _, r := range results {
		status := string(r.Outcome)
		if r.Error != "" {
			status += " (" + r.Error + ")"
		}
		fmt.Fprintf(out, "  %-36s %s\n", r.Capability.Name, status)
	}
}

// ── Session event builder ─────────────────────────────────────────────────

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

// ── Live skill search ─────────────────────────────────────────────────────

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
	// Catalog not seeded
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
	if _, lookErr := exec.LookPath("npx"); lookErr != nil {
		return nil
	}
	npxKeywords := keywords
	if len(npxKeywords) > 3 {
		npxKeywords = npxKeywords[:3]
	}
	groups := []skillGroup{}
	for _, kw := range npxKeywords {
		if isTTY() {
			fmt.Printf("  %s Searching skills.sh for %s…\n",
				styleMuted.Render("→"),
				styleAccent.Render(kw))
		} else {
			fmt.Fprintf(out, "  Searching skills.sh for %q (npx fallback)…\n", kw)
		}
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

// ── npx output parser ─────────────────────────────────────────────────────

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
		desc = strings.Trim(desc, "-–—:\"'|·•·\t ")

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

// ── runSetup orchestration ────────────────────────────────────────────────

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

	if isTTY() {
		printHeader("brain setup")
	}

	// ── --status path ────────────────────────────────────────────────────
	if f["--status"] != "" {
		stored, err := store.LatestProfile(project.ID)
		if err != nil {
			return err
		}
		if stored == nil {
			if isTTY() {
				fmt.Println(styleWarn.Render("  No profile found for this project."))
				fmt.Println(styleMuted.Render("  Run 'brain setup' to create one."))
			} else {
				fmt.Fprintln(out, "No profile found for the current project. Run 'brain setup' to create one.")
			}
			return nil
		}
		if isTTY() {
			printSectionLabel("Stored profile")
			fmt.Println()
			printKeyValue("Project", styleAccent.Render(stored.ProjectID))
			printKeyValue("Root", styleMuted.Render(stored.Root))
			printKeyValue("Created", styleMuted.Render(stored.CreatedAt))
			var profile intelligence.ProjectProfile
			if json.Unmarshal([]byte(stored.ProfileJSON), &profile) == nil {
				displayProfile(os.Stdout, profile)
			}
			if stored.CapabilitiesJSON != "" && stored.CapabilitiesJSON != "[]" {
				var caps []intelligence.ProvisionResult
				if json.Unmarshal([]byte(stored.CapabilitiesJSON), &caps) == nil {
					printSectionLabel("Provisioned capabilities")
					fmt.Println()
					for _, c := range caps {
						if c.Outcome == intelligence.OutcomeProvisioned {
							fmt.Printf("  %s  %s\n", kindBadge(c.Capability.Kind), styleWhite.Render(c.Capability.Name))
						}
					}
				}
			}
		} else {
			fmt.Fprintln(out, "Stored profile for project:", stored.ProjectID)
			fmt.Fprintln(out, "Root:      ", stored.Root)
			fmt.Fprintln(out, "Created:   ", stored.CreatedAt)
			var profile intelligence.ProjectProfile
			if json.Unmarshal([]byte(stored.ProfileJSON), &profile) == nil {
				displayProfile(out, profile)
			}
		}
		return nil
	}

	// ── Detection phase ──────────────────────────────────────────────────
	root := projectRoot(project.Root)
	profile, err := intelligence.Profile(root)
	if err != nil {
		return err
	}
	displayProfile(out, profile)

	// ── Overwrite check ──────────────────────────────────────────────────
	if f["--refresh"] == "" {
		stored, err := store.LatestProfile(project.ID)
		if err != nil {
			return err
		}
		if stored != nil {
			if isTTY() {
				fmt.Printf("  %s  %s\n", styleMuted.Render("Existing profile:"), styleMuted.Render("created "+stored.CreatedAt))
			} else {
				fmt.Fprintf(out, "\nExisting profile (created %s): root=%s\n", stored.CreatedAt, stored.Root)
			}
			ok, err := promptConfirm(in, out, "Overwrite existing profile?")
			if err != nil {
				return err
			}
			if !ok {
				if isTTY() {
					fmt.Println(styleMuted.Render("  Setup cancelled."))
				} else {
					fmt.Fprintln(out, "Setup cancelled.")
				}
				return nil
			}
		}
	}

	// ── Capability discovery ──────────────────────────────────────────────
	trusted, err := store.Skills()
	if err != nil {
		return err
	}
	catalogCaps := intelligence.DiscoverCapabilities(profile, trusted)

	seen := map[string]bool{}
	for _, c := range catalogCaps {
		seen[c.Kind+":"+c.Ref] = true
	}

	// Silently seed the embedded catalog if the store has never been synced.
	// This makes skill suggestions work immediately without any manual step.
	_, _ = intelligence.SeedEmbeddedCatalog(store)

	var liveGroups []skillGroup
	if len(setupKeywords(profile)) > 0 {
		if isTTY() {
			printSectionLabel("Searching AAS skill catalog…")
			fmt.Println()
		} else {
			fmt.Fprintln(out, "\nSearching AAS skill catalog…")
		}
		liveGroups = searchCatalogSkills(profile, store, seen, out, errOut, skipLive)
	}

	totalLive := 0
	for _, g := range liveGroups {
		totalLive += len(g.caps)
	}
	if len(catalogCaps) == 0 && totalLive == 0 {
		if isTTY() {
			fmt.Println(styleMuted.Render("  No capabilities matched the detected project profile."))
		} else {
			fmt.Fprintln(out, "\nNo capabilities matched the detected project profile.")
		}
	}

	// ── Dry run ───────────────────────────────────────────────────────────
	if dryRun {
		if isTTY() {
			printSectionLabel("[dry-run] built-in capabilities")
			fmt.Println()
			for _, cap := range catalogCaps {
				e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason}
				fmt.Println(" ", capOptionLabel(e))
			}
			offset := len(catalogCaps)
			for _, g := range liveGroups {
				if len(g.caps) == 0 {
					continue
				}
				fmt.Println(styleSection.Render("  [dry-run] " + g.label))
				for i, cap := range g.caps {
					e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason, InstallCount: cap.InstallCount}
					fmt.Printf("  %2d. %s\n", offset+i+1, capOptionLabel(e))
				}
				offset += len(g.caps)
			}
			fmt.Println(styleMuted.Render("\n  [dry-run] no files or records written."))
		} else {
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
		}
		return nil
	}

	// ── Selection & provisioning ──────────────────────────────────────────
	agentPaths, err := intelligence.DefaultAgentConfigPaths()
	if err != nil {
		return err
	}
	selected, err := promptSelection(in, out, catalogCaps, liveGroups)
	if err != nil {
		return err
	}

	results := []intelligence.ProvisionResult{}
	for _, cap := range selected {
		if isTTY() {
			fmt.Printf("  %s Provisioning %s…\n",
				styleMuted.Render("→"),
				styleAccent.Render(cap.Name))
		}
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

	if isTTY() {
		fmt.Println(styleSuccess.Render("  ✦ Setup complete.") + styleMuted.Render("  Run 'brain setup --status' to inspect."))
		fmt.Println()
	} else {
		fmt.Fprintln(out, "\nSetup complete. Run 'brain setup --status' to inspect the stored profile.")
	}
	return nil
}
