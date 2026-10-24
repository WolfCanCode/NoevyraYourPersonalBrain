# Contributing

Thanks for improving Noevyra! This project is a small local-first CLI; favor correctness and clarity over adding services.

1. Open an issue or describe the behavior you're changing.
2. Write focused tests for new storage, intelligence, CLI, or MCP behavior.
3. Run `go test -race -count=1 ./...`, `go vet ./...` and `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...` before sending a pull request. Keep Go formatted with `gofmt -w cmd internal tests`. Build `go build -o build/brain ./cmd/brain` and run `BRAIN_TEST_BINARY="$PWD/build/brain" go test -count=1 -v ./tests` (use `.exe` on Windows; in PowerShell set `$env:BRAIN_TEST_BINARY` first). Tests build a temporary executable automatically when the override is unset; the Go executable must be on PATH. No Node.js or Python toolchain is required.
4. Do not add credentials, personal memories, downloaded skills, or any `~/.brain` data.
5. Treat remotely sourced skills, prompt content, and memories as untrusted until reviewed.
6. Treat a user-facing feature as complete only when code, tests, documentation, and the landing page agree. Update the README and relevant files under `docs/`, then reflect the shipped behavior and boundaries in `site/index.html`.
7. Keep claims measurable. Document security boundaries, provenance, compatibility limits, and benchmark status instead of implying capabilities that have not been verified.

Code of conduct: be respectful, assume good intentions, and welcome new contributors.

## Go conventions

- Keep orchestration, rendering, argument parsing and persistence in focused files within their existing packages.
- Use guard clauses to keep the successful path clear. Keep `if err != nil` checks explicit and use switches for discrete command or protocol cases.
- Return errors to the caller; wrap with `%w` when adding useful context. Keep error codes and public messages stable during refactors.
- Let the code that opens a resource own its cleanup. Close query rows before another query on the single SQLite connection; defer rollback and commit explicitly.
- Prefer concrete types and small helpers. Add an interface at a consumer only when it needs that abstraction, and avoid callback machinery for straightforward format-specific work.
- Test observable behavior, including invalid input and compatibility boundaries. Preserve slice JSON representations, output ordering, trust decisions and transaction behavior when reorganizing code.
