# brain setup

One command — detect, discover, select, provision, persist.

```sh
brain setup
```

---

## What it does

| Step | What happens |
|---|---|
| **Detect** | Walks up to 10 ancestor dirs for `.git`, scans ≤ 4 000 files, parses manifests |
| **Profile** | Derives languages, framework signals (angular, spring, django…), operational signals |
| **Discover** | Searches 2 737 embedded skills + built-in MCP catalog — **per tech**, top 5 each |
| **Select** | Interactive TUI: ↑↓ navigate · Space select · Enter confirm · a toggle all |
| **Provision** | Writes MCP config to every detected agent; prompts per-skill approval |
| **Persist** | Saves profile to SQLite (last 3 per project); any `brain run` reads it instantly |

---

## Flags

| Flag | Effect |
|---|---|
| `--status` | Show stored profile and provisioned capabilities |
| `--refresh` | Re-run detection without the overwrite prompt |
| `--dry-run` | Preview what would be detected and provisioned — writes nothing |
| `--no-live-skills` | Skip the skill catalog search (fast, offline) |

---

## What the interactive TUI looks like

```
  ✦ Noevyra · brain setup
  ────────────────────────
  Languages   java  ·  typescript
  Stacks      jvm  ·  spring  +3 more
  Signals     spring  ·  github  ·  containers

  Searching AAS skill catalog…

  Available capabilities:

  Built-in recommendations:
   ⚙ MCP    GitHub MCP              ⬡ network
   ⚙ MCP    Documentation MCP       ⬡ network
   ⚙ MCP    Database MCP            ⬡ network

  From AAS catalog — spring (top 5):
   ◆ SKILL  spring-boot-patterns    📦 42K
   ◆ SKILL  spring-security         📦 31K
   ◆ SKILL  spring-data-jpa         📦 28K

  From AAS catalog — java (top 5):
   ◆ SKILL  java-concurrency        📦 18K
   ◆ SKILL  java-clean-code         📦 12K

  ↑↓ navigate  ·  Space select  ·  Enter confirm  ·  a toggle all
```

---

## Capability catalog

Every release embeds the full [AAS skills catalog](https://github.com/sickn33/agentic-awesome-skills) (~2 737 skills). Searches are per-technology so a Java/Spring project gets Spring and Java results, not generic TypeScript suggestions.

Built-in MCP entries always shown when signals match:

| Capability | Triggered by |
|---|---|
| GitHub MCP | `github`, `ci-cd`, `github-actions` |
| Documentation MCP | `go`, `python`, `angular`, `react`, `api`… |
| Database MCP | `database`, `sql`, `postgres`, `sqlite`… |
| Browser MCP | `web`, `angular`, `react`, `e2e`, `playwright` |
| Figma MCP | `figma`, `design`, `storybook` |

---

## MCP provisioning

When you select an MCP capability, Brain writes to every agent it finds in `PATH`:

| Agent | Binary | Config file |
|---|---|---|
| Claude | `claude` | `~/.claude.json` |
| Codex | `codex` | `~/.codex/config.json` |
| Cursor | `cursor-agent` | `~/.cursor/mcp.json` |
| Kiro | `kiro` | `~/.kiro/settings/mcp.json` |

Writes are atomic and idempotent — running `brain setup` again updates in place.

---

## Switching agents

The profile is stored in SQLite, not in any agent session. After setup:

```sh
brain run kiro "add search endpoint"
# or
brain run claude "add search endpoint"
```

Both get the same profile and capability context.

---

## Refreshing after major changes

```sh
brain setup --refresh   # re-detect, no overwrite prompt
brain setup --status    # inspect current stored profile
```
