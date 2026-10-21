# Brain setup

`brain setup` is the fastest way to go from a fresh checkout to a fully configured agent workspace. It runs project detection, capability discovery, user-approved provisioning, and profile persistence in a single interactive command.

---

## What it does

```
brain setup
```

1. **Detect project root** — walks up from the current directory looking for `.git`, up to 10 levels. Falls back to the current directory if none found.
2. **Scan the repository** — reads up to 4000 files, derives languages, stacks and signals. Parses root-level manifests (`go.mod`, `package.json`, `Cargo.toml`, etc.) for dependency names.
3. **Display the profile** — prints detected languages, stacks and signals so you can verify before proceeding.
4. **Check for an existing profile** — if a profile already exists and `--refresh` was not passed, prompts for confirmation before overwriting.
5. **Discover capabilities** — queries the catalog for all entries whose trigger signals intersect with the detected profile. Ranks up to 10 results by confidence score.
6. **Present the selection** — shows a numbered list. Enter `all`, `none`, or comma-separated numbers (e.g. `1,3`).
7. **Provision selected capabilities** — writes MCP config entries, registers approved skills, checks CLI tools.
8. **Persist the profile** — stores the profile JSON and provisioning results in SQLite. Retains the last 3 profiles per project.
9. **Log the session** — records a `setup_session` event with outcome (`completed` / `partial` / `failed`).

---

## Flags

| Flag | Behaviour |
|---|---|
| `--status` | Show the stored profile for the current project without running detection |
| `--refresh` | Re-run detection without the overwrite confirmation prompt |
| `--dry-run` | Print what would be detected and provisioned; write nothing |

---

## The capability catalog

The catalog ships with Noevyra and covers these signal values:

| Capability | Kind | Signals | Network |
|---|---|---|---|
| GitHub MCP | mcp | `github`, `github-actions`, `ci-cd` | network |
| Documentation MCP | mcp | `go`, `rust`, `python`, `typescript`, `javascript`, `api` | network |
| Database MCP | mcp | `database`, `sql`, `postgres`, `mysql`, `sqlite`, `migration` | network |
| Browser MCP | mcp | `web`, `frontend`, `ui`, `e2e`, `playwright` | network |
| Security Review Skill | skill | `security`, `auth`, `secret`, `token`, `webhook`, `payment` | network |
| Figma MCP | mcp | `figma`, `design`, `ui`, `component`, `frontend` | network |
| API Contract Skill | skill | `api`, `backend`, `schema`, `contract`, `grpc`, `rest` | local |
| Container Ops Skill | skill | `containers`, `docker`, `compose` | network |
| Test Strategy Skill | skill | `test`, `bug`, `refactor`, `ci`, `release` | local |
| ripgrep | cli | `code`, `refactor`, `bug`, `project` | local |

Any skills you have previously trusted with `brain skills trust` and whose keyword matches your profile are also included, with confidence 0.95.

---

## MCP provisioning

When you select an MCP capability, Brain writes an `mcpServers` entry to the config file for every agent client it detects in `PATH`:

| Agent | Binary detected | Config file written |
|---|---|---|
| Claude | `claude` | `~/.claude.json` |
| Codex | `codex` | `~/.codex/config.json` |
| Cursor | `cursor-agent` | `~/.cursor/mcp.json` |
| Kiro | `kiro` | `~/.kiro/settings/mcp.json` |

The write is atomic (write-to-temp + rename) and idempotent — running `brain setup` again updates the entry in place without duplicating it.

---

## Inspecting a stored profile

```sh
brain setup --status
```

Shows:
- Project ID and root
- Detection timestamp
- Detected languages, stacks and signals
- Provisioned capabilities and their outcomes

---

## Example session

```
$ brain setup

Detected profile:
  Languages:  go, typescript
  Stacks:     gin, react, pgx
  Signals:    ci-cd, database, git, github, web

Available capabilities:
   1. [mcp][network] GitHub MCP — Inspect issues, pull requests, commits and CI state.
   2. [mcp][network] Documentation MCP — Retrieve current primary documentation.
   3. [mcp][network] Database MCP — Inspect schema and query results.
   4. [mcp][network] Browser MCP — Verify browser behavior and UI state.
   5. [skill][network] Security Review Skill — Threat-model and unsafe-input checklist.
   6. [skill][local] API Contract Skill — Request/response and compatibility constraints.
   7. [skill][local] Test Strategy Skill — Smallest useful unit, integration, E2E checks.
   8. [cli][local] ripgrep — Fast code search.

Select capabilities to provision (all/none/1,2,3): 1,3,8

  Skill: Security Review Skill
  Structured checklist for auth, secrets, injection, and webhook security.
  Approve? [y/N]: (skipped — not selected)

Provisioning summary:
  GitHub MCP                     provisioned
  Database MCP                   provisioned
  ripgrep                        failed (rg not found in PATH; install it manually)

Setup complete. Run 'brain setup --status' to inspect the stored profile.
```

---

## Switching agents

Because the profile is stored in the local database (not in any agent session), switching from Claude to Kiro requires no re-setup:

```sh
brain run kiro  "add a search endpoint"
brain run claude "add a search endpoint"
```

Both get the same context pack, which includes the stored profile.

---

## Refreshing after project changes

When you add a major dependency or change your CI setup, re-run detection:

```sh
brain setup --refresh
```

The old profile is kept (up to 3 versions) and a new one is stored as the current.
