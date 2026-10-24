# Native Go verification

Go is the sole Brain runtime. The legacy JavaScript implementation, npm manifest, and JavaScript test tooling have been retired. Existing SQLite stores remain supported through native migrations.

## Scope

- CLI memory commands, FTS5, transactional migrations and maintenance.
- Candidate-only MCP writes, explicit user approval and project scoping.
- Installed-agent wrappers, reviewed skills integration and optional macOS/Linux scheduling.
- Five native build targets, executable smoke tests, checksummed archives and tag-triggered release notes.

## Validation

`go test -race -count=1 ./...` runs unit, integration and installer checks. The integration suite builds a temporary executable by default. Set `BRAIN_TEST_BINARY` to test an existing release executable (absolute path or relative to the repository root); CI also sets `BRAIN_EXPECT_VERSION` to check embedded metadata.

The smoke test starts with `tests/testdata/legacy.sql`, a frozen schema snapshot from the former runtime, and verifies migration, persistent CLI writes, project isolation, personal memory, MCP candidate/approval, sync events and executable configuration. Packaging tests verify tar.gz/zip contents, executable modes and SHA-256 sidecars.

Go dependency versions/checksums are committed in `go.mod` and `go.sum`. CI uses `-mod=readonly`, verifies dependencies, checks formatting, runs static analysis and race tests, then tests the exact executable before Go packaging. All five native jobs must pass; current CI checks are the source of truth for each commit.

## Remaining limitations

Windows scheduling is manual; macOS/Windows binaries are unsigned. MCP implements a bounded stdio subset, and project identity depends on the process working directory. Optional external integrations may need their own runtimes. Performance comparisons require benchmarks on the same hardware and dataset.

A merge alone does not publish a GitHub Release. Explicit version tags run the release workflow described in [RELEASING.md](RELEASING.md).
