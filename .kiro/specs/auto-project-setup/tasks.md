# Implementation Plan: Auto-Project-Setup

## Overview

Implement the `brain setup` command, persistent project profile storage, capability discovery and provisioning, Kiro agent support, and setup diagnostics. Two new files are created (`internal/app/setup.go`, `internal/intelligence/provision.go`) and three existing files are modified (`internal/memory/store.go`, `internal/intelligence/compiler.go`, `internal/app/cli.go`). Four test files are added or extended. The implementation language is Go, matching the existing codebase.

## Tasks

- [x] 1. Extend memory store with project profile persistence
  - [x] 1.1 Add `StoredProfile` struct, schema v3 DDL, and migration version bump to `store.go`
    - Add `StoredProfile` struct with fields `ID`, `ProjectID`, `Root`, `ProfileJSON`, `CapabilitiesJSON`, `CreatedAt` (matching the design's definition)
    - Append the `project_profiles` table DDL and `ix_project_profiles_pid_ts` index to the `schema` constant
    - Change the version guard from `version > 2` to `version > 3`
    - Change `PRAGMA user_version=2` to `PRAGMA user_version=3`
    - Add an empty migration block for `version < 3` (no data migration required; `CREATE TABLE IF NOT EXISTS` handles all existing databases)
    - _Requirements: 2.1, 2.3, 2.8_

  - [x] 1.2 Implement `UpsertProfile` method on `*Store`
    - Open a single transaction; query `count(*)` of `project_profiles` rows for `projectID`
    - If count ≥ 3, delete the oldest row by `created_at` for that `projectID` before inserting
    - Insert the new row and return the resulting `StoredProfile`; roll back and return the error if any SQL step fails
    - _Requirements: 2.1, 2.3, 2.8_

  - [x] 1.3 Implement `LatestProfile` method on `*Store`
    - Query `project_profiles` for the single most recent row by `created_at DESC LIMIT 1` where `project_id = ?`
    - Return `(nil, nil)` when no row exists; return `(*StoredProfile, nil)` otherwise
    - _Requirements: 2.4, 7.1, 7.2_

  - [x] 1.4 Write property test for profile store count invariant
    - **Property 2: Profile store count never exceeds three per project**
    - **Validates: Requirement 2.3**
    - Add `pgregory.net/rapid` as a test-only dependency (`go get -t pgregory.net/rapid`) in `internal/memory/store_test.go`
    - Implement `TestUpsertProfileNeverExceedsThree`: draw a random `projectID` (non-empty string) and a call count N (1–50); after each `UpsertProfile` call query the row count and assert it is ≤ 3

  - [x] 1.5 Write unit tests for `UpsertProfile` and `LatestProfile`
    - `TestUpsertProfileAndLatest` — save three profiles for the same project, call `LatestProfile`, verify `ProjectID`, `Root`, and `CreatedAt` match the third insert
    - `TestUpsertProfileEvictsOldest` — insert four profiles; verify the first insert's `ID` is gone and the fourth is present
    - `TestLatestProfileNilWhenMissing` — call `LatestProfile` with an unknown project ID, assert `(nil, nil)`
    - `TestSchemaV3Migration` — open a store that sets `PRAGMA user_version=2` on a fresh SQLite file, close it, reopen it via `memory.Open`, and assert the `project_profiles` table exists
    - _Requirements: 2.1, 2.3, 2.4, 2.8_

- [x] 2. Create provision types, catalog, and `DiscoverCapabilities`
  - [x] 2.1 Create `internal/intelligence/provision.go` with all types and `ProvisionCatalog`
    - Define `NetworkReq` as a string type with constants `NetworkLocal = "local"` and `NetworkNetwork = "network"`
    - Define structs: `MCPServerDef` (Command, Args), `ProvisionableCapability` (Kind, Ref, Name, Reason, Description, Confidence, TriggerSignals, Network, MCPServer, SkillKeyword, CLIBinary), `ProvisionResult` (Capability, Outcome, Error), `AgentConfigPaths` (Paths map)
    - Define `ProvisionOutcome` string type with constants `OutcomeProvisioned`, `OutcomeSkipped`, `OutcomeFailed`
    - Declare `ProvisionCatalog []ProvisionableCapability` containing all 10 entries from the design's catalog table, ordered by base confidence descending; ensure all 11 required signal values (`go`, `rust`, `python`, `typescript`, `javascript`, `database`, `web`, `api`, `security`, `containers`, `ci-cd`) have at least one covering entry
    - _Requirements: 3.2, 3.5_

  - [x] 2.2 Implement `DefaultAgentConfigPaths` in `provision.go`
    - Call `os.UserHomeDir()` to expand `~`; return `AgentConfigPaths` with paths for all four agents: `claude` → `~/.claude.json`, `codex` → `~/.codex/config.json`, `cursor` → `~/.cursor/mcp.json`, `kiro` → `~/.kiro/settings/mcp.json`
    - _Requirements: 4.3, 6.6_

  - [x] 2.3 Implement `DiscoverCapabilities` in `provision.go`
    - Build a lowercase term set from the union of `profile.Languages`, `profile.Stacks`, and `profile.Signals`
    - Iterate `ProvisionCatalog`; include an entry when at least one of its `TriggerSignals` matches any term in the set by case-insensitive exact comparison
    - Append trusted skills from `approved_skills` (passed in as `[]memory.Skill`) whose `Keyword` matches any term in the set; set their `Confidence` to 0.95 and mark them as trusted before applying the cap
    - Sort the result by `Confidence` descending; cap at 10 entries; return an empty (non-nil) slice when nothing matches
    - _Requirements: 3.1, 3.3, 3.6, 3.7_

  - [x] 2.4 Write property test for capability discovery
    - **Property 4: Capability discovery is relevant, sorted, and bounded**
    - **Validates: Requirements 3.1, 3.3**
    - Implement `TestDiscoverCapabilitiesRelevanceSortedBounded` in `internal/intelligence/provision_test.go` using `pgregory.net/rapid`: generate arbitrary Language/Stack/Signal slices; assert len ≤ 10, non-increasing Confidence order, and that every returned entry has at least one TriggerSignal in the profile's term set

  - [x] 2.5 Write property test for `ProjectProfile` serialization round-trip
    - **Property 3: ProjectProfile serialization round-trip**
    - **Validates: Requirement 2.6**
    - Implement `TestProfileSerializationRoundTrip` in `internal/intelligence/provision_test.go` (same package `intelligence`, not `intelligence_test`, to access the struct directly): generate arbitrary `ProjectProfile` values; marshal to JSON, unmarshal back, and assert deep equality with the original using `require.Equal`

- [x] 3. Implement `writeMCPEntry` and `Provision`
  - [x] 3.1 Implement unexported `writeMCPEntry` in `provision.go`
    - Read the existing JSON file at `path` (or start from `{}` if the file does not exist); decode into `map[string]json.RawMessage`
    - Marshal the `entry` (`MCPServerDef`) and insert or replace `mcpServers[serverName]`, preserving all other top-level keys
    - Use a write-to-temp-then-rename strategy: write the updated JSON to a temp file in the same directory, then call `os.Rename`
    - Create any missing parent directories with `os.MkdirAll(..., 0700)` before writing the temp file
    - _Requirements: 4.3, 4.6_

  - [x] 3.2 Write property test for MCP config write idempotence
    - **Property 5: MCP config write is idempotent**
    - **Validates: Requirement 4.6**
    - Implement `TestWriteMCPEntryIdempotent` in `internal/intelligence/provision_test.go` (same package, to call the unexported function): generate a server name (non-empty), a call count N (1–10), and a temp file path; call `writeMCPEntry` N times; assert the final file contains exactly one entry under `mcpServers` with the correct name and that all other top-level keys (if seeded) remain

  - [x] 3.3 Implement `Provision` in `provision.go`
    - `kind == "mcp"`: resolve the brain binary path via `os.Executable()`; for each agent in `agentPaths.Paths` whose binary is found via `exec.LookPath`, call `writeMCPEntry`; if no agent binary is found, return `OutcomeFailed`
    - `kind == "skill"`: print the skill name and description to `out`; read a `y/N` approval from `in`; on approval call `store.Trust(cap.Ref, cap.SkillKeyword)`; on denial return `OutcomeSkipped`
    - `kind == "cli"`: call `exec.LookPath(cap.CLIBinary)`; return `OutcomeFailed` if not found, `OutcomeProvisioned` if found
    - When `dryRun` is `true`: skip all file and DB writes; return `OutcomeProvisioned` for mcp and cli kinds, and `OutcomeSkipped` for skill kinds
    - _Requirements: 4.1, 4.2, 4.3, 4.7, 4.8, 4.9_

  - [x] 3.4 Write unit tests for `Provision` and `writeMCPEntry`
    - `TestProvisionCatalogCoversAllRequiredSignals` — assert each of the 11 required signal values has ≥ 1 covering entry in `ProvisionCatalog`
    - `TestProvisionCatalogNetworkLabels` — assert every entry has `Network` equal to `"local"` or `"network"`
    - `TestDiscoverCapabilitiesIncludesTrustedSkills` — trust a skill with a matching keyword; create a profile with that keyword in Signals; verify the skill appears in the result
    - `TestDiscoverCapabilitiesEmptyProfile` — empty `ProjectProfile` returns empty slice
    - `TestProvisionMCPNoAgentsInstalled` — call `Provision` with a fake `AgentConfigPaths` whose binaries are absent from PATH; assert `OutcomeFailed`
    - `TestProvisionSkillApproved` / `TestProvisionSkillDenied` — pass `"y\n"` / `"n\n"` as stdin; verify trust-store write vs skip
    - `TestWriteMCPEntryCreatesParentDirs` — call `writeMCPEntry` with a path inside a non-existent subdirectory; assert the directory and file are created
    - `TestWriteMCPEntryPreservesOtherKeys` — seed the file with a non-`mcpServers` top-level key; after calling `writeMCPEntry`, assert that key is still present
    - _Requirements: 3.2, 4.2, 4.3, 4.6, 4.8, 4.9_

- [x] 4. Refactor `compiler.go` to use `ProvisionCatalog`
  - [x] 4.1 Replace the internal `capability` type with `ProvisionableCapability` in `compiler.go`
    - Remove the unexported `capability` struct and the `capabilityCatalog` var
    - In `capabilityRecommendations`, replace the old iteration with one over `ProvisionCatalog` (from `provision.go`, same package); read `item.TriggerSignals`, `item.Kind`, `item.Ref`, `item.Reason` in place of the old unexported fields
    - Verify that no exported function signatures change (`Compile`, `Feedback`, `KnowledgeCount`, `PackSummary`, `SyncSources`, etc.)
    - _Requirements: 3.1, 3.3_

- [x] 5. Checkpoint — ensure memory and intelligence layer tests pass
  - Run `go test ./internal/memory/... ./internal/intelligence/...` and confirm all tests pass; ask the user if questions arise before continuing.

- [x] 6. Implement `setup.go` orchestration
  - [x] 6.1 Create `internal/app/setup.go` with `projectRoot` and interactive helpers
    - Implement `projectRoot(cwd string) string`: check `cwd` first for a `.git` entry via `os.Stat`; then walk up with `filepath.Dir`, stopping after 10 iterations; return `cwd` unchanged if no `.git` is found within that bound
    - Implement `displayProfile(out io.Writer, profile intelligence.ProjectProfile)`: print detected languages, stacks, and signals in human-readable form
    - Implement `promptConfirm(in io.Reader, out io.Writer, msg string) (bool, error)`: write the message and `" [y/N]: "` to `out`; read one line from `in`; return `true` only for `"y"` or `"Y"`
    - Implement `promptSelection(in io.Reader, out io.Writer, caps []intelligence.ProvisionableCapability) ([]intelligence.ProvisionableCapability, error)`: print a numbered list with each capability's name, kind, and network label; read one line; parse `"all"` (return all), `"none"` (return empty), or comma-separated integers
    - Implement `printProvisionSummary(out io.Writer, results []intelligence.ProvisionResult)`: print an aligned table of capability name → outcome
    - Implement `buildSessionEvent(results []intelligence.ProvisionResult) map[string]any`: return a map with `"capabilities"` (names attempted) and `"outcome"` classified as `"completed"`, `"partial"`, or `"failed"` per Requirement 7.4
    - _Requirements: 1.1, 1.6, 1.7, 4.5, 7.4_

  - [x] 6.2 Write property test for `projectRoot` traversal bound
    - **Property 1: Project root traversal respects the 10-level ancestor bound**
    - **Validates: Requirements 1.1, 1.7**
    - Implement `TestProjectRootTraversalBound` in `internal/app/setup_test.go` (package `app`) using `pgregory.net/rapid`: draw a depth (0–12) and a git level within [0, depth]; build a directory chain of `depth` levels in a temp dir; place `.git` at `chain[gitLevel]`; call `projectRoot(chain[depth])`; assert it returns `chain[gitLevel]` when `depth - gitLevel ≤ 10`, else returns `chain[depth]`

  - [x] 6.3 Implement `runSetup` in `setup.go`
    - Parse flags `--refresh`, `--dry-run`, `--status` using the existing `flags()` helper from `cli.go`
    - `--status` path: call `store.LatestProfile(project.ID)`; if nil, print the no-profile message from Requirement 7.2 and return; otherwise display all stored fields (languages, stacks, signals, capabilities, timestamp) per Requirement 7.1
    - Main path: call `projectRoot(cwd)` and `intelligence.Profile(root)`; call `displayProfile`; call `store.LatestProfile`; if a profile exists and `--refresh` is not set, call `promptConfirm` with the existing profile's summary; on denial print a cancellation message and return without writing
    - Call `intelligence.DiscoverCapabilities(profile, trusted)`; if result is empty, print the no-match message per Requirement 3.7 and skip selection
    - If `--dry-run`: print the ranked capability list and the planned provisioning steps, then return without writing anything
    - Call `promptSelection`; for each selected capability call `intelligence.Provision(..., dryRun: false, ...)`; collect `[]ProvisionResult`
    - Call `store.UpsertProfile(project.ID, root, profileJSON, capabilitiesJSON)` with JSON-marshaled `profile` and `[]ProvisionResult`
    - Call `store.Log("setup_session", project.ID, buildSessionEvent(results))`; on log error, write to `errOut` and continue
    - Call `printProvisionSummary`
    - _Requirements: 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 2.1, 2.4, 2.5, 2.7, 3.4, 3.7, 4.1, 4.4, 4.5, 7.1, 7.2, 7.4_

  - [x] 6.4 Write unit tests for `runSetup`
    - `TestProjectRootFallbackNoGit` — create a directory chain deeper than 10 levels with no `.git`; verify `projectRoot` returns the deepest directory
    - `TestRunSetupStatus_NoProfile` — call `runSetup` with `--status` on a fresh store; assert stdout contains the no-profile instruction message
    - `TestRunSetupStatus_WithProfile` — insert a `StoredProfile` row; call `runSetup --status`; assert all required fields appear in stdout
    - `TestRunSetupDryRun` — call `runSetup --dry-run`; assert no rows in `project_profiles` or `events`, and no files created
    - `TestRunSetupRefreshSkipsConfirmation` — insert a profile; call `runSetup --refresh` with stdin that would deny; assert setup proceeds without prompting
    - `TestRunSetupAbortOnDenyConfirmation` — insert a profile; provide `"n\n"` on stdin; assert no changes to `project_profiles` and a cancellation message in stdout
    - `TestRunSetupSessionEventRecorded` — complete a full setup run; query the `events` table and assert a `setup_session` row exists
    - `TestRunSetupProvisionSummaryOutcomes` — mock provisioning results to cover all three outcomes; assert `"provisioned"`, `"skipped"`, and `"failed"` each appear in stdout
    - _Requirements: 1.7, 2.4, 2.5, 2.7, 3.4, 3.7, 4.4, 4.7, 7.1, 7.2, 7.4_

- [x] 7. Update `cli.go` for setup command and Kiro agent support
  - [x] 7.1 Add `setup` command dispatch to `cli.go`
    - Add `"setup": true` to the `allowed` map
    - Add `case "setup": return runSetup(store, project, args, in, out, errOut, home)` in the command switch (placed before the default)
    - _Requirements: 1.1_

  - [x] 7.2 Add Kiro to `runAgent` in `cli.go`
    - Add `"kiro": "kiro"` to the `binaries` map inside `runAgent`
    - Update the arg-count error message to `"use brain run claude|codex|cursor|kiro [task]"`
    - After the `binaries[args[0]]` lookup, add a pre-flight `exec.LookPath(binary)` check; when the agent is `"kiro"` and the binary is not found, write `"brain: kiro executable not found in PATH; install the Kiro CLI and ensure it is accessible before retrying"` to `errOut` and return the lookup error
    - _Requirements: 5.3, 6.1, 6.2, 6.3, 6.4, 6.5, 6.7_

  - [x] 7.3 Add `kiro` to the `doctor` binary check in `cli.go`
    - Add `"kiro"` to the binary slice iterated in the `doctor` case: `[]string{"git", "claude", "codex", "cursor-agent", "kiro", "npx"}`
    - _Requirements: 6.8_

  - [x] 7.4 Write unit tests for CLI Kiro support
    - `TestDoctorIncludesKiro` — call `Run` with `["doctor"]`; decode the JSON output and assert `commands["kiro"]` is present (bool, either true or false depending on PATH)
    - `TestRunAgentKiroNotFound` — set PATH to an empty temp dir; call `Run` with `["run", "kiro", "test task"]`; assert stderr contains the kiro-specific install message
    - `TestMcpConfigIsKiroCompatible` — call `Run` with `["mcp-config"]`; decode JSON and assert `mcpServers.brain.command` is non-empty and `mcpServers.brain.args` equals `["mcp"]`
    - _Requirements: 6.1, 6.6, 6.7, 6.8_

- [x] 8. Final checkpoint — ensure all tests pass
  - Run `go test ./...` and confirm all tests pass; ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- `pgregory.net/rapid` is a test-only dependency; add it once via `go get -t pgregory.net/rapid` as part of task 1.4 — it is available to all subsequent property test tasks
- The property tests for `writeMCPEntry` (3.2) and `ProfileSerializationRoundTrip` (2.5) should be placed in `package intelligence` (not `package intelligence_test`) so they can call unexported functions directly
- `brain mcp-config` requires no code change — its existing output is already Kiro-compatible (Requirement 6.6); task 7.4's `TestMcpConfigIsKiroCompatible` only verifies this existing behavior
- Tasks 7.1, 7.2, and 7.3 each modify `cli.go` and must be applied in sequence; the dependency graph enforces this ordering
- The `UpsertProfile` eviction query uses `DELETE … ORDER BY created_at ASC LIMIT 1`; SQLite supports this syntax with the `SQLITE_ENABLE_UPDATE_DELETE_LIMIT` compile-time option, which `modernc.org/sqlite` enables by default
- Each task references specific requirements for traceability

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["1.3", "2.2", "6.1"] },
    { "id": 3, "tasks": ["1.4", "2.3", "6.2"] },
    { "id": 4, "tasks": ["1.5", "3.1"] },
    { "id": 5, "tasks": ["2.4", "4.1"] },
    { "id": 6, "tasks": ["2.5"] },
    { "id": 7, "tasks": ["3.2", "3.3"] },
    { "id": 8, "tasks": ["3.4", "6.3"] },
    { "id": 9, "tasks": ["6.4", "7.1"] },
    { "id": 10, "tasks": ["7.2"] },
    { "id": 11, "tasks": ["7.3"] },
    { "id": 12, "tasks": ["7.4"] }
  ]
}
```
