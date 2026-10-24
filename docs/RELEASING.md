# Native binaries and releases

Every PR and push to `main` runs the same reusable binary workflow that a tagged release uses:

1. Formatting, static analysis, Go unit/race tests and installer checks.
2. Build with CGO disabled, trimmed paths and version/commit/time metadata.
3. Run the actual binary: migrate a legacy SQLite schema fixture, verify persistent writes and project isolation, MCP candidate/approval, sync and configuration.
4. Archive the tested executable with license/docs using Go; calculate SHA-256.

From the repository root, test and package a local executable with:

```sh
go build -o build/brain ./cmd/brain
BRAIN_TEST_BINARY="$PWD/build/brain" go test -count=1 -v ./tests
go run ./cmd/package build/brain dev darwin arm64
```

Use your executable's OS and architecture for the final two arguments. Windows packages use `.zip` and `brain.exe`; Linux and macOS use `.tar.gz` and an executable `brain`. The packager writes `dist/brain_<version>_<os>_<arch>.<extension>` and its `.sha256` sidecar. Neither testing nor packaging requires Node.js or Python. GitHub's official actions manage their own runtime.

Targets: Linux x64 and ARM64, macOS Intel and Apple Silicon, Windows x64. These are native builds tested on matching runners, not merely cross-compiled executables. PR artifacts are development builds with a 14-day retention period; they are not GitHub Releases.

## Create a release

Run **Bump and Release** on `main` with `patch`, `minor`, or `major` to commit the next version, create its tag, test that exact commit on all five native runners, and publish the archives. Optional `git_name` and `git_email` inputs select the bump commit's author; otherwise it uses the triggering account's login and GitHub noreply address. The executable, README badge and landing-page version labels update together. The binary smoke test verifies both the version and the checked-out commit metadata.

After merging tested changes into `main`, create and push a semantic version tag from that exact commit:

```sh
git checkout main
git pull --ff-only
git tag v0.3.0
git push origin v0.3.0
```

`release.yml` validates the tag, reruns the entire build/test matrix and publishes only if every target passes. GitHub generates release notes from the changes, PRs and contributors since the preceding release, categorized by `.github/release.yml`. Tags such as `v0.2.1-rc.1` create prereleases. No PAT or external release action is required; only the publish job receives `contents: write`.

Do not move published tags. Fix a failed release before retrying; if a release already exists, inspect it and its assets instead of blindly replacing downloads. Creating a tag is an explicit release action; merging code alone does not publish a version.

Download the appropriate archive and `SHA256SUMS` from the release, verify it with `sha256sum --check SHA256SUMS` (Linux), `shasum -a 256` (macOS), or `Get-FileHash -Algorithm SHA256` (PowerShell), extract and place the binary on PATH. macOS downloads are not Developer ID signed/notarized and Windows executables are not Authenticode signed; OS trust prompts may appear. Checksums detect corrupted downloads but are not publisher signatures.

The repository-root `install.sh` and `install.ps1` automate that same download, checksum verification and PATH-directory installation. The POSIX installer supports Linux/macOS on AMD64 and ARM64; the PowerShell installer supports the published Windows AMD64 target. Both default to the latest GitHub Release and accept a pinned version and custom install directory. Keep the asset naming convention and installer tests synchronized whenever the release matrix changes.

Before switching, stop running Brain processes and back up the database with SQLite's backup API/command; copying only `brain.sqlite` while writers use WAL can omit recent transactions. Reconfigure MCP to use `/absolute/path/brain mcp` and reinstall local schedules with the new permanent executable path via `brain init --schedule`. The legacy JavaScript runtime has been retired; the Go runtime retains its SQLite schema compatibility. Integration tests migrate a frozen legacy schema, reopen the database across CLI processes, and verify old and new memories remain searchable.

## Sources

- [GitHub generated release notes](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes)
- [GitHub workflow security](https://docs.github.com/en/actions/reference/security/secure-use)
- [GitHub hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)

Actions are pinned to full commit SHAs and Dependabot proposes weekly action/module updates. Checksums, tests and changelog generation do not substitute for code review or binary signing.
