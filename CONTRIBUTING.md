# Contributing

Thanks for improving Personal AI Brain! This project is a small local-first CLI; favor correctness and clarity over adding services.

1. Open an issue or describe the behavior you're changing.
2. Write focused tests for new storage, intelligence, CLI, or MCP behavior.
3. Run `go test -race ./...`, `go vet ./...`, `npm test` and `npm run check` before sending a pull request. Keep Go formatted with `gofmt`. Build `go build -o build/brain ./cmd/brain` and run `node scripts/smoke.mjs build/brain` (use `.exe` on Windows).
4. Do not add credentials, personal memories, downloaded skills, or any `~/.brain` data.
5. Treat remotely sourced skills, prompt content, and memories as untrusted until reviewed.
6. Treat a user-facing feature as complete only when code, tests, documentation, and the landing page agree. Update the README and relevant files under `docs/`, then reflect the shipped behavior and boundaries in `site/index.html`.
7. Keep claims measurable. Document security boundaries, provenance, compatibility limits, and benchmark status instead of implying capabilities that have not been verified.

Code of conduct: be respectful, assume good intentions, and welcome new contributors.
