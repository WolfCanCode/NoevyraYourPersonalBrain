<p align="center">
  <img src="site/assets/full-logo.png" alt="Noevyra — Intelligence That Evolves" width="500">
</p>

<p align="center">
  <strong>One brain. Any agent. Zero re-setup.</strong><br>
  <sub>Local-first · SQLite FTS5 · MCP stdio · 2,737 skills bundled · MIT · v0.7.3</sub>
</p>

<p align="center">
  <a href="https://github.com/WolfCanCode/personal-ai-brain/releases/latest">↓ Download</a> ·
  <a href="docs/getting-started.md">Get started</a> ·
  <a href="docs/commands.md">Commands</a> ·
  <a href="docs/mcp.md">MCP</a> ·
  <a href="docs/setup.md">brain setup</a>
</p>

---

> **Your model is not the bottleneck. Rebuilding the same context is.**

Run `brain setup` once per project. It detects your stack, searches **2,737 bundled community skills** per technology, lets you pick with arrow keys, provisions MCPs to every agent client, and persists a profile that travels with you across Claude, Codex, Cursor, and Kiro.

## Install

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
brain init                  # auto-seeds 2,737 skills
```

```powershell
# Windows
irm https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.ps1 | iex
```

Already installed? `brain update` pulls the latest release in-place.

## Quickstart

```sh
cd ~/my-project
brain setup          # detect stack → pick skills ↑↓ Space Enter → done
brain run kiro   "add rate limiting"   # any agent gets the same context
brain run claude "add rate limiting"
```

## What `brain setup` looks like

```
  ☽ ✦  brain setup
  ─────────────────────────────────────
  Languages   java  ·  typescript
  Stacks      jvm  ·  spring  +3 more
  Signals     spring  github  containers

  ⟳ Indexing skill catalog (first run)…
  ✓ 2737 skills ready

  Searching skill catalog…

  Select capabilities to provision
  ↑↓ navigate  ·  Space select  ·  Enter confirm  ·  a toggle all
  ┌─────────────────────────────────────────────────────────────┐
  │ [ ] ⚙ MCP    GitHub MCP              ⬡ network             │
  │ [ ] ⚙ MCP    Database MCP            ⬡ network             │
  │ [ ] ◆ SKILL  spring-boot-patterns    📦 42K                 │
  │ [ ] ◆ SKILL  java-concurrency        📦 18K                 │
  │ [ ] ◆ SKILL  spring-security         📦 31K                 │
  └─────────────────────────────────────────────────────────────┘

  ✓ provisioned  GitHub MCP
  ✓ provisioned  spring-boot-patterns
  ✦ Setup complete.
```

## Commands

| Command | What it does |
|---|---|
| `brain init` | Create store, auto-seed 2,737 skills |
| `brain setup` | Detect stack → pick capabilities with ↑↓ Space → persist profile |
| `brain setup --status` | Show stored profile and provisioned capabilities |
| `brain run claude\|codex\|cursor\|kiro "task"` | Launch agent with compiled context pack |
| `brain advise "task" --budget 1200` | Full intelligence pack — profile + memories + evidence |
| `brain remember "…" --type decision` | Store a verified project memory |
| `brain recall "query"` | Search verified memories |
| `brain skills find "react"` | Search 2,737 bundled skills |
| `brain sources sync` | Refresh trusted sources + update skills catalog |
| `brain update` | Self-update to latest release |
| `brain doctor` | Health check — tools, scheduler, profile |
| `brain mcp-config` | Print MCP config JSON for your agent client |

→ [Full command reference](docs/commands.md)

## MCP tools

Connect once with `brain mcp-config`, then every agent gets:

`brain_advise` · `brain_profile` · `brain_context` · `brain_recall` · `brain_remember` · `brain_feedback` · `brain_finish_task` · `brain_handoff`

→ [MCP guide](docs/mcp.md)

## Roadmap

- [x] SQLite + FTS5 memory store with MCP gateway
- [x] Token-budgeted context compiler with feedback ranking
- [x] 2,737 AAS skills embedded in binary — zero-setup catalog
- [x] Interactive TUI — arrow keys + spacebar selection with Dracula theme
- [x] Brand-colored CLI — gold `#D4A843` · navy `#7B9FCB`
- [x] Claude · Codex · Cursor · Kiro agent support
- [x] `brain update` — self-update from GitHub Releases
- [x] Automated bump-and-release CI
- [ ] Project identity handshake across MCP clients
- [ ] Session capture and automatic outcome extraction
- [ ] Cross-device sync

---

MIT © [WolfCanCode](https://github.com/WolfCanCode) · [Releases](https://github.com/WolfCanCode/personal-ai-brain/releases) · [Docs](docs/)
