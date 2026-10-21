package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// isTTY returns true when stdout is a real terminal (not a pipe/test/CI).
// lipgloss also auto-detects this and strips color codes when false.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// ── Palette ────────────────────────────────────────────────────────────────

var (
	clrPurple = lipgloss.Color("#c7b4ff")
	clrGreen  = lipgloss.Color("#b4e8bf")
	clrRed    = lipgloss.Color("#ff6b6b")
	clrYellow = lipgloss.Color("#fcd34d")
	clrBlue   = lipgloss.Color("#7dd3fc")
	clrOrange = lipgloss.Color("#fb923c")
	clrMuted  = lipgloss.Color("#888899")
	clrWhite  = lipgloss.Color("#f4f2ee")
	clrPanel  = lipgloss.Color("#19191f")
)

// ── Base styles ────────────────────────────────────────────────────────────

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleMuted   = lipgloss.NewStyle().Foreground(clrMuted)
	styleSuccess = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	styleError   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(clrYellow)
	styleAccent  = lipgloss.NewStyle().Foreground(clrPurple).Bold(true)
	styleWhite   = lipgloss.NewStyle().Foreground(clrWhite)

	styleHeader = lipgloss.NewStyle().
			Foreground(clrPurple).
			Bold(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#303039")).
			PaddingBottom(1).
			MarginBottom(1)

	styleSection = lipgloss.NewStyle().
			Foreground(clrMuted).
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
			Foreground(clrPurple).Bold(true).
			Background(lipgloss.Color("#2d1f4e")).
			Padding(0, 1).Render("⚙ MCP")
	case "skill":
		return lipgloss.NewStyle().
			Foreground(clrBlue).Bold(true).
			Background(lipgloss.Color("#1a2a40")).
			Padding(0, 1).Render("◆ SKILL")
	case "cli":
		return lipgloss.NewStyle().
			Foreground(clrYellow).Bold(true).
			Background(lipgloss.Color("#2a2010")).
			Padding(0, 1).Render("▸ CLI")
	default:
		return styleMuted.Render(kind)
	}
}

func networkBadge(network string) string {
	if network == "local" {
		return lipgloss.NewStyle().Foreground(clrGreen).Render("⬡ local")
	}
	return lipgloss.NewStyle().Foreground(clrOrange).Render("⬡ network")
}

func installBadge(count string) string {
	if count == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(clrYellow).
		Bold(true).Render("📦 " + count)
}

func riskBadge(risk string) string {
	switch strings.ToLower(risk) {
	case "critical":
		return lipgloss.NewStyle().Foreground(clrRed).Render("🔴 critical")
	case "safe":
		return lipgloss.NewStyle().Foreground(clrGreen).Render("🟢 safe")
	case "none":
		return lipgloss.NewStyle().Foreground(clrMuted).Render("○ none")
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

// ── Formatted capability line (for huh option labels and non-TTY lists) ──

// capOptionLabel returns a single-line label for a capability,
// used both in huh multi-select and the fallback text list.
func capOptionLabel(cap capEntry) string {
	badge := kindBadge(cap.Kind)
	net := networkBadge(string(cap.Network))

	// Name: trim long refs to keep it readable
	name := cap.Name
	if len(name) > 44 {
		name = name[:41] + "…"
	}
	styledName := styleWhite.Width(44).Render(name)

	suffix := net
	if cap.InstallCount != "" {
		suffix = installBadge(cap.InstallCount)
	}
	return fmt.Sprintf("%s  %s  %s", badge, styledName, suffix)
}

// capDetailLine returns the description line shown below the option label.
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

// ── capEntry wraps both ProvisionableCapability fields + install count ────
// (used by ui.go to avoid importing intelligence directly here,
//  actual conversion happens in setup.go)

type capEntry struct {
	Ref          string
	Kind         string
	Name         string
	Network      string
	Reason       string
	InstallCount string
}

// ── Shared print helpers ──────────────────────────────────────────────────

func printHeader(title string) {
	fmt.Println(styleHeader.Render("  ✦ Noevyra · " + title))
}

func printKeyValue(key, value string) {
	fmt.Printf("%s %s\n", styleKey.Render(key), styleValue.Render(value))
}

func printSectionLabel(label string) {
	fmt.Println(styleSection.Render("  " + label))
}

func printDivider() {
	fmt.Println(styleMuted.Render(strings.Repeat("─", 60)))
}
