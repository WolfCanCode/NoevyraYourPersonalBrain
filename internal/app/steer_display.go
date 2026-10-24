package app

import (
	"fmt"
	"io"
	"strings"
)

func printSteeringPreview(out io.Writer, agent steeringAgent, target string, installed bool) {
	status := "not in PATH"
	styled := styleMuted.Render(status)
	if installed {
		status = "detected"
		styled = styleSuccess.Render(status)
	}
	if isTTY() {
		fmt.Printf("  %s  %s  %s\n", kindBadge("skill"), styleWhite.Render(fmt.Sprintf("%-8s", agent.name)), styled)
		fmt.Printf("    %s\n\n", styleMuted.Render(target))
		return
	}
	fmt.Fprintf(out, "  %-8s  %s\n    %s\n\n", agent.name, status, target)
}

func printSteeringResult(out io.Writer, agent steeringAgent, target, outcome string) {
	symbol := styleMuted.Render("⊝")
	text := "not in PATH — skipped"
	styled := styleMuted.Render(text)
	switch outcome {
	case "skipped":
		text = "already configured — skipped"
		styled = styleMuted.Render("already has brain instructions — skipped (use --force to overwrite)")
	case "written":
		symbol = styleSuccess.Render("✓")
		text = "written → " + target
		styled = styleMuted.Render(target)
	}
	if isTTY() {
		fmt.Printf("  %s  %-8s  %s\n", symbol, agent.name, styled)
		return
	}
	fmt.Fprintf(out, "  %-8s  %s\n", agent.name, text)
}

func printSteeringSummary(out io.Writer, written, skipped, notFound []string) {
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
}
