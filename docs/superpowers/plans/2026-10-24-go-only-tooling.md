# Go-only tooling implementation plan

**Goal:** Retire the JavaScript runtime, retain SQLite data compatibility, and validate and package native releases using Go.

**Architecture:** Keep `cmd/brain` and the existing app, memory, and intelligence packages. Put executable integration tests in `tests`, independent archive tooling in `internal/release`, and a thin packaging entry point in `cmd/package`. Preserve installer scripts and the static HTML/CSS site; replace its JavaScript demo switcher with native HTML controls.

**Tech stack:** Go standard library, existing SQLite driver, GitHub Actions.

## Tasks

- [x] Reproduce formatting failure and run the existing Go test baseline.
- [x] Load Brain context and its trusted Go guidelines; retain graph-based code discovery.
- [x] Add Go executable smoke tests covering version metadata, persistent CLI writes, project isolation, personal memory, MCP candidate approval, sync events, and executable configuration. Use `BRAIN_TEST_BINARY` to exercise the exact release executable; otherwise build an isolated test executable.
- [x] Port installer checks to Go; keep the POSIX syntax check conditional on shell availability.
- [x] Add archive-content, checksum, mode, invalid-input, and missing-input tests before implementing Go packaging. Use table-driven Linux/macOS tar.gz and Windows zip cases.
- [x] Implement `release.Package(root, binary, version, goos, arch)` with standard-library archive writers and SHA-256; return errors to the packaging command.
- [x] Apply gofmt to the two failing files. Remove legacy mjs runtime/tests and Python packaging only after replacement checks pass.
- [x] Update binary CI to run Go smoke tests and packaging, remove Node setup/baseline, and update checkout to a verified Node 24 action revision. Preserve five native runners and race checks.
- [x] Remove inline site JavaScript while retaining the agent command selector through HTML/CSS. Remove Python from the catalog helper.
- [x] Update contributor, runtime, architecture, and release documentation to the Go-only workflow.
- [x] Run gofmt, dependency verification, go vet, race tests, exact-binary smoke tests, package verification, and cross-compilation for the release targets. Review the diff and graph impact; record Brain feedback/outcome.

## Validation commands

```sh
gofmt -l cmd internal tests
go mod verify
go vet ./...
go test -race -count=1 ./...
go build -trimpath -o build/brain ./cmd/brain
BRAIN_TEST_BINARY="$PWD/build/brain" go test -count=1 -v ./tests
go run ./cmd/package build/brain dev darwin arm64
git diff --check
```

Implementation branch: `codex/go-only-tooling`. Existing untracked agent configuration is preserved.

## Verification results

- Go 1.27.3: gofmt, go vet, module verification, and full race suite passed.
- All five release targets cross-compiled with CGO disabled. Only darwin/arm64 was executed locally; native runners remain the cross-platform execution gate.
- The exact binary and the executable extracted from the macOS archive passed integration smoke tests, including the legacy SQLite fixture.
- macOS tar.gz and Windows zip contents and SHA-256 sidecars verified. Unit tests cover archive formats for all five targets.
- Workflow YAML parsed; shell syntax and git diff whitespace checks passed. No JavaScript or Python source files remain.
- Read-only code review found no actionable issues.
- HTML structure checks passed. Rendered interaction checks remain unverified: Playwright's Chrome is unavailable and in-app local-file navigation was blocked.
- Hosted CI had not rerun at the time of local verification.
