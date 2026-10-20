# Native Go preview verification

The native implementation is developed in [PR #1](https://github.com/WolfCanCode/personal-ai-brain/pull/1). It preserves the JavaScript CLI and SQLite format while adding a standalone executable.

## Scope

- CLI memory commands, FTS5, transactional migrations and maintenance.
- Candidate-only MCP writes, explicit user approval and project scoping.
- Installed-agent wrappers, reviewed skills integration and optional macOS/Linux scheduling.
- Five native build targets, executable smoke tests, checksummed archives and tag-triggered release notes.

## Validation status

JavaScript baseline: 4/4 tests passed locally. SQL fixtures and workflow syntax checks also passed. The PR CI matrix validates native compilation, race tests, packaged executables and bidirectional SQLite interoperability on all five supported platforms; use the checks for the current commit as the source of truth.

Go source is formatted and dependency versions/checksums are committed in `go.mod` and `go.sum`. CI runs with `-mod=readonly`, verifies dependencies and rejects module-file or formatting drift. The initial bootstrap steps have been removed. All five native jobs must pass before the PR is ready.

## Remaining limitations

Windows scheduling is manual; macOS/Windows binaries are unsigned. MCP implements a bounded stdio subset, and project identity still depends on the process working directory. Performance comparisons require benchmarks on the same hardware and dataset.

A merge alone does not publish a GitHub Release. Explicit version tags run the release workflow described in [RELEASING.md](RELEASING.md).
