package app

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

type setupCapabilities struct {
	builtIn []intelligence.ProvisionableCapability
	skills  []skillGroup
}

func confirmSetupOverwrite(store *memory.Store, projectID string, refresh bool, in io.Reader, out io.Writer) (bool, error) {
	if refresh {
		return true, nil
	}
	stored, err := store.LatestProfile(projectID)
	if err != nil {
		return false, err
	}
	if stored == nil {
		return true, nil
	}
	if isTTY() {
		fmt.Printf("  %s  %s\n", styleMuted.Render("Existing profile:"), styleMuted.Render("created "+stored.CreatedAt))
	} else {
		fmt.Fprintf(out, "\nExisting profile (created %s): root=%s\n", stored.CreatedAt, stored.Root)
	}
	ok, err := promptConfirm(in, out, "Overwrite existing profile?")
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	if isTTY() {
		fmt.Println(styleMuted.Render("  Setup cancelled."))
	} else {
		fmt.Fprintln(out, "Setup cancelled.")
	}
	return false, nil
}

func discoverSetupCapabilities(store *memory.Store, profile intelligence.ProjectProfile, skipLive bool, out, errOut io.Writer) (setupCapabilities, error) {
	trusted, err := store.Skills()
	if err != nil {
		return setupCapabilities{}, err
	}
	result := setupCapabilities{builtIn: intelligence.DiscoverCapabilities(profile, trusted)}
	seen := map[string]bool{}
	for _, cap := range result.builtIn {
		seen[cap.Kind+":"+cap.Ref] = true
	}
	// Seed first so offline discovery works without a manual sync.
	_, _ = intelligence.SeedEmbeddedCatalog(store)
	if len(setupKeywords(profile)) > 0 {
		if isTTY() {
			printSectionLabel("Searching AAS skill catalog…")
			fmt.Println()
		} else {
			fmt.Fprintln(out, "\nSearching AAS skill catalog…")
		}
		result.skills = searchCatalogSkills(profile, store, seen, out, errOut, skipLive)
	}
	total := len(result.builtIn)
	for _, group := range result.skills {
		total += len(group.caps)
	}
	if total == 0 {
		if isTTY() {
			fmt.Println(styleMuted.Render("  No capabilities matched the detected project profile."))
		} else {
			fmt.Fprintln(out, "\nNo capabilities matched the detected project profile.")
		}
	}
	return result, nil
}

func provisionSetupCapabilities(store *memory.Store, projectID string, selected []intelligence.ProvisionableCapability, paths intelligence.AgentConfigPaths, in io.Reader, out, errOut io.Writer) []intelligence.ProvisionResult {
	results := make([]intelligence.ProvisionResult, 0, len(selected))
	for _, cap := range selected {
		if isTTY() {
			fmt.Printf("  %s Provisioning %s…\n", styleMuted.Render("→"), styleAccent.Render(cap.Name))
		}
		result := intelligence.Provision(cap, paths, store, false, in, out)
		results = append(results, result)
		if result.Outcome != intelligence.OutcomeFailed || result.Error == "" {
			continue
		}
		payload, _ := json.Marshal(map[string]any{"capability": cap.Name, "error": result.Error})
		if _, err := store.Log("setup_session", projectID, json.RawMessage(payload)); err != nil {
			fmt.Fprintln(errOut, "brain: setup event log failed:", err)
		}
	}
	return results
}

func persistSetupProfile(store *memory.Store, projectID, root string, profile intelligence.ProjectProfile, results []intelligence.ProvisionResult, errOut io.Writer) error {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	capsJSON, err := json.Marshal(results)
	if err != nil {
		return err
	}
	if _, err := store.UpsertProfile(projectID, root, string(profileJSON), string(capsJSON)); err != nil {
		return fmt.Errorf("failed to persist project profile: %w", err)
	}
	if _, err := store.Log("setup_session", projectID, buildSessionEvent(results)); err != nil {
		fmt.Fprintln(errOut, "brain: setup session log failed:", err)
	}
	return nil
}
