package app

import (
	"fmt"
	"io"
)

// helpEntry is one row in the help table.
type helpEntry struct{ cmd, desc string }

// helpGroups defines all commands grouped by category.
var helpGroups = []struct {
	name    string
	entries []helpEntry
}{
	{"Memory", []helpEntry{
		{"brain init [--schedule]", "Create store · auto-seed 2,737 skills · install daily timer"},
		{"brain remember \"text\" [--type T]", "Store a verified project memory"},
		{"brain recall \"query\"", "Full-text search over verified memories"},
		{"brain context \"task\"", "Generate a simple context pack"},
		{"brain approve <id>", "Promote an agent-proposed candidate to verified"},
		{"brain status", "Memory counts, scheduler state, active knowledge"},
		{"brain digest [--days N]", "Recent memories and event log"},
	}},
	{"Intelligence", []helpEntry{
		{"brain profile", "Detect project stack from bounded evidence (read-only)"},
		{"brain advise \"task\" [--budget N]", "Full intelligence pack: profile + memories + recommendations"},
		{"brain feedback <id> --outcome …", "Record pack usefulness · adjusts future ranking"},
	}},
	{"Setup & Agents", []helpEntry{
		{"brain setup [--refresh|--dry-run|--status]", "Detect stack, provision capabilities, write agent steering files"},
		{"brain run claude|codex|cursor|kiro|opencode|pi", "Launch agent with compiled context pack"},
	}},
	{"Sources & Skills", []helpEntry{
		{"brain sources seed|add|list|sync", "Manage explicitly trusted knowledge sources"},
		{"brain skills find \"query\"", "Search 2,737 bundled AAS skills (also npx fallback)"},
		{"brain skills trust owner/repo@skill", "Approve a reviewed community skill"},
		{"brain skills list", "Show all trusted skills"},
		{"brain sync [--sources|--skills]", "Local maintenance · optional source/skill refresh"},
	}},
	{"System", []helpEntry{
		{"brain steer [--global] [--dry-run]", "Write brain auto-memory rules into every agent steering file"},
		{"brain update [--check]", "Self-update from latest GitHub Release"},
		{"brain mcp", "Start MCP server over stdio"},
		{"brain mcp-config", "Print agent MCP configuration JSON"},
		{"brain doctor", "Check prerequisites, tools, and stored profile"},
		{"brain version", "Build metadata"},
	}},
}

// renderHelp writes the formatted help to out.
// In TTY mode it uses gold section headers and muted descriptions.
// In non-TTY mode it outputs plain text with consistent 46-char left column.
func renderHelp(out io.Writer) {
	const cmdWidth = 46
	if isTTY() {
		for _, g := range helpGroups {
			fmt.Fprintln(out)
			fmt.Fprintln(out, styleAccent.Render("  "+g.name))
			for _, e := range g.entries {
				cmd := styleWhite.Render(fmt.Sprintf("  %-*s", cmdWidth, e.cmd))
				desc := styleMuted.Render(e.desc)
				fmt.Fprintf(out, "%s %s\n", cmd, desc)
			}
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, styleMuted.Render("  BRAIN_HOME overrides ~/.brain · Skills require npx"))
		fmt.Fprintln(out)
		return
	}
	// Plain text fallback.
	fmt.Fprintln(out, "Noevyra — intelligence that evolves")
	fmt.Fprintln(out)
	for _, g := range helpGroups {
		fmt.Fprintf(out, "  %s:\n", g.name)
		for _, e := range g.entries {
			fmt.Fprintf(out, "  %-*s  %s\n", cmdWidth, e.cmd, e.desc)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "  BRAIN_HOME overrides ~/.brain. Skills require npx.")
}
