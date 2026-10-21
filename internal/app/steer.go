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
}

// runSteer implements "brain steer".
//
//	--global   write to home-directory files instead of the current project
//	--dry-run  print what would be written without creating files
func runSteer(args []string, out, errOut io.Writer) error {
	_, f, err := flags(args, nil, []string{"--global", "--dry-run"})
	if err != nil {
		return err
	}
	global := f["--global"] != ""
	dryRun := f["--dry-run"] != ""

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
		// Detect whether the agent is installed.
		_, lookErr := exec.LookPath(agent.binary)
		installed := lookErr == nil

		// Determine target file path.
		rel := agent.projectFile
		if global {
			rel = agent.globalFile
		}
		var target string
		if global {
			target = filepath.Join(home, rel)
		} else {
			target = filepath.Join(cwd, rel)
		}

		content := agent.header + brainSteeringContent

		if dryRun {
			status := styleMuted.Render("not in PATH")
			if installed {
				status = styleSuccess.Render("detected")
			}
			if isTTY() {
				fmt.Printf("  %s  %s  %s\n",
					kindBadge("skill"),
					styleWhite.Render(fmt.Sprintf("%-8s", agent.name)),
					status,
				)
				fmt.Printf("    %s\n\n", styleMuted.Render(target))
			} else {
				fmt.Fprintf(out, "  %-8s  %s\n    %s\n\n",
					agent.name, map[bool]string{true: "detected", false: "not in PATH"}[installed], target)
			}
			continue
		}

		if !installed {
			notFound = append(notFound, agent.name)
			if isTTY() {
				fmt.Printf("  %s  %-8s  %s\n",
					styleMuted.Render("⊝"),
					agent.name,
					styleMuted.Render("not in PATH — skipped"),
				)
			} else {
				fmt.Fprintf(out, "  %-8s  not in PATH — skipped\n", agent.name)
			}
			continue
		}

		// Check if file already contains brain content.
		if existing, readErr := os.ReadFile(target); readErr == nil {
			if strings.Contains(string(existing), "brain_recall") ||
				strings.Contains(string(existing), "brain_remember") {
				skipped = append(skipped, agent.name)
				if isTTY() {
					fmt.Printf("  %s  %-8s  %s\n",
						styleMuted.Render("⊝"),
						agent.name,
						styleMuted.Render("already has brain instructions — skipped (use --force to overwrite)"),
					)
				} else {
					fmt.Fprintf(out, "  %-8s  already configured — skipped\n", agent.name)
				}
				continue
			}
			// Append to existing file.
			content = strings.TrimRight(string(existing), "\n") + "\n\n" + agent.header + brainSteeringContent
		}

		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			fmt.Fprintf(errOut, "brain steer: mkdir %s: %v\n", filepath.Dir(target), err)
			continue
		}
		if err = os.WriteFile(target, []byte(content), 0600); err != nil {
			fmt.Fprintf(errOut, "brain steer: write %s: %v\n", target, err)
			continue
		}
		written = append(written, agent.name)
		if isTTY() {
			fmt.Printf("  %s  %-8s  %s\n",
				styleSuccess.Render("✓"),
				agent.name,
				styleMuted.Render(target),
			)
		} else {
			fmt.Fprintf(out, "  %-8s  written → %s\n", agent.name, target)
		}
	}

	fmt.Fprintln(out)
	if isTTY() {
		if len(written) > 0 {
			fmt.Println(styleSuccess.Render(fmt.Sprintf("  ✦ %d agent(s) configured.", len(written))))
		}
		fmt.Println(styleMuted.Render("  Each agent will now automatically call brain_recall, brain_remember,"))
		fmt.Println(styleMuted.Render("  and brain_finish_task without being asked."))
		if len(notFound) > 0 {
			fmt.Println(styleMuted.Render(fmt.Sprintf("  Not in PATH: %s — install and re-run 'brain steer'.", strings.Join(notFound, ", "))))
		}
		if len(skipped) > 0 {
			fmt.Println(styleMuted.Render(fmt.Sprintf("  Already configured: %s.", strings.Join(skipped, ", "))))
		}
	} else {
		if len(written) > 0 {
			fmt.Fprintf(out, "Done. %d agent(s) configured: %s\n", len(written), strings.Join(written, ", "))
		}
		fmt.Fprintln(out, "Each agent will now auto-call brain_recall, brain_remember, brain_finish_task.")
	}
	fmt.Fprintln(out)
	return nil
}
