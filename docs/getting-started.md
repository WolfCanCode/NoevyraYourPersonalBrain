# Getting started

Noevyra ships as a single Go binary. No Node, no Docker, no cloud account.

## Install

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.sh | sh
export PATH="$HOME/.local/bin:$PATH"
```

The installer detects your OS and architecture, downloads the matching release archive from GitHub, verifies the SHA-256 checksum against `SHA256SUMS`, and places `brain` in `~/.local/bin`. No `sudo` required.

**Options:**

```sh
# Pin a specific release
BRAIN_VERSION=v0.3.0 curl -fsSL .../install.sh | sh

# Install to a custom directory
BRAIN_INSTALL_DIR=~/bin curl -fsSL .../install.sh | sh
```

### Windows

```powershell
irm https://raw.githubusercontent.com/WolfCanCode/personal-ai-brain/main/install.ps1 | iex
$env:Path = "$HOME\.local\bin;$env:Path"
```

### Build from source

Requires Go ≥ 1.26.

```sh
git clone https://github.com/WolfCanCode/personal-ai-brain.git
cd personal-ai-brain
go build -o build/brain ./cmd/brain
export PATH="$PWD/build:$PATH"
```

### Verify installation

```sh
brain version
brain doctor
```

`brain doctor` checks that Git and any agent CLIs are accessible and confirms the store path.

---

## First use

### 1. Initialise the store

```sh
brain init
```

Creates `~/.brain/brain.sqlite` — your private, local memory store. Nothing is sent anywhere.

### 2. Set up your project

```sh
cd ~/projects/my-api
brain setup
```

Brain scans the repository (bounded to 4000 files), detects your languages, stacks and signals, then presents a ranked list of recommended skills, MCPs and CLI tools for your project type. Select what to provision and Brain writes the MCP configuration to every agent client it detects installed.

The resulting profile is saved to the local database. The next time you switch agents or resume a session, the profile is already there — no re-detection needed.

### 3. Store a decision

```sh
brain remember "All API responses use camelCase" --type decision
brain remember "We use PostgreSQL 15 with pgx/v5" --type decision
```

User-entered memories are immediately verified. Agent-proposed memories via MCP are held as candidates until you approve them with `brain approve <id>`.

### 4. Launch an agent

```sh
brain run kiro "Add pagination to the /orders endpoint"
brain run claude "Add pagination to the /orders endpoint"
```

Brain compiles a context pack (project profile + matching memories + trusted source excerpts + capability recommendations), writes it to a `0600` temporary file, and starts the agent with the file path in `BRAIN_CONTEXT_FILE`. The file is deleted when the agent exits.

---

## Daily workflow

```sh
# Morning: sync trusted sources (if you use them)
brain sync --sources

# Before a task: get a quick context pack
brain advise "refactor the payment service" --budget 1200

# After a task: record the outcome
brain feedback <pack-id> --outcome helpful --tokens 340

# End of day: review recent activity
brain digest --days 1
```

### Set up daily auto-refresh (macOS / Linux)

```sh
brain sources seed        # register the bundled Go, OWASP, MCP source catalog
brain sources sync        # initial pull
brain init --schedule     # register a daily timer at 08:00 local time
```

The timer runs `brain sync --sources` — local maintenance plus a conditional fetch of your registered sources. It never installs or executes anything.

---

## Keep Brain up to date

```sh
brain update          # download and install the latest release
brain update --check  # check without installing
```

---

## Next steps

- [Commands reference](commands.md) — every flag and option
- [Brain setup in depth](setup.md) — capability discovery and provisioning
- [MCP integration](mcp.md) — connecting Claude, Codex, Cursor, Kiro
