# Architecture and trust boundaries

**Canonical store:** One local SQLite database at `${BRAIN_HOME:-~/.brain}/brain.sqlite`. GitHub is authoritative for the source code, not runtime data. Memory contents are never committed by the app.

## Runtime

- `src/store.mjs`: SQLite schema, FTS5, scope-filtered search, candidate review, project hashing and append-only event records.
- `src/mcp.mjs`: JSON-RPC 2.0 stdio gateway; tool calls are restricted to the invoking project. Agent writes become candidates.
- `src/cli.mjs`: CLI, agent wrappers, skills CLI integration, status and diagnostics.
- `src/scheduler.mjs`: macOS `launchd` or Linux `systemd` daily maintenance.

## Trust classes

| Input | Trust | Allowed action |
| --- | --- | --- |
| User invoking `brain remember` | User-approved | Verified write |
| Agent writing through MCP | Untrusted | Candidate only |
| Third-party skill from skills.sh | Untrusted | Explicit trust required |
| Git working tree | May be attacker-controlled | Reads only when requested by agent |
| Remote LLM output | Untrusted | No automatic skill promotion |

## Known limitations

- No embedding search, only SQLite FTS5.
- No authoritative merge or cross-device sync.
- Project scope relies on working directory and Git root.
- MCP implementation is intentionally minimal; it is not a complete MCP SDK implementation.
- `brain run` passes a short instruction containing a file path to installed CLIs; it doesn't inject native lifecycle hooks.
- Daily maintenance checks skills for updates but doesn't automatically upgrade them.

## Planned next architecture milestone

Project identity handshake, robust permissions, structured agent execution events, and a skill resolver with an explicit human-approval queue. Once those are tested, add evaluated skill promotion.
