# Architecture and trust boundaries

**Canonical store:** One local SQLite database at `${BRAIN_HOME:-~/.brain}/brain.sqlite`. GitHub is authoritative for source code, not runtime data. Memory contents are never committed.

---

## Package structure

```
cmd/brain/          — process entry point; injects version/commit/builtAt via ldflags
internal/
  app/              — CLI, MCP adapter, OS adapters
    cli.go          — command dispatch and flag parsing
    setup.go        — brain setup orchestration (projectRoot, promptSelection, runSetup)
    update.go       — brain update (fetchLatestTag, installUpdate, replaceBinary)
    mcp.go          — JSON-RPC 2.0 stdio MCP server
    scheduler.go    — macOS launchd / Linux systemd timer
  intelligence/     — stateless intelligence layer; never invokes an LLM
    profile.go      — bounded project profiling (language, stack, signal detection)
    provision.go    — ProvisionCatalog, DiscoverCapabilities, Provision, writeMCPEntry
    compiler.go     — token-budgeted context compiler (Compile, Feedback)
    source.go       — trusted external source sync (AddSource, SyncSources)
  memory/           — persistence and trust policy
    store.go        — SQLite schema, FTS5, Recall, UpsertProfile, Trust, Skills
src/                — JavaScript compatibility layer (legacy CLI + MCP)
```

---

## Component diagram

```
Claude · Codex · Cursor · Kiro · any MCP agent
              │
     ┌────────┴────────┐
     │                 │
  brain run         brain mcp
  (CLI layer)       (MCP adapter)
     │                 │
     └────────┬─────────┘
              │
       Intelligence layer
       ┌──────┴──────┐
       │             │
   profile.go    compiler.go ─── provision.go
   (detect)      (compile)       (discover/provision)
       │             │                  │
       └──────┬──────┘           Agent config files
              │                  (~/.kiro/settings/mcp.json
       Memory layer               ~/.claude.json, etc.)
       store.go
       brain.sqlite (WAL · FTS5)
              │
       Trusted external sources ──► daily conditional fetch
```

---

## Data flow: `brain setup`

```
cwd
 │  projectRoot() — walk ≤10 ancestors for .git
 ▼
Project root
 │  intelligence.Profile() — scan ≤4000 files
 ▼
ProjectProfile { Languages, Stacks, Signals, Manifests, Counts }
 │  store.LatestProfile() — check existing
 │  prompt confirm if exists and --refresh not set
 ▼
intelligence.DiscoverCapabilities() — query ProvisionCatalog
 │  up to 10 entries ranked by confidence
 ▼
promptSelection() — user picks subset
 │
 ▼  for each selected capability:
intelligence.Provision()
  mcp  → writeMCPEntry() — atomic write-then-rename to agent config files
  skill → prompt per-skill approval → store.Trust()
  cli  → exec.LookPath()
 │
 ▼
store.UpsertProfile() — persist; evict oldest when count > 3
store.Log("setup_session", …)
```

## Data flow: `brain run <agent>`

```
store.LatestProfile(projectID)  →  stored profile (or detect on-the-fly)
intelligence.Compile()          →  context pack (memories + knowledge + recommendations)
os.MkdirTemp()                  →  session directory 0700
os.WriteFile(context.md, 0600)  →  compact pack + approved skill instructions
exec.Command(agent, task…)      →  BRAIN_CONTEXT_FILE=<path>
defer os.RemoveAll(sessionDir)  →  cleaned up on agent exit
```

---

## Schema (v3)

```sql
memory          — id, type, scope, project_id, content, source, status, expires_at
events          — id, kind, project_id, payload, created_at
settings        — key, value
approved_skills — ref, keyword, approved_at
knowledge_sources — id, name, url, tags, status, etag, last_modified, content_hash, last_synced_at
knowledge_items — id, source_id, title, content, tags, content_hash, status, quality
context_packs   — id, project_id, task, budget_tokens, estimated_tokens
context_pack_items — pack_id, item_kind, item_id, score, estimated_tokens
project_profiles — id, project_id, root, profile_json, capabilities_json, created_at

memory_fts      — FTS5 external-content over memory.content
knowledge_fts   — FTS5 external-content over knowledge_items (title, content, tags)
```

`project_profiles` retains the 3 most recent profiles per `project_id`. `UpsertProfile` evicts the oldest before inserting when count ≥ 3.

---

## Trust classes

| Input | Trust | Allowed action |
|---|---|---|
| User `brain remember` | User-approved | Verified write |
| Agent `brain_remember` via MCP | Untrusted | Candidate only |
| `brain skills trust` | User-approved after review | Skill trust store write |
| `brain setup` provisioning | User-approved per capability | MCP config write, skill trust |
| Registered HTTPS source | Trusted provenance, untrusted content | Bounded text, attributed context |
| Git working tree | May be attacker-controlled | Bounded read-only profiling |
| Remote LLM output | Untrusted | No automatic promotion |

---

## Design constraints

- Dependencies point inward: adapters call storage policy; storage never launches agents.
- The intelligence layer is model-free and provider-neutral; it never calls an LLM.
- `brain setup` and `brain run` never start processes without a user-initiated command.
- External source content remains inert evidence — it is never executed or interpreted as instructions.
- `ProvisionCatalog` is the single source of truth for both `brain setup` capability discovery and `brain advise` context-pack recommendations.
- Schema migrations are additive (`CREATE TABLE IF NOT EXISTS`); version guards prevent downgrade.

---

## Known limitations

- FTS5 lexical search only; no vector/semantic search.
- Project scope relies on Git root of the MCP process working directory.
- MCP implements a bounded stdio subset; validate with your client before depending on edge cases.
- `brain run` is a basic wrapper; it does not sandbox agent processes.
- Windows binary is not Authenticode-signed; macOS binary is not notarized.
- Cross-device sync is not implemented.

---

## Related decisions

- [ADR 0001](adr/0001-native-go-runtime.md) — Why Go over Node or Rust
- [ADR 0002](adr/0002-intelligence-substrate.md) — Intelligence substrate design
- [ADR 0003](adr/0003-external-structural-intelligence.md) — External structural intelligence approach
