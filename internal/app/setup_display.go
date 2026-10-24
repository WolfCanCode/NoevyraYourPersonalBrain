package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/intelligence"
	"github.com/charmbracelet/lipgloss"
)

func displayProfile(out io.Writer, profile intelligence.ProjectProfile) {
	if isTTY() {
		fmt.Println()
		printSectionLabel("Detected profile")
		fmt.Println()
		if len(profile.Languages) > 0 {
			printKeyValue("Languages", strings.Join(profile.Languages, "  "+styleMuted.Render("·")+"  "))
		}
		if len(profile.Stacks) > 0 {
			shown := profile.Stacks
			extra := 0
			if len(shown) > 5 {
				extra = len(shown) - 5
				shown = shown[:5]
			}
			line := strings.Join(shown, "  "+styleMuted.Render("·")+"  ")
			if extra > 0 {
				line += styleMuted.Render(fmt.Sprintf("  +%d more", extra))
			}
			printKeyValue("Stacks", line)
		}
		if len(profile.Signals) > 0 {
			signals := make([]string, len(profile.Signals))
			for i, s := range profile.Signals {
				signals[i] = lipgloss.NewStyle().Foreground(clrGold).Render(s)
			}
			printKeyValue("Signals", strings.Join(signals, "  "))
		}
		if profile.Truncated {
			fmt.Println(styleMuted.Render(fmt.Sprintf("  ⚠  scan truncated at %d files", intelligence.MaxProfileScanFiles)))
		}
		fmt.Println()
		return
	}
	// Non-TTY fallback
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
		fmt.Fprintf(out, "  (scan truncated at %d files)\n", intelligence.MaxProfileScanFiles)
	}
}

func printProvisionSummary(out io.Writer, results []intelligence.ProvisionResult) {
	if isTTY() {
		fmt.Println()
		printSectionLabel("Provisioning summary")
		fmt.Println()
		for _, r := range results {
			badge := outcomeBadge(string(r.Outcome))
			name := styleWhite.Render(r.Capability.Name)
			if r.Error != "" {
				name += styleMuted.Render("  " + r.Error)
			}
			fmt.Printf("  %s  %s\n", badge, name)
		}
		fmt.Println()
		return
	}
	fmt.Fprintln(out, "\nProvisioning summary:")
	for _, r := range results {
		status := string(r.Outcome)
		if r.Error != "" {
			status += " (" + r.Error + ")"
		}
		fmt.Fprintf(out, "  %-36s %s\n", r.Capability.Name, status)
	}
}

func printSetupDryRun(out io.Writer, capabilities setupCapabilities) {
	catalogCaps, liveGroups := capabilities.builtIn, capabilities.skills
	if isTTY() {
		printSectionLabel("[dry-run] built-in capabilities")
		fmt.Println()
		for _, cap := range catalogCaps {
			e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason}
			fmt.Println(" ", capOptionLabel(e))
		}
		offset := len(catalogCaps)
		for _, g := range liveGroups {
			if len(g.caps) == 0 {
				continue
			}
			fmt.Println(styleSection.Render("  [dry-run] " + g.label))
			for i, cap := range g.caps {
				e := capEntry{Ref: cap.Ref, Kind: cap.Kind, Name: cap.Name, Network: string(cap.Network), Reason: cap.Reason, InstallCount: cap.InstallCount}
				fmt.Printf("  %2d. %s\n", offset+i+1, capOptionLabel(e))
			}
			offset += len(g.caps)
		}
		fmt.Println(styleMuted.Render("\n  [dry-run] no files or records written."))
		return
	}
	fmt.Fprintln(out, "\n[dry-run] built-in capabilities:")
	for i, cap := range catalogCaps {
		fmt.Fprintf(out, "  %2d. [%s][%s] %s\n", i+1, cap.Kind, string(cap.Network), cap.Name)
	}
	offset := len(catalogCaps)
	for _, g := range liveGroups {
		if len(g.caps) == 0 {
			continue
		}
		fmt.Fprintf(out, "\n[dry-run] %s:\n", g.label)
		for i, cap := range g.caps {
			fmt.Fprintf(out, "  %2d. [skill][network] %s\n", offset+i+1, cap.Name)
		}
		offset += len(g.caps)
	}
	fmt.Fprintln(out, "\n[dry-run] no files or database records were written.")
}

func printSetupComplete(out io.Writer) {
	if isTTY() {
		fmt.Println(styleSuccess.Render("  ✦ Setup complete.") + styleMuted.Render("  Run 'brain setup --status' to inspect."))
		fmt.Println()
		return
	}
	fmt.Fprintln(out, "\nSetup complete. Run 'brain setup --status' to inspect the stored profile.")
}
