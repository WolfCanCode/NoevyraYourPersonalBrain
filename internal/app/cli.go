package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

// Run is the top-level entry point for the brain CLI.
// It dispatches to per-command handlers based on args[0].
func Run(args []string, version, commit, builtAt string, in io.Reader, out, errOut io.Writer) error {
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	// Commands that do not require the memory store.
	switch cmd {
	case "help", "--help", "-h":
		if isTTY() {
			printBanner()
		}
		renderHelp(out)
		return nil
	case "version", "--version":
		return printJSON(out, map[string]any{
			"version": version, "commit": commit, "builtAt": builtAt,
			"go": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH,
		})
	case "mcp-config":
		v, err := mcpConfig()
		if err != nil {
			return err
		}
		return printJSON(out, v)
	}

	allowed := map[string]bool{
		"init": true, "remember": true, "recall": true, "context": true,
		"profile": true, "advise": true, "feedback": true, "sources": true,
		"approve": true, "status": true, "digest": true, "sync": true,
		"skills": true, "run": true, "doctor": true, "mcp": true,
		"setup": true, "update": true, "steer": true,
	}
	if !allowed[cmd] {
		return fmt.Errorf("unknown command %q; run brain help", cmd)
	}

	home, err := memory.Home()
	if err != nil {
		return err
	}

	// Commands that need home but not the store.
	switch cmd {
	case "update":
		return runUpdate(version, args, out, errOut)
	case "steer":
		return runSteer(args, out, errOut)
	case "doctor":
		return runDoctorCmd(version, home, out)
	}

	store, err := memory.Open(home)
	if err != nil {
		return err
	}
	defer store.Close()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	project, err := memory.ProjectFor(cwd)
	if err != nil {
		return err
	}

	// Commands that require the store.
	switch cmd {
	case "mcp":
		return ServeMCP(store, project, version, in, out)
	case "init":
		return runInitCmd(store, args, home, out, errOut)
	case "remember":
		return runRememberCmd(store, project, args, out)
	case "recall", "context":
		return runQueryCmd(cmd, store, project, args, out)
	case "profile":
		return runProfileCmd(project, args, out)
	case "advise":
		return runAdviseCmd(store, project, args, out)
	case "feedback":
		return runFeedbackCmd(store, args, out)
	case "sources":
		return sources(store, args, out)
	case "approve":
		return runApproveCmd(store, args, out)
	case "status":
		return runStatusCmd(store, project, out)
	case "digest":
		return runDigestCmd(store, args, out)
	case "sync":
		return runSyncCmd(store, args, out)
	case "skills":
		return skills(store, args, out)
	case "run":
		return runAgent(store, project, args, in, out, errOut, home)
	case "setup":
		return runSetup(store, project, args, in, out, errOut, home)
	}
	return errors.New("unhandled command")
}
