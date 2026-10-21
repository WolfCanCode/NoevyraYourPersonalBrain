# Getting started

Noevyra ships as a single binary. No Node, no Docker, no cloud account, no manual catalog setup.

## Install

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

**Windows:**
```powershell
irm https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.ps1 | iex
$env:Path = "$HOME\.local\bin;$env:Path"
```

**From source** (Go ≥ 1.26):
```sh
go build -o build/brain ./cmd/brain
```

## First run

```sh
brain init
```

Creates `~/.brain/brain.sqlite` and automatically seeds the embedded catalog of **2 737 community skills** — no separate `sources sync` step needed.

## Set up your first project

```sh
cd ~/my-project
brain setup
```

Noevyra:
1. Detects your stack (languages, frameworks, signals) from the project files
2. Searches the bundled skill catalog for relevant matches per technology
3. Shows an interactive list — **arrow keys** to navigate, **Space** to select, **Enter** to confirm
4. Provisions selected MCPs to your agent config files and registers approved skills
5. Saves the profile to SQLite so any agent can read it without re-detection

## Launch an agent

```sh
brain run kiro  "add rate limiting to the payments endpoint"
brain run claude "add rate limiting to the payments endpoint"
brain run codex  "add rate limiting to the payments endpoint"
```

Each command compiles a context pack (profile + memories + evidence + recommendations), writes it to a `0600` temp file, and starts the agent. Switch agents freely — the brain stays the same.

## Store decisions

```sh
brain remember "All API responses use camelCase" --type decision
brain remember "We use PostgreSQL 15 with pgx/v5" --type decision
```

These are immediately verified and available to all future packs and agent sessions.

## Daily workflow

```sh
# Get context for a task
brain advise "refactor the payment service" --budget 1200

# After the task, feed back the outcome
brain feedback <pack-id> --outcome helpful --tokens 340

# Keep sources and catalog fresh
brain sources sync

# Stay on the latest binary
brain update --check
brain update
```

## Next steps

- [Commands reference](commands.md)
- [brain setup deep dive](setup.md)
- [MCP integration](mcp.md)
- [Security model](security.md)
