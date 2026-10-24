package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

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

// runSetup coordinates detection, approval, provisioning and persistence.
func runSetup(store *memory.Store, project memory.Project, args []string, in io.Reader, out, errOut io.Writer, home string) error {
	pos, options, err := flags(args, nil, []string{"--refresh", "--dry-run", "--status", "--no-live-skills", "--no-steer"})
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("setup accepts no positional arguments; got %v", pos)
	}
	if isTTY() {
		printHeader("brain setup")
	}
	if options["--status"] != "" {
		return showSetupStatus(store, project, out)
	}
	root := projectRoot(project.Root)
	profile, err := intelligence.Profile(root)
	if err != nil {
		return err
	}
	displayProfile(out, profile)
	proceed, err := confirmSetupOverwrite(store, project.ID, options["--refresh"] != "", in, out)
	if err != nil || !proceed {
		return err
	}
	capabilities, err := discoverSetupCapabilities(store, profile, options["--no-live-skills"] != "", out, errOut)
	if err != nil {
		return err
	}
	if options["--dry-run"] != "" {
		printSetupDryRun(out, capabilities)
		return nil
	}
	agentPaths, err := intelligence.DefaultAgentConfigPaths()
	if err != nil {
		return err
	}
	selected, err := promptSelection(in, out, capabilities.builtIn, capabilities.skills)
	if err != nil {
		return err
	}
	results := provisionSetupCapabilities(store, project.ID, selected, agentPaths, in, out, errOut)
	if err := persistSetupProfile(store, project.ID, root, profile, results, errOut); err != nil {
		return err
	}
	printProvisionSummary(out, results)
	printSetupComplete(out)
	if options["--no-steer"] != "" {
		return nil
	}
	if isTTY() {
		printSectionLabel("Configuring agent steering files…")
		fmt.Println()
	}
	_ = runSteer(nil, out, errOut)
	return nil
}
