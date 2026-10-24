package app

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// runDoctorCmd checks prerequisites, binary availability, and scheduler state.
func runDoctorCmd(version, home string, out io.Writer) error {
	checkedBins := []string{"git", "claude", "codex", "cursor-agent", "kiro", "opencode", "pi", "npx"}
	commands := make(map[string]bool, len(checkedBins))
	for _, bin := range checkedBins {
		_, e := exec.LookPath(bin)
		commands[bin] = e == nil
	}
	if isTTY() {
		printHeader("brain doctor")
		fmt.Println()
		printKeyValue("Version", styleAccent.Render(version))
		printKeyValue("Platform", styleMuted.Render(runtime.GOOS+"/"+runtime.GOARCH))
		printKeyValue("Go", styleMuted.Render(runtime.Version()))
		printKeyValue("Store", styleMuted.Render(home))
		fmt.Println()
		printSectionLabel("Tools")
		fmt.Println()
		for _, bin := range checkedBins {
			fmt.Printf("  %s  %s\n", checkBadge(commands[bin]), styleWhite.Render(bin))
		}
		fmt.Println()
		printSectionLabel("Scheduler")
		fmt.Println()
		schStatus := schedulerStatus()
		if active, ok := schStatus["active"].(bool); ok {
			fmt.Printf("  %s  daily sync\n", checkBadge(active))
		}
		fmt.Println()
		return nil
	}
	return printJSON(out, map[string]any{
		"version":   version,
		"go":        runtime.Version(),
		"platform":  runtime.GOOS,
		"store":     home,
		"commands":  commands,
		"scheduler": schedulerStatus(),
	})
}
