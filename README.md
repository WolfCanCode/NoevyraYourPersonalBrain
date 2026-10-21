<p align="center">
  <img src="assets/full-logo.png" alt="Noevyra — Intelligence That Evolves" width="480">
</p>

<p align="center">
  <strong>One brain. Any agent. Zero re-setup.</strong><br>
  <sub>Local-first · SQLite FTS5 · MCP stdio · MIT · v0.7.0</sub>
</p>

<p align="center">
  <a href="https://github.com/WolfCanCode/personal-ai-brain/releases/latest">Download</a> ·
  <a href="docs/getting-started.md">Get started</a> ·
  <a href="docs/commands.md">Commands</a> ·
  <a href="docs/mcp.md">MCP</a> ·
  <a href="docs/setup.md">brain setup</a>
</p>

---

> **Your model is not the bottleneck. Rebuilding the same context is.**

Run `brain setup` once per project — it detects your stack, picks relevant skills and MCPs from a catalog of **2 737 community skills** (bundled in the binary, no sync needed), lets you choose with arrow keys, and persists a profile that travels with you across Claude, Codex, Cursor, and Kiro.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
brain init
```

**Windows:**
```powershell
irm https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.ps1 | iex
```

Already installed? `brain update` pulls the latest release in-place.

## Quickstart

```sh
cd ~/my-project

brain setup          # detect stack → pick skills with ↑↓ Space Enter → done
brain run kiro "add rate limiting"   # launch any agent with ready context
brain run claude "add rate limiting" # same brain, different agent
```

That's it. Switch agents without re-explaining the project.

## What happens in `brain setup`

```
  ✦ Noevyra · brain setup
  ────────────────────────
  Languages   go  ·  typescript
  Signals     spring  ·  github  ·  containers

  Searching AAS skill catalog…

  Available capabilities:
  ┌──────────────────────────────────────────────────────┐
  │ > ⚙ MCP    GitHub MCP              ⬡ network        │
  │   ◆ SKILL  spring-boot-patterns    📦 42K            │
  │   ◆ SKILL  java-concurrency        📦 18K            │
  │   ▸ CLI    ripgrep                 ⬡ local           │
  └──────────────────────────────────────────────────────┘
    ↑↓ navigate  ·  Space select  ·  Enter confirm

  ✓ provisioned  GitHub MCP
  ✓ provisioned  spring-boot-patterns
  ✦ Setup complete.
```

## Commands

| Command | What it does |
|---|---|
| `brain init` | Create the SQLite store (auto-seeds 2 737 skills) |
| `brain setup` | Detect project, pick capabilities with arrow keys, persist profile |
| `brain setup --status` | Show stored profile and provisioned capabilities |
| `brain run claude\|codex\|cursor\|kiro "task"` | Launch agent with pre-built context pack |
| `brain advise "task" --budget 1200` | Full intelligence pack — profile + memories + evidence + recommendations |
| `brain remember "…" --type decision` | Store a verified project memory |
| `brain recall "query"` | Search verified memories |
| `brain skills find "react"` | Search 2 737 bundled skills by keyword |
| `brain skills trust owner/repo@skill --keyword kw` | Approve a skill |
| `brain sources sync` | Refresh trusted sources + update skills catalog from upstream |
| `brain update` | Self-update to latest release |
| `brain update --check` | Check for update without installing |
| `brain doctor` | Health check — tools, scheduler, profile status |
| `brain mcp` | Start MCP server over stdio |
| `brain mcp-config` | Print MCP config JSON for Claude/Codex/Cursor/Kiro |

→ [Full command reference](docs/commands.md)

## MCP tools

Connect once (`brain mcp-config` prints the JSON), then every agent session gets:

`brain_advise` · `brain_profile` · `brain_context` · `brain_recall` · `brain_remember` · `brain_feedback` · `brain_finish_task` · `brain_handoff`

→ [MCP integration guide](docs/mcp.md)

## Architecture

```
Claude · Codex · Cursor · Kiro
        │
   brain run / brain mcp
        │
  Intelligence layer ──── ProvisionCatalog (2 737 skills embedded)
        │                          │
  Memory layer ────────── brain.sqlite (WAL · FTS5)
```

→ [Architecture docs](docs/ARCHITECTURE.md)  ·  [Security](docs/security.md)

## Roadmap

- [x] SQLite + FTS5 memory store with MCP gateway
- [x] Token-budgeted context compiler with feedback ranking
- [x] Daily trusted-source refresh with provenance
- [x] Auto-project setup — `brain setup` with persistent profiles
- [x] 2 737 AAS skills embedded in binary — zero-setup catalog
- [x] Interactive TUI — arrow keys + space bar selection
- [x] Badge-style CLI — kind · network · install count
- [x] Claude · Codex · Cursor · **Kiro** agent support
- [x] `brain update` — self-update from GitHub Releases
- [x] Automated bump-and-release CI
- [ ] Project identity handshake across MCP clients
- [ ] Session capture and automatic outcome extraction
- [ ] Cross-device sync

---

MIT © [WolfCanCode](https://github.com/WolfCanCode) · [Releases](https://github.com/WolfCanCode/personal-ai-brain/releases) · [Docs](docs/)
