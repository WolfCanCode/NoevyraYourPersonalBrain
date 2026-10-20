# Architecture and trust boundaries

**Canonical store:** One local SQLite database at `${BRAIN_HOME:-~/.brain}/brain.sqlite`. GitHub is authoritative for the source code, not runtime data. Memory contents are never committed by the app.

## Runtime

Native Go preview adds `cmd/brain` (process entry), `internal/app` (CLI, MCP and OS adapters), and `internal/memory` (persistence and trust policy). It preserves the JavaScript format below. See [ADR 0001](adr/0001-native-go-runtime.md) for the decision, migration and limitations, and [releases](RELEASING.md) for native verification and distribution.

The v0.2 intelligence layer lives in `internal/intelligence`: bounded project profiling, explicitly trusted text-source synchronization, token-budgeted evidence compilation, explainable capability routing and feedback-weighted knowledge ranking. It remains model-free and provider-neutral. See [ADR 0002](adr/0002-intelligence-substrate.md).

Keep dependencies pointing inward: adapters call storage policy; storage never launches agents or invokes a model. Migration, FTS rebuild, and maintenance updates use transactions. WAL, a busy timeout, and one pooled connection per process avoid accidental per-connection pragma differences. SQL parameters carry user values; FTS tokens are quoted. Operational errors are returned instead of silently degrading search.

Best practices are adopted from primary sources with a reason and a compatibility test, not copied indiscriminately from agent frameworks. Prefer local data, explicit trust transitions, bounded protocol messages/subprocess output, opt-in external execution, minimal workflow permissions and SHA-pinned actions. The opted-in daily timer refreshes registered sources with conditional requests; it never installs or executes fetched instructions.

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
| Explicitly registered HTTPS source | Trusted provenance, untrusted content | Bounded text retrieval and attributed context only |
| Project files and manifests | Untrusted evidence | Bounded read-only profiling; never interpreted as control instructions |

## Known limitations

- No embedding search, only SQLite FTS5.
- No authoritative merge or cross-device sync.
- Project scope relies on working directory and Git root.
- MCP implementation is intentionally minimal; it is not a complete MCP SDK implementation.
- `brain run` passes a short instruction containing a file path to installed CLIs; it doesn't inject native lifecycle hooks.
- Daily maintenance refreshes only explicitly trusted sources. Skill checks and upgrades remain explicit because the skills CLI executes third-party code.

## Planned next architecture milestone

Measure recommendation precision, tokens per successful task and repeated-context reduction. Then add a curated primary-source feed catalog, source-specific extractors, project identity handshake and evaluated skill promotion. Any model-assisted reranker must beat the deterministic baseline before becoming a default.
