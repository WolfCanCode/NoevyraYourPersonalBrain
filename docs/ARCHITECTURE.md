# Architecture

**Store:** `${BRAIN_HOME:-~/.brain}/brain.sqlite` — single-writer WAL, FTS5 virtual tables. Source code is public; runtime data is never committed.

---

## Package layout

```
cmd/brain/              — entry point; version/commit/builtAt injected via ldflags
internal/
  app/
    cli.go              — command dispatch, flag parsing
    setup.go            — brain setup: detection → TUI select → provision → persist
    update.go           — brain update: GitHub release download, SHA-256, atomic replace
    mcp.go              — JSON-RPC 2.0 stdio MCP server
    ui.go               — lipgloss styles, huh TUI helpers, badge renderers, isTTY()
    scheduler.go        — macOS launchd / Linux systemd timer
  intelligence/
    profile.go          — bounded project profiling (language, stack, signal detection)
    provision.go        — ProvisionCatalog, DiscoverCapabilities, Provision, writeMCPEntry
    aas.go              — SearchSkillsCatalog, CatalogSeeded (FTS5 over skill items)
    embed.go            — //go:embed skills_index.json, SeedEmbeddedCatalog
    skills_index.json   — 2 737 AAS skills, bundled at build time, refreshed in CI
    compiler.go         — token-budgeted context compiler (Compile, Feedback)
    source.go           — trusted source sync, chunksFromSkillsIndex JSON parser
  memory/
    store.go            — schema v3, FTS5, Recall, UpsertProfile, Trust, Skills
```

---

## Data flow: `brain setup`

```
cwd
 │  projectRoot() — walk ≤10 ancestors
 ▼
Project root
 │  intelligence.Profile() — scan ≤4 000 files, parse manifests
 ▼
ProjectProfile { Languages, Stacks, Signals }
 │
 ├── SeedEmbeddedCatalog() — one-time import of 2 737 embedded skills
 │
 ├── DiscoverCapabilities() — built-in MCP catalog matched to profile signals
 │
 ├── SearchSkillsCatalog() × N keywords — FTS5 search, top 5 per technology
 │
 ├── huh.NewMultiSelect() — interactive TUI (arrow + space + enter) or text fallback
 │
 ├── Provision() per selected capability:
 │     mcp   → writeMCPEntry() atomic write to ~/.claude.json / ~/.kiro/settings/mcp.json / …
 │     skill → per-skill approval prompt → store.Trust()
 │     cli   → exec.LookPath()
 │
 └── store.UpsertProfile() — persists profile JSON + ProvisionResult[] (retains last 3)
```

## Data flow: `brain run <agent>`

```
store.LatestProfile()  →  stored ProjectProfile (or detect on-the-fly)
intelligence.Compile() →  context pack: memories + knowledge + recommendations
os.MkdirTemp()         →  session dir 0700
os.WriteFile(0600)     →  context.md with pack JSON
exec.Command(agent)    →  BRAIN_CONTEXT_FILE=<path>
defer os.RemoveAll()   →  cleaned up on agent exit regardless of exit code
```

---

## Schema v3

```sql
memory              — verified/candidate project and personal memories (FTS5)
events              — audit log for tasks, handoffs, sync, setup sessions
settings            — key/value (last_sync, etc.)
approved_skills     — trusted skill refs with keywords
knowledge_sources   — registered HTTPS sources (AAS catalog + user sources)
knowledge_items     — chunked source content with quality scores (FTS5)
context_packs       — compiled packs with token metrics
context_pack_items  — pack → item links for feedback scoring
project_profiles    — persisted ProjectProfile + ProvisionResult JSON (≤3 per project)
```

---

## Trust classes

| Input | Trust | Action allowed |
|---|---|---|
| `brain remember` CLI | User-approved | Verified write |
| `brain_remember` via MCP | Untrusted | Candidate — requires `brain approve` |
| `brain setup` capability selection | User-approved per item | MCP config write, skill trust |
| Embedded `skills_index.json` | Bundled at build, AAS upstream | Read-only catalog — no execution |
| Registered HTTPS source | Trusted provenance, untrusted content | Bounded text, attributed context |
| Git working tree | May be attacker-controlled | Bounded read-only profiling |

---

## Embedded catalog lifecycle

```
Build time:   CI fetches latest skills_index.json → //go:embed → binary
First run:    brain init calls SeedEmbeddedCatalog() → 2 737 rows in knowledge_items
Refresh:      brain sources sync → conditional HTTP fetch → updates SQLite
Next release: CI fetches again → new binary ships newer catalog
```

---

## Related decisions

- [ADR 0001](adr/0001-native-go-runtime.md) — Why Go
- [ADR 0002](adr/0002-intelligence-substrate.md) — Intelligence substrate design
- [ADR 0003](adr/0003-external-structural-intelligence.md) — External structural intelligence
