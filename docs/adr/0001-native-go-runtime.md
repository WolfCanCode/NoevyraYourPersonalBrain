# ADR 0001: Incremental native Go runtime

Date: 2026-10-20. Status: accepted for a native preview; the JavaScript CLI remains supported.

## Context and decision

The product is a local memory CLI and stdio MCP gateway, not an LLM inference engine. Shipping one executable, preserving private SQLite data and keeping maintenance manageable matter more than speculative throughput. Use Go with `modernc.org/sqlite` (CGO-free SQLite with FTS5) for the first native implementation. Use `cmd/brain` for process entry, `internal/app` for adapters and `internal/memory` for storage and trust rules. Do not introduce a network service, ORM or a generic repository abstraction for one storage backend.

Rust would also fit a native CLI and has no tracing garbage collector. It adds a separate language/tooling learning cost; SQLite/linker cross-compilation needs consideration. Go trades some runtime/memory overhead for simpler iteration and distribution. Neither choice proves faster recall than the existing Node implementation. Keep the baseline and measure before claiming a win. Do not rewrite the landing page or require a Vercel agent framework for a local tool.

## Compatibility and consequences

- Both CLIs read/write the same `brain.sqlite` schema and project SHA-256 IDs.
- Native migration uses SQLite `user_version=1`; unsupported future versions fail without applying DDL.
- The first native open rebuilds the external-content FTS index in the schema transaction. Row counts cannot detect missing external-content index entries.
- Candidate-only agent writes and CLI-only personal memory/approval are enforced by storage entry points.
- `brain init --schedule` explicitly installs a timer; plain native `init` only initializes data. Native timer runs local `sync`, without network or skill execution.
- Node remains required for the legacy CLI and optional `npx skills` integration, not native memory/MCP operations.
- Windows file permissions depend on the user's ACLs. Unix modes do not establish Windows confidentiality.
- MCP is a bounded, sequential stdio subset supporting negotiated stable 2024/2025 lifecycle versions. It is not a complete implementation of every current MCP transport or feature. Unknown newer client versions receive a supported version and must decide whether to continue.

## Verification and follow-up

Require native unit tests, race detection, legacy tests and bidirectional database interoperability against each packaged executable. Benchmark `go test ./internal/memory -bench Recall -benchmem`; compare startup, RSS, binary size and end-to-end recall on the same hardware/data before making performance claims. Capture p50/p95 on representative repositories. Revisit Rust only if measured bottlenecks justify the migration cost.

The path-based project identity and independently installed agent CLIs remain limitations. A project handshake, pinned third-party skills, official MCP SDK evaluation and retrieval quality corpus are follow-up work.

## Primary references

- [Go module layout](https://go.dev/doc/modules/layout)
- [modernc SQLite driver](https://pkg.go.dev/modernc.org/sqlite)
- [SQLite FTS5 external-content tables and rebuild](https://www.sqlite.org/fts5.html#external_content_tables)
- [SQLite WAL and concurrency](https://www.sqlite.org/wal.html)
- [MCP lifecycle 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)
- [MCP tool errors](https://modelcontextprotocol.io/specification/2025-06-18/server/tools#error-handling)
