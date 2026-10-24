# Architecture

**Store:** `${BRAIN_HOME:-~/.brain}/brain.sqlite` — single-writer WAL, FTS5 virtual tables. Source code is public; runtime data is never committed.

---

## Package layout

```
cmd/brain/              — entry point; version/commit/builtAt injected via ldflags
cmd/package/            — archive and checksum a tested native executable
internal/
  app/
    cli.go              — command dispatch and store lifecycle
    args.go, help.go    — flat argument parsing, JSON output, command help
    commands_*.go       — memory and intelligence command handlers
    skills.go           — local catalog search and explicit external fallback
    agent.go, process.go — agent sessions and bounded subprocess execution
    setup.go            — brain setup orchestration
    setup_flow.go       — overwrite approval, discovery, provisioning, persistence
    setup_display.go, setup_status.go — setup rendering and stored-profile view
    setup_selection.go, setup_skills.go — capability selection and skill discovery
    update.go           — brain update: GitHub release download, SHA-256, atomic replace
    mcp.go              — bounded JSON-RPC 2.0 stdio transport
    mcp_session.go      — MCP initialization, readiness, dispatch, error boundaries
    mcp_tools.go        — tool schemas, argument validation, storage adapters
    ui.go               — lipgloss styles, huh TUI helpers, badge renderers, isTTY()
    scheduler.go        — macOS launchd / Linux systemd timer
    steer.go, steer_display.go — steering-file writes and their presentation
  intelligence/
    profile.go          — bounded project profiling (language, stack, signal detection)
    provision.go        — ProvisionCatalog, DiscoverCapabilities, Provision, writeMCPEntry
    aas.go              — SearchSkillsCatalog, CatalogSeeded (FTS5 over skill items)
    embed.go            — //go:embed skills_index.json, SeedEmbeddedCatalog
    skills_index.json   — 2 737 AAS skills, bundled at build time, refreshed in CI
    compiler.go         — context-pack types, token helpers, Compile orchestration
    retrieval.go, selection.go — evidence retrieval, ranking, budget selection
    recommendations.go  — capability recommendations and reasoning frames
    feedback.go         — feedback persistence and adaptive budgeting
    source.go           — trusted source registry, persistence and synchronization
    source_fetch.go, source_chunks.go — bounded HTTP fetches and text chunking
  memory/
    store.go, schema.go — private connection lifecycle and transactional migrations
    models.go, project.go — data types and project identity
    memories.go         — verified/candidate writes, approval, FTS5 recall
    events.go           — event log, statistics, sync and digest
    skills.go, profiles.go — reviewed skills and persisted project profiles
  release/
    package.go          — standard-library tar.gz/zip archives and SHA-256 sidecars
tests/                  — native executable smoke tests and installer checks
  testdata/legacy.sql   — frozen schema from the retired JavaScript runtime
site/                   — static HTML/CSS, including the native radio command selector
```

Handlers use early returns for invalid input and failed operations; ordinary Go error checks remain explicit. CLI dispatch and capability kinds use switches. Presentation is separate from orchestration so terminal and plain-text branches do not obscure storage or approval decisions.

The storage package owns connection initialization and migrations. Transactions retain deferred rollback and explicit commit, and query readers close before the next query on the single connection. MCP protocol errors, tool execution errors and transport failures remain separate. Refactors preserve JSON field names, empty-array behavior, retrieval ordering and trust rules; behavioral tests cover these boundaries.

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
