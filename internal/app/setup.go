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
func promptSelection(in io.Reader, out io.Writer, catalog []intelligence.ProvisionableCapability, live []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
	all := append(catalog, live...)
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
	if len(live) > 0 {
		fmt.Fprintln(out, "\n  From skills.sh:")
		for i, cap := range live {
			fmt.Fprintf(out, "    %2d. [skill][network] %s\n        %s\n", len(catalog)+i+1, cap.Name, cap.Reason)
		}
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

// setupKeywords returns at most 2 search keywords derived from the project
// profile, prioritising specific framework signals over generic language names.
// These are passed to `npx skills find` to retrieve relevant community skills.
var keywordPriority = []string{
	// Framework signals (most specific — detected from package deps)
	"angular", "react", "vue", "svelte", "nextjs", "nestjs",
	"graphql", "storybook", "tailwind", "redux",
	// Language-level signals
	"go", "rust", "python",
	// Generic (only if nothing more specific matched)
	"typescript", "javascript",
	// Operational signals
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
	result := []string{}
	for _, kw := range keywordPriority {
		if sigSet[kw] {
			result = append(result, kw)
			if len(result) == 2 {
				break
			}
		}
	}
	return result
}

// skillRefRe matches owner/repo@skill-name patterns in skills.sh output.
var skillRefRe = regexp.MustCompile(`[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+`)

// parseSkillOutput extracts ProvisionableCapabilities from `npx skills find` output.
func parseSkillOutput(raw, keyword string) []intelligence.ProvisionableCapability {
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

// searchLiveSkills calls `npx skills find <keyword>` for each keyword and
// returns parsed skill capabilities not already present in the seen map.
// Errors are soft-logged to errOut so a missing npx or network failure does
// not block the rest of the setup flow.
func searchLiveSkills(keywords []string, seen map[string]bool, out, errOut io.Writer) []intelligence.ProvisionableCapability {
	if _, err := exec.LookPath("npx"); err != nil {
		return nil
	}
	caps := []intelligence.ProvisionableCapability{}
	for _, kw := range keywords {
		fmt.Fprintf(out, "  Searching skills.sh for %q…\n", kw)
		raw, err := external("npx", []string{"--yes", "skills", "find", kw})
		if err != nil {
			fmt.Fprintf(errOut, "brain: skills search for %q unavailable: %v\n", kw, err)
			continue
		}
		for _, cap := range parseSkillOutput(raw, kw) {
			key := "skill:" + cap.Ref
			if seen[key] {
				continue
			}
			seen[key] = true
			caps = append(caps, cap)
		}
	}
	return caps
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
	var liveCaps []intelligence.ProvisionableCapability
	if !skipLive {
		keywords := setupKeywords(profile)
		if len(keywords) > 0 {
			fmt.Fprintln(out, "\nSearching skills.sh for project-relevant skills…")
			liveCaps = searchLiveSkills(keywords, seen, out, errOut)
		}
	}

	if len(catalogCaps) == 0 && len(liveCaps) == 0 {
		fmt.Fprintln(out, "\nNo capabilities matched the detected project profile.")
	}

	if dryRun {
		fmt.Fprintln(out, "\n[dry-run] built-in capabilities:")
		for i, cap := range catalogCaps {
			fmt.Fprintf(out, "  %2d. [%s][%s] %s\n", i+1, cap.Kind, string(cap.Network), cap.Name)
		}
		if len(liveCaps) > 0 {
			fmt.Fprintln(out, "\n[dry-run] skills from skills.sh:")
			for i, cap := range liveCaps {
				fmt.Fprintf(out, "  %2d. [skill][network] %s\n", len(catalogCaps)+i+1, cap.Name)
			}
		}
		fmt.Fprintln(out, "\n[dry-run] no files or database records were written.")
		return nil
	}

	// Selection
	agentPaths, err := intelligence.DefaultAgentConfigPaths()
	if err != nil {
		return err
	}
	selected, err := promptSelection(in, out, catalogCaps, liveCaps)
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
