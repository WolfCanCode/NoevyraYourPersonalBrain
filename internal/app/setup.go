package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
			break // filesystem root
		}
		dir = parent
	}
	return cwd
}

// displayProfile writes detected languages, stacks, and signals to out (Req 1.6).
func displayProfile(out io.Writer, profile intelligence.ProjectProfile) {
	fmt.Fprintln(out, "Detected profile:")
	if len(profile.Languages) > 0 {
		fmt.Fprintln(out, "  Languages:", strings.Join(profile.Languages, ", "))
	}
	if len(profile.Stacks) > 0 {
		fmt.Fprintln(out, "  Stacks:   ", strings.Join(profile.Stacks, ", "))
	}
	if len(profile.Signals) > 0 {
		fmt.Fprintln(out, "  Signals:  ", strings.Join(profile.Signals, ", "))
	}
	if profile.Truncated {
		fmt.Fprintln(out, "  (scan truncated at 4000 files)")
	}
}

// promptConfirm reads a y/N answer from in. Returns true only for "y" or "Y".
func promptConfirm(in io.Reader, out io.Writer, msg string) (bool, error) {
	fmt.Fprintf(out, "%s [y/N]: ", msg)
	scanner := bufio.NewScanner(in)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return line == "y" || line == "Y", nil
	}
	return false, scanner.Err()
}

// promptSelection prints a numbered capability list and reads the user's choice.
// Accepts "all", "none", or comma-separated integers (e.g. "1,3").
func promptSelection(in io.Reader, out io.Writer, caps []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error) {
	if len(caps) == 0 {
		return nil, nil
	}
	fmt.Fprintln(out, "\nAvailable capabilities:")
	for i, cap := range caps {
		networkLabel := string(cap.Network)
		fmt.Fprintf(out, "  %2d. [%s][%s] %s — %s\n", i+1, cap.Kind, networkLabel, cap.Name, cap.Reason)
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
		return caps, nil
	}
	selected := []intelligence.ProvisionableCapability{}
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil || idx < 1 || idx > len(caps) {
			fmt.Fprintf(out, "  Skipping invalid selection: %q\n", part)
			continue
		}
		selected = append(selected, caps[idx-1])
	}
	return selected, nil
}

// printProvisionSummary writes an aligned table of capability name → outcome (Req 4.5).
func printProvisionSummary(out io.Writer, results []intelligence.ProvisionResult) {
	fmt.Fprintln(out, "\nProvisioning summary:")
	for _, r := range results {
		status := string(r.Outcome)
		if r.Error != "" {
			status += " (" + r.Error + ")"
		}
		fmt.Fprintf(out, "  %-30s %s\n", r.Capability.Name, status)
	}
}

// buildSessionEvent builds the payload for the setup_session event log entry (Req 7.4).
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

// runSetup is the handler for the "brain setup" CLI command.
// Flags: --refresh (skip overwrite prompt), --dry-run (print plan, write nothing),
//        --status (display stored profile, exit).
func runSetup(store *memory.Store, project memory.Project, args []string, in io.Reader, out, errOut io.Writer, home string) error {
	pos, f, err := flags(args, nil, []string{"--refresh", "--dry-run", "--status"})
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("setup accepts no positional arguments; got %v", pos)
	}
	dryRun := f["--dry-run"] != ""

	// --status: display stored profile and exit (Req 7.1, 7.2)
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

	// Detection phase (Req 1.1–1.6)
	root := projectRoot(project.Root)
	profile, err := intelligence.Profile(root)
	if err != nil {
		return err
	}
	displayProfile(out, profile)

	// Overwrite check (Req 2.4, 2.7)
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

	// Capability discovery (Req 3.1–3.7)
	trusted, err := store.Skills()
	if err != nil {
		return err
	}
	caps := intelligence.DiscoverCapabilities(profile, trusted)
	if len(caps) == 0 {
		fmt.Fprintln(out, "\nNo capabilities matched the detected project profile.")
	}

	if dryRun {
		fmt.Fprintln(out, "\n[dry-run] capability list:")
		for i, cap := range caps {
			fmt.Fprintf(out, "  %2d. [%s][%s] %s\n", i+1, cap.Kind, string(cap.Network), cap.Name)
		}
		fmt.Fprintln(out, "\n[dry-run] no files or database records were written.")
		return nil
	}

	// Selection and provisioning (Req 4.1–4.9)
	agentPaths, err := intelligence.DefaultAgentConfigPaths()
	if err != nil {
		return err
	}
	selected, err := promptSelection(in, out, caps)
	if err != nil {
		return err
	}
	results := []intelligence.ProvisionResult{}
	for _, cap := range selected {
		result := intelligence.Provision(cap, agentPaths, store, false, in, out)
		results = append(results, result)
		if result.Outcome == intelligence.OutcomeFailed && result.Error != "" {
			sessionPayload, _ := json.Marshal(map[string]any{"capability": cap.Name, "error": result.Error})
			_, logErr := store.Log("setup_session", project.ID, json.RawMessage(sessionPayload))
			if logErr != nil {
				fmt.Fprintln(errOut, "brain: setup event log failed:", logErr)
			}
		}
	}

	// Persist profile (Req 2.1)
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

	// Session event (Req 7.4)
	if _, logErr := store.Log("setup_session", project.ID, buildSessionEvent(results)); logErr != nil {
		fmt.Fprintln(errOut, "brain: setup session log failed:", logErr)
	}

	printProvisionSummary(out, results)
	fmt.Fprintln(out, "\nSetup complete. Run 'brain setup --status' to inspect the stored profile.")
	return nil
}
