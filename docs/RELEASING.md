# Native binaries and releases

Every PR and push to `main` runs the same reusable binary workflow that a tagged release uses:

1. Static analysis, Go unit/race tests and the JavaScript baseline.
2. Build with CGO disabled, trimmed paths and version/commit/time metadata.
3. Run the actual binary: SQLite writes, bidirectional JavaScript compatibility, MCP candidate/approval, sync and configuration.
4. Archive the tested executable with license/docs; calculate SHA-256.

Targets: Linux x64 and ARM64, macOS Intel and Apple Silicon, Windows x64. These are native builds tested on matching runners, not merely cross-compiled executables. PR artifacts are development builds with a 14-day retention period; they are not GitHub Releases.

## Create a release

After merging tested changes into `main`, create and push a semantic version tag from that exact commit:

```sh
git checkout main
git pull --ff-only
git tag v0.2.0
git push origin v0.2.0
```

`release.yml` validates the tag, reruns the entire build/test matrix and publishes only if every target passes. GitHub generates release notes from the changes, PRs and contributors since the preceding release, categorized by `.github/release.yml`. Tags such as `v0.2.0-rc.1` create prereleases. No PAT or external release action is required; only the publish job receives `contents: write`.

Do not move published tags. Fix a failed release before retrying; if a release already exists, inspect it and its assets instead of blindly replacing downloads. Creating a tag is an explicit release action; merging code alone does not publish a version.

Download the appropriate archive and `SHA256SUMS` from the release, verify it with `sha256sum --check SHA256SUMS` (Linux), `shasum -a 256` (macOS), or `Get-FileHash -Algorithm SHA256` (PowerShell), extract and place the binary on PATH. macOS downloads are not Developer ID signed/notarized and Windows executables are not Authenticode signed; OS trust prompts may appear. Checksums detect corrupted downloads but are not publisher signatures.

Before switching, stop running Brain processes and back up the database with SQLite's backup API/command; copying only `brain.sqlite` while writers use WAL can omit recent transactions. Reconfigure MCP to use `/absolute/path/brain mcp` and reinstall local schedules with the new permanent executable path via `brain init --schedule`. Native preview and legacy JavaScript remain available during migration.

## Sources

- [GitHub generated release notes](https://docs.github.com/en/repositories/releasing-projects-on-github/automatically-generated-release-notes)
- [GitHub workflow security](https://docs.github.com/en/actions/reference/security/secure-use)
- [GitHub hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)

Actions are pinned to full commit SHAs and Dependabot proposes weekly action/module updates. Checksums, tests and changelog generation do not substitute for code review or binary signing.
