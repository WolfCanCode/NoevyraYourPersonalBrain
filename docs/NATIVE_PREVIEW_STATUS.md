# Native preview review status — 20 October 2026

Prepared locally against GitHub main `5ed882aeef359e67d8343157b0a29fde959a820c`.

## Implemented draft

- Go CLI, SQLite/FTS memory store, transactional migration/maintenance, candidate trust policy, stdio MCP, installed-agent wrappers, reviewed skills integration and optional macOS/Linux scheduling.
- Unit/race test sources and an executable smoke test with bidirectional JavaScript SQLite interoperability.
- Five-target reusable CI/build workflow, archives/checksums, tag-based release workflow with generated release notes, SHA-pinned actions, Dependabot, architecture ADR and release documentation.
- Existing JavaScript implementation and database format preserved.

## Verified here

- JavaScript baseline: 4/4 tests passed; syntax checks passed.
- New Node smoke script syntax and Python packaging script syntax passed.
- Workflow YAML parsing and embedded Bash syntax checks passed.
- Python SQLite fixture: legacy missing-index rebuild and FTS insert/update/delete triggers passed. This validates SQL, not the Go driver/runtime.
- Changes contain application code, tests, workflow configuration and docs; no runtime memory databases or environment files. Credential-pattern scan had no findings.

## Unverified and next gates

There is no Go toolchain in this environment; downloads timed out. No native compilation, Go unit/race tests, packaged executable smoke tests or cross-platform run has happened. Do not call these tests passed or treat this preview as release-ready.

Automatic approval review rejected public GitHub source upload twice, including after verifying the exact repository and scanning the changed files. No implementation commit was published. An empty `feat/native-go-releases` branch exists at the previous main commit. Obtain explicit user approval for publishing these changes to public `WolfCanCode/personal-ai-brain` with the already approved Tommy Le author profile, creating a PR and running CI. Do not work around the rejection or use the scheduled task to bypass it.

Once authorized:

1. Re-read current remote head, preserve any intervening changes, publish on a feature branch and open a PR.
2. Run the bootstrap workflow to obtain `gofmt` output, the resolved `go.mod` and `go.sum`; commit those results to the branch. The initial workflow deliberately runs `go mod tidy` and exports the formatted source/lockfiles as a `go-bootstrap` artifact.
3. Replace bootstrap dependency resolution with `go mod download` / `go mod verify`; require a clean `gofmt` result and no module-file changes. Remove bootstrap logs/artifact steps. Use `-mod=readonly` for verification/build commands.
4. Verify actual runner architectures, toolchain availability, all native builds, Go tests/race detection, SQLite interoperability and exact-binary smoke tests. Fix failures and rerun affected checks.
5. Only then consider merging. Do not create a release tag or publish production automatically. A version tag will trigger the separately requested release workflow and generated release notes.

The weekly automation was updated to review relevant primary architecture sources, preserve compatibility, maintain binary/release workflows and report blockers truthfully; its Monday 08:00 Europe/Amsterdam schedule is unchanged.
