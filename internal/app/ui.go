package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// isTTY returns true when stdout is a real terminal (not a pipe/test/CI).
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// ── Brand palette — derived from the Noevyra logo ─────────────────────────
//
//   Gold   #D4A843  (crescent moon, hair highlights, wordmark)
//   Navy   #7B9FCB  (face silhouette — lightened for dark-bg readability)
//   Dark gold bg    #2A1C00
//   Dark navy bg    #0D1F3A

var (
	clrGold    = lipgloss.Color("#D4A843") // primary accent — logo gold
	clrGoldDim = lipgloss.Color("#9B7A2A") // dim gold for secondary hints
	clrNavy    = lipgloss.Color("#7B9FCB") // lightened logo navy — readable on dark
	clrGreen   = lipgloss.Color("#7ECE94") // success
	clrRed     = lipgloss.Color("#FF6B6B") // error
	clrAmber   = lipgloss.Color("#E8A545") // warm amber (network / install badges)
	clrMuted   = lipgloss.Color("#7A7A8E") // muted text
	clrWhite   = lipgloss.Color("#F4F2EE") // primary text
)

// ── Base styles ────────────────────────────────────────────────────────────

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleMuted   = lipgloss.NewStyle().Foreground(clrMuted)
	styleSuccess = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	styleError   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(clrAmber)
	styleAccent  = lipgloss.NewStyle().Foreground(clrGold).Bold(true)
	styleNavy    = lipgloss.NewStyle().Foreground(clrNavy)
	styleWhite   = lipgloss.NewStyle().Foreground(clrWhite)

	styleHeader = lipgloss.NewStyle().
			Foreground(clrGold).
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#3A3020")).
			PaddingBottom(1).
			MarginBottom(1)

	styleSection = lipgloss.NewStyle().
			Foreground(clrGoldDim).
			Bold(true).
			MarginTop(1)

	styleKey   = lipgloss.NewStyle().Foreground(clrMuted).Width(12)
	styleValue = lipgloss.NewStyle().Foreground(clrWhite)
)

// ── Badge renderers ────────────────────────────────────────────────────────

func kindBadge(kind string) string {
	switch kind {
	case "mcp":
		return lipgloss.NewStyle().
			Foreground(clrGold).Bold(true).
			Background(lipgloss.Color("#2A1C00")).
			Padding(0, 1).Render("⚙ MCP")
	case "skill":
		return lipgloss.NewStyle().
			Foreground(clrNavy).Bold(true).
			Background(lipgloss.Color("#0D1F3A")).
			Padding(0, 1).Render("◆ SKILL")
	case "cli":
		return lipgloss.NewStyle().
			Foreground(clrAmber).Bold(true).
			Background(lipgloss.Color("#1E1200")).
			Padding(0, 1).Render("▸ CLI")
	default:
		return styleMuted.Render(kind)
	}
}

func networkBadge(network string) string {
	if network == "local" {
		return lipgloss.NewStyle().Foreground(clrGreen).Render("⬡ local")
	}
	return lipgloss.NewStyle().Foreground(clrAmber).Render("⬡ network")
}

func installBadge(count string) string {
	if count == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(clrGold).Bold(true).Render("📦 " + count)
}

func riskBadge(risk string) string {
	switch strings.ToLower(risk) {
	case "critical":
		return styleError.Render("🔴 critical")
	case "safe":
		return styleSuccess.Render("🟢 safe")
	case "none":
		return styleMuted.Render("○ none")
	default:
		return ""
	}
}

func outcomeBadge(outcome string) string {
	switch outcome {
	case "provisioned":
		return styleSuccess.Render("✓ provisioned")
	case "failed":
		return styleError.Render("✗ failed")
	case "skipped":
		return styleMuted.Render("⊝ skipped")
	default:
		return styleMuted.Render(outcome)
	}
}

func checkBadge(ok bool) string {
	if ok {
		return styleSuccess.Render("✓")
	}
	return styleError.Render("✗")
}

// ── Capability line formatters ─────────────────────────────────────────────

type capEntry struct {
	Ref          string
	Kind         string
	Name         string
	Network      string
	Reason       string
	InstallCount string
}

func capOptionLabel(cap capEntry) string {
	badge := kindBadge(cap.Kind)
	name := styleWhite.Width(44).Render(func() string {
		n := cap.Name
		if len(n) > 44 {
			return n[:41] + "…"
		}
		return n
	}())
	suffix := networkBadge(cap.Network)
	if cap.InstallCount != "" {
		suffix = installBadge(cap.InstallCount)
	}
	return fmt.Sprintf("%s  %s  %s", badge, name, suffix)
}

func capDetailLine(reason string) string {
	if reason == "" {
		return ""
	}
	r := reason
	if len(r) > 72 {
		r = r[:69] + "…"
	}
	return styleMuted.Render("        " + r)
}

// ── Shared print helpers ──────────────────────────────────────────────────

func printHeader(title string) {
	fmt.Println(styleHeader.Render(
		lipgloss.NewStyle().Foreground(clrGoldDim).Render("  ☽ ✦  ") +
			styleAccent.Render(title),
	))
}

func printKeyValue(key, value string) {
	fmt.Printf("%s %s\n", styleKey.Render(key), styleValue.Render(value))
}

func printSectionLabel(label string) {
	fmt.Println(styleSection.Render("  " + label))
}

// printBanner shows the Noevyra brand banner on TTY — used by brain init
// and brain help. Uses only universally-supported Unicode so it renders
// cleanly in every terminal and font without misalignment.
func printBanner() {
	gold := lipgloss.NewStyle().Foreground(clrGold).Bold(true)
	dim := lipgloss.NewStyle().Foreground(clrGoldDim)
	mute := styleMuted
	navy := styleNavy

	line := dim.Render("  ") + gold.Render("─────────────────────────────────")

	fmt.Println()
	fmt.Printf("  %s  %s\n",
		gold.Render("☽ ✦"),
		gold.Render("N O E V Y R A"),
	)
	fmt.Println(line)
	fmt.Printf("     %s\n", navy.Render("Intelligence that evolves."))
	fmt.Println(line)
	fmt.Printf("  %s\n", mute.Render("Local · Private · Portable · MIT"))
	fmt.Println()
}
