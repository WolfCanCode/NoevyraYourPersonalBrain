package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func showSetupStatus(store *memory.Store, project memory.Project, out io.Writer) error {
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
	if !isTTY() {
		fmt.Fprintln(out, "Stored profile for project:", stored.ProjectID)
		fmt.Fprintln(out, "Root:      ", stored.Root)
		fmt.Fprintln(out, "Created:   ", stored.CreatedAt)
		displayStoredProfile(out, stored.ProfileJSON)
		return nil
	}
	printSectionLabel("Stored profile")
	fmt.Println()
	printKeyValue("Project", styleAccent.Render(stored.ProjectID))
	printKeyValue("Root", styleMuted.Render(stored.Root))
	printKeyValue("Created", styleMuted.Render(stored.CreatedAt))
	displayStoredProfile(os.Stdout, stored.ProfileJSON)
	if stored.CapabilitiesJSON == "" || stored.CapabilitiesJSON == "[]" {
		return nil
	}
	var caps []intelligence.ProvisionResult
	if json.Unmarshal([]byte(stored.CapabilitiesJSON), &caps) != nil {
		return nil
	}
	printSectionLabel("Provisioned capabilities")
	fmt.Println()
	for _, cap := range caps {
		if cap.Outcome != intelligence.OutcomeProvisioned {
			continue
		}
		fmt.Printf("  %s  %s\n", kindBadge(cap.Capability.Kind), styleWhite.Render(cap.Capability.Name))
	}
	return nil
}

func displayStoredProfile(out io.Writer, raw string) {
	var profile intelligence.ProjectProfile
	if json.Unmarshal([]byte(raw), &profile) == nil {
		displayProfile(out, profile)
	}
}
