package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// steeringAgent describes one agent's steering file locations and format.
type steeringAgent struct {
	name        string // display name
	binary      string // exec.LookPath target
	projectFile string // path relative to cwd  (e.g. "CLAUDE.md")
	globalFile  string // path relative to home  (e.g. ".claude/CLAUDE.md")
	header      string // optional front-matter / heading for this agent
}

// brainSteeringContent is the shared core instruction block written into
// every agent's steering file. It tells the agent to use brain MCP tools
// automatically without requiring explicit user prompts.
const brainSteeringContent = `
## Brain — automatic memory workflow

Brain (Noevyra) is connected as an MCP server. Use its tools automatically
without waiting for the user to ask.

**MCP tools available:**
brain_advise · brain_recall · brain_remember · brain_context ·
brain_feedback · brain_finish_task · brain_handoff

---

### At the START of every implementation task

1. Call brain_recall(query="<task keywords>") to surface relevant decisions
   and constraints the team has already recorded.
2. For complex or risky tasks, call brain_advise(task="<description>") to get
   a full pack: profile, reasoning frame, risks, and capability recommendations.

### DURING the task — call brain_remember when

- An architectural or design decision is confirmed
- A convention or constraint is identified ("we use X not Y")
- A non-obvious approach is chosen and the reason should be preserved

  brain_remember(content="<decision and reasoning>", type="decision")

  Agent-proposed memories are held as candidates until the user approves
  them with ` + "`brain approve <id>`" + `. Do not treat them as verified facts.

### After COMPLETING a task

Call brain_finish_task to record the outcome:

  brain_finish_task(
    summary  = "<concise description of what was done>",
    status   = "completed" | "blocked" | "failed",
    evidence = "<key file, commit, or decision — optional>"
  )

### When HANDING OFF to another agent

  brain_handoff(
    objective  = "<original goal>",
    completed  = ["<step 1>", "<step 2>"],
    remaining  = ["<next step>"]
  )

---

Do all of the above automatically. Do not ask the user for permission.
Keep brain_remember entries concise (one decision per entry).
`

var steeringAgents = []steeringAgent{
	{
		name:        "Claude",
		binary:      "claude",
		projectFile: "CLAUDE.md",
		globalFile:  ".claude/CLAUDE.md",
		header:      "# Noevyra brain integration\n",
	},
	{
		name:        "Codex",
		binary:      "codex",
		projectFile: "AGENTS.md",
		globalFile:  ".codex/AGENTS.md",
		header:      "# Noevyra brain integration\n",
	},
	{
		name:        "Cursor",
		binary:      "cursor-agent",
		projectFile: ".cursor/rules/brain.mdc",
		globalFile:  ".cursor/rules/brain.mdc",
		header: `---
description: Brain (Noevyra) automatic memory workflow
globs: ["**/*"]
alwaysApply: true
---
# Noevyra brain integration
`,
	},
	{
		name:        "Kiro",
		binary:      "kiro",
		projectFile: ".kiro/steering/brain.md",
		globalFile:  ".kiro/steering/brain.md",
		header:      "# Noevyra brain integration\n",
	},
	// OpenCode uses AGENTS.md (same as Codex for project-level; the existing-content
	// check avoids duplicates when both are installed).
	{
		name:        "OpenCode",
		binary:      "opencode",
		projectFile: "AGENTS.md",
		globalFile:  ".config/opencode/AGENTS.md",
		header:      "# Noevyra brain integration\n",
	},
	// Pi (pi.dev coding agent — @mariozechner/pi-coding-agent) also uses AGENTS.md.
	{
		name:        "Pi",
		binary:      "pi",
		projectFile: "AGENTS.md",
		globalFile:  ".config/pi/AGENTS.md",
		header:      "# Noevyra brain integration\n",
	},
}

// runSteer implements "brain steer".
//
//	--global   write to home-directory files instead of the current project
//	--dry-run  print what would be written without creating files
func runSteer(args []string, out, errOut io.Writer) error {
	_, options, err := flags(args, nil, []string{"--global", "--dry-run"})
	if err != nil {
		return err
	}
	global, dryRun := options["--global"] != "", options["--dry-run"] != ""
	if isTTY() {
		printHeader("brain steer")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	scope := "project"
	if global {
		scope = "global"
	}
	if isTTY() {
		fmt.Printf("  %s  scope: %s\n\n", styleMuted.Render("Creating steering files for all detected agents —"), styleAccent.Render(scope))
	} else {
		fmt.Fprintf(out, "Creating brain steering files (%s scope)\n\n", scope)
	}
	written, skipped, notFound := []string{}, []string{}, []string{}
	for _, agent := range steeringAgents {
		target := steeringTarget(agent, cwd, home, global)
		_, lookErr := exec.LookPath(agent.binary)
		if dryRun {
			printSteeringPreview(out, agent, target, lookErr == nil)
			continue
		}
		if lookErr != nil {
			notFound = append(notFound, agent.name)
			printSteeringResult(out, agent, target, "missing")
			continue
		}
		changed, err := configureSteeringFile(agent, target)
		if err != nil {
			fmt.Fprintln(errOut, "brain steer:", err)
			continue
		}
		if !changed {
			skipped = append(skipped, agent.name)
			printSteeringResult(out, agent, target, "skipped")
			continue
		}
		written = append(written, agent.name)
		printSteeringResult(out, agent, target, "written")
	}
	printSteeringSummary(out, written, skipped, notFound)
	return nil
}

func steeringTarget(agent steeringAgent, cwd, home string, global bool) string {
	if global {
		return filepath.Join(home, agent.globalFile)
	}
	return filepath.Join(cwd, agent.projectFile)
}

func configureSteeringFile(agent steeringAgent, target string) (bool, error) {
	content := agent.header + brainSteeringContent
	existing, readErr := os.ReadFile(target)
	if readErr == nil {
		if strings.Contains(string(existing), "brain_recall") || strings.Contains(string(existing), "brain_remember") {
			return false, nil
		}
		content = strings.TrimRight(string(existing), "\n") + "\n\n" + content
	}
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
	}
	if err := os.WriteFile(target, []byte(content), 0600); err != nil {
		return false, fmt.Errorf("write %s: %w", target, err)
	}
	return true, nil
}
