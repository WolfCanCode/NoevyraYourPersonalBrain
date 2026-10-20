# 🧠 Personal AI Brain

**One brain. Any agent. Always learning.**

Local-first CLI + MCP memory gateway for existing coding agents. Keep shared project decisions and personal preferences outside any single model or chat session. Find skills through [skills.sh](https://skills.sh), review them, and let your agents reuse them by task keyword.

> **Status: early MVP (v0.1.0)**. This is not a self-improving autonomous agent. Memory retrieval and approved skill loading work today; automatic skill selection from untrusted sources, benchmark-driven evolution, and reliable native hooks for every agent are future work.

## Quick start

Requirements: Node.js >= 22.13 (we recommend 22.20+ for `skills` CLI), macOS or Linux. Windows is usable manually but scheduled setup is not yet implemented.

```bash
git clone https://github.com/WolfCanCode/personal-ai-brain.git
cd personal-ai-brain
npm link
brain init
cd /your/project
brain remember "We require idempotency for payment webhooks" --type decision
brain recall "payment"
brain run claude "Implement payment webhook"
```

`npm link` installs the `brain` executable on your machine; the package does not depend on any cloud service or LLM provider. The CLI uses `node:sqlite` and `fts5` built into Node 22; Node 22 may print an experimental warning when opening SQLite. No dependencies or Docker required.

## Commands

| Command | Function |
| --- | --- |
| `brain init` | Create `~/.brain/brain.sqlite` and configure daily local maintenance |
| `brain remember "..." --type decision` | Store a user-approved project memory |
| `brain remember "..." --scope personal` | Store personal memory available to any project |
| `brain recall "payment"` | Search relevant verified memory |
| `brain context "build payment API"` | Generate task context pack |
| `brain approve <id>` | Promote an agent-proposed candidate to verified |
| `brain run claude "task"` | Launch Claude CLI with private context file |
| `brain run codex "task"` | Launch Codex CLI with the same brain |
| `brain run cursor "task"` | Launch `cursor-agent` with the same brain |
| `brain skills find "react"` | Find skills from skills.sh (network / npx required) |
| `brain skills trust owner/repo@skill --keyword react` | Approve a reviewed skill for matching tasks |
| `brain skills list` | Show approved skills |
| `brain sync --online` | Expire stale memory and check installed skill updates |
| `brain digest --days 7` | Recent memories and events |
| `brain mcp` | Start MCP server over stdio |
| `brain mcp-config` | Print MCP server configuration JSON |
| `brain doctor` | Local dependency diagnostics |

## Connect the MCP gateway

Connect once per client (replace paths according to your installation):

```bash
claude mcp add --scope user brain -- node /ABS/PATH/personal-ai-brain/bin/brain.mjs mcp
codex mcp add brain -- node /ABS/PATH/personal-ai-brain/bin/brain.mjs mcp
brain mcp-config  # use JSON for Cursor's MCP config
```

**Project identity:** The MCP server uses its working directory's Git root to scope project memories. Ensure the client starts the Brain MCP process in the same workspace when relying on project memory. A cross-project project-ID handshake is planned.

### MCP tools

- `brain_context(task, limit?)` — verified context pack for a task.
- `brain_recall(query, limit?)` — verified scoped memory search.
- `brain_remember(content, type?)` — create a **project-scoped candidate** for review.
- `brain_finish_task(summary, status, evidence?)` — store a task outcome event.
- `brain_handoff(objective, completed?, remaining?, evidence?)` — record handoff metadata.

MCP server speaks newline-delimited JSON-RPC over stdio, with no `console.log` on stdout outside protocol. This implementation deliberately supports a small subset of MCP methods; validate with your actual MCP host before depending on interoperability.

## Skills workflow

```bash
brain skills find "typescript"
# Review the contents, scripts, and source of the selected skill yourself.
brain skills trust vercel-labs/agent-skills@web-design-guidelines --keyword design
brain run claude "design a new product landing page"
```

The wrapper matches trusted skills by case-insensitive keyword and loads their prompt via `npx skills use`. **Untrusted skills are never auto-installed or silently trusted.** Skill loading depends on the current version of the external `skills` CLI; network access may be necessary. The initial MVP does **not** auto-recommend or install a newly discovered skill when no trusted match exists. `brain skills find` is an explicit discovery command.

## Daily updates

`brain init` registers a local maintenance schedule at **08:00 local time**:

- macOS: user `launchd` LaunchAgent
- Linux: user `systemd` timer with `Persistent=true` (missed runs are recovered)
- Windows: manual `brain sync --online` for now

The job expires memories with elapsed `expires_at`, records a digest event, and runs `npx skills check`. It **does not** automatically update skills, rewrite memories, access GitHub, or train a model. Run `brain sync` manually when offline. It does not yet recover missed runs on macOS beyond the scheduler's normal behavior.

## Architecture

```text
Claude / Codex / Cursor CLI
     │
     ├── brain run ────────────→ Private Context Pack
     │
     └── MCP stdio ────────────→ Memory API
                                   │
                            One SQLite DB
                          ~/.brain/brain.sqlite
                                   │
               Memory + FTS5 + Events + Trusted Skills
                                   │
                           Daily sync scheduler
```

GitHub contains **public application source**, not private memories. Do not add local databases, secrets, context packs or API keys to commits. The `.gitignore` covers standard local files; still review changes before committing.

## Security and caveats

- User-origin CLI memories are verified; agent-origin memories require explicit approval.
- Memory should be treated as untrusted reference, never as higher-priority instructions.
- Scope checks cover project or personal memory, but this is a single-user local tool, not a multi-tenant authorization boundary.
- Skill trust does not sandbox its scripts: only trust and use skills after reviewing them.
- Agent launch wrapper creates a `0600` temporary context file and removes it when the child process exits.
- Private SQLite data lives in `~/.brain` (directory `0700`, database `0600`). Backups must be protected separately.
- No embeddings, semantic vector search, conflict reconciliation, automatic lesson learning, or cross-device sync in this MVP.
- Agent CLI invocation and MCP compatibility should be tested against your installed versions; `brain run` is a basic wrapper, not a transparent full agent harness.

## Development

```bash
npm test
npm run check
node bin/brain.mjs help
```

Landing page: [`site/index.html`](site/index.html), deploy as a static site. No backend or access to personal memory.

## Roadmap

- [x] SQLite + FTS5 canonical memory store
- [x] Scoped recall, manual review, basic context packs
- [x] MCP stdio tool gateway
- [x] CLI wrapper for installed agents
- [x] Daily maintenance for macOS/Linux
- [x] skills.sh find + manually approved skill use
- [ ] Runtime skill discovery/recommendation when no match (with policy and prompt-injection protection)
- [ ] Session capture adapters and automatic outcome extraction
- [ ] Project identity handshake across MCP clients
- [ ] Skill version pinning and validated upgrades
- [ ] Cross-device sync with single authoritative writer
- [ ] Context quality benchmarks and self-improvement evaluation

Contributions welcome. See [`CONTRIBUTING.md`](CONTRIBUTING.md).

MIT © [WolfCanCode](https://github.com/WolfCanCode)
