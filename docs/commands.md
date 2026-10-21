# Command reference

All commands read from `$BRAIN_HOME` (default `~/.brain/brain.sqlite`). Set `BRAIN_HOME` to use a different location.

---

## brain init

```
brain init [--schedule]
```

Creates the SQLite store and prints the MCP configuration JSON.

`--schedule` — registers a daily maintenance + source-refresh timer:
- **macOS**: user `launchd` LaunchAgent at `08:00`
- **Linux**: user `systemd` timer with `Persistent=true`
- **Windows**: not yet automated; run `brain sync --sources` manually

---

## brain setup

```
brain setup [--refresh] [--dry-run] [--status]
```

Detects the current project's stack from bounded repository metadata, discovers best-practice capabilities from the catalog, prompts for your selection, provisions approved items, and persists the profile.

| Flag | Effect |
|---|---|
| _(none)_ | Full setup run; prompts before overwriting an existing profile |
| `--refresh` | Re-run detection without the overwrite prompt |
| `--dry-run` | Print what would be detected and provisioned; write nothing |
| `--status` | Display the stored profile — languages, stacks, signals, provisioned capabilities, timestamp |

**Project detection** scans up to 4000 files, skipping `node_modules`, `vendor`, `dist`, `.git`, etc. It parses root-level manifests (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `requirements.txt`, `pom.xml`, `build.gradle`) to extract dependency names into the `stacks` field.

**Capability types** discovered:

| Kind | What is provisioned |
|---|---|
| `mcp` | Writes an `mcpServers` entry to every detected agent's config file |
| `skill` | Prompts for per-skill approval, then registers in the local skill trust store |
| `cli` | Checks whether the tool is in `PATH`; reports if manual install is needed |

---

## brain remember

```
brain remember "text" [--type TYPE] [--scope SCOPE]
```

Store a verified memory directly from the CLI.

`--type` — one of: `decision` · `identity` · `semantic` · `episodic` · `project` · `procedural`  
`--scope` — `project` (default, scoped to current repo) or `personal` (available across all projects)

---

## brain recall

```
brain recall "query" 
```

Full-text search across verified memories scoped to the current project and personal memories.

---

## brain context

```
brain context "task"
```

Returns verified memories relevant to a task as a simple JSON pack. For the full intelligence pack with profile and capability recommendations, use `brain advise`.

---

## brain advise

```
brain advise "task" [--budget N]
```

Compiles a full intelligence pack:
- Project profile (languages, stacks, signals)
- Reasoning frame (approach, risks, verification gates)
- Ranked context items (memories + trusted source excerpts)
- Capability recommendations (skills / MCPs / CLIs) with confidence scores

`--budget N` — token budget (128–16000; default adapts from past feedback outcomes).

Returns a JSON pack with a stable `id` you can pass to `brain feedback`.

---

## brain feedback

```
brain feedback <pack-id> --outcome helpful|partial|harmful [--tokens N] [--notes "…"]
```

Records the pack outcome and adjusts quality scores for knowledge items in that pack. After ≥3 helpful outcomes the compiler adapts its default token budget.

---

## brain profile

```
brain profile
```

Reads the project without writing anything. Returns detected languages, stacks, signals, manifests, and file counts as JSON.

---

## brain run

```
brain run claude|codex|cursor|kiro [task]
```

Compiles a context pack, writes it to a `0600` temp file, and starts the agent with the file path in `BRAIN_CONTEXT_FILE`. The file is deleted when the agent exits.

If the agent binary is not found in `PATH`, `brain run` prints a specific install message and exits without starting anything.

---

## brain sources

```
brain sources seed
brain sources add URL --name NAME [--tags go,security]
brain sources list
brain sources sync
brain sources pause <source-id>
```

Manage the set of HTTPS sources you explicitly trust. `sync` does a conditional fetch (respects `ETag` / `Last-Modified`) and splits content into searchable chunks with provenance metadata.

`seed` registers the bundled starter catalog:
- Go code review comments
- OWASP secure coding with AI
- MCP security best practices

---

## brain skills

```
brain skills find "query"
brain skills trust owner/repo@skill --keyword TERM
brain skills list
```

`find` calls the external `skills.sh` CLI (requires `npx`). Review the skill source before trusting.  
`trust` adds the skill to the local trust store. Brain matches it against tasks by the keyword you provide.

---

## brain approve

```
brain approve <memory-id>
```

Promotes an agent-proposed candidate memory to verified status. Candidates are created when an agent calls `brain_remember` through MCP.

---

## brain update

```
brain update [--check]
```

`--check` — report the latest release version and whether it is newer than the installed binary; do not install.

Without `--check` — downloads the matching release archive for the current OS/arch, verifies the SHA-256 checksum, extracts the binary, and atomically replaces the running executable. Restart any running `brain mcp` processes to pick up the new version.

---

## brain sync

```
brain sync [--sources] [--skills] [--online]
```

Runs local maintenance (expires stale memories, records a digest event).

`--sources` — also fetches registered sources with conditional requests.  
`--skills` — also runs `npx skills check`.  
`--online` — alias for `--sources --skills`.

---

## brain digest

```
brain digest [--days N]
```

Returns recent events and memories. Default: 1 day. Range: 1–3650 days.

---

## brain mcp

```
brain mcp
```

Starts the MCP server over stdio. Connect your agent client to the absolute path of the binary with `args: ["mcp"]`. See [MCP integration](mcp.md).

---

## brain mcp-config

```
brain mcp-config
```

Prints the JSON block to paste into your agent's MCP configuration.

---

## brain doctor

```
brain doctor
```

Reports:
- Installed version, Go version, platform
- Store path
- Presence in `PATH` of: `git`, `claude`, `codex`, `cursor-agent`, `kiro`, `npx`
- Scheduler status
- Whether a stored project profile exists for the current directory

---

## brain status

```
brain status
```

Reports memory counts (total, verified, candidates), last sync time, active knowledge items, and project identity.

---

## brain version

```
brain version
```

Prints version, commit SHA, build timestamp, Go version, and platform as JSON.
