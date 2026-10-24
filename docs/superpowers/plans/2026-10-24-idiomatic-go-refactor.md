# Idiomatic Go refactor

**Goal:** Improve every Go package through focused responsibilities, shallow control flow and explicit error handling while preserving existing behavior and public APIs.

**Compatibility contract:** Keep CLI arguments/output/errors, MCP lifecycle/error codes/tool schemas, memory trust rules, schema migrations, source bounds and release archive names/content stable. Preserve all work from the Go-only migration and existing agent configuration. Retain ordinary Go error guards; do not introduce a framework or interfaces without a concrete consumer.

## Tasks

- [x] Inspect graph architecture, symbols, dependencies, patterns and test gaps; collect actual baseline coverage because graph complexity/coverage results are incomplete.
- [x] Split CLI help, parsing, memory commands, intelligence commands, skills, subprocess and agent execution into focused files; simplify dispatch and flatten nested skill branches.
- [x] Separate setup rendering, selection and skill discovery from orchestration; extract status, overwrite confirmation, capability discovery, dry-run and persistence stages.
- [x] Separate MCP transport/session dispatch from typed tool handlers; preserve initialization and tool-vs-protocol error distinctions.
- [x] Split memory schema/models, migrations, memories, events, trusted skills and stored profiles; simplify connection/transaction cleanup, query construction and repeated row decoding.
- [x] Refactor intelligence compilation/ranking/recommendations, manifest inspection, source synchronization and provisioning to reduce nested branches and duplication. Preserve retrieval ordering, token budget metrics, config merge semantics and failure handling.
- [x] Review updater, scheduler, steering and release packaging; flatten branching and separate format-specific code where it improves clarity.
- [x] Add behavioral characterization tests around refactored boundaries, run package tests between steps, and perform spec and code-quality review.
- [x] Run gofmt, go vet, full race tests, executable smoke tests and five-target compilation. Update architecture docs and record Brain feedback/outcome.

## Validation

Use the temporary Go 1.27.3 toolchain at `/tmp/noevyra-go-toolchain/go/bin/go` (with its bin directory on PATH for child commands). Baseline package coverage: app 28.6%, intelligence 60.4%, memory 56.3%, release 79.0%. Coverage measures exercised behavior; it is not a target for cosmetic tests.

```sh
go test ./internal/app ./internal/intelligence ./internal/memory ./internal/release
go vet ./...
go test -race -count=1 ./...
go build -trimpath -o build/brain ./cmd/brain
BRAIN_TEST_BINARY=build/brain go test -count=1 -v ./tests
git diff --check
```

Keep changes local on `codex/go-only-tooling`; do not publish a release or alter stored user data as part of refactoring.

## Results

- CLI, setup, steering, MCP, memory, intelligence and release code now have focused orchestration, transport, rendering, storage and format-specific responsibilities. Small entry points and already-clear helpers remain unchanged.
- Setup orchestration decreased from 244 to 56 lines; steering orchestration from 148 to 55; Compile from 83 to 45. Switches and explicit Go error guards remain where they express the domain clearly.
- Behavioral characterization tests cover command validation, metadata commands without storage side effects, MCP lifecycle/transport errors, future-schema protection, digest/skill persistence, steering idempotence, selection boundaries and JSON config merging.
- Fixed one discovered bug: null MCP config objects/envelopes now act as empty maps instead of panicking.
- Staticcheck v0.8.2 passes and is pinned in native binary CI. Removed unused helpers/styles and a duplicate trim character; smoke builds now find the Go executable on PATH instead of using deprecated runtime.GOROOT.
- gofmt, go vet, module verification, full race tests, exact-binary smoke tests, extracted-archive smoke tests, checksum verification and all five target builds passed locally. Workflow YAML and git diff whitespace checks passed.
- Package coverage after added characterization tests: app 35.8%, intelligence 65.9%, memory 73.5%, release 81.6% (before final dead-code-only cleanup).
- Separate compatibility/spec review and code-quality review found no actionable issues. Graph review returned generic caller/test-update warnings and no errors; source review and tests verified the extracted boundaries.
- Implementation branch: codex/go-only-tooling; hosted CI was pending at local verification time.
