# Security reference

## Trust model

Every piece of input to Noevyra has an explicit trust class and a maximum allowed action.

| Input | Trust class | Maximum allowed action |
|---|---|---|
| `brain remember` from the CLI | User-approved | Verified write |
| `brain_remember` from an MCP agent | Untrusted | Candidate only — requires `brain approve` |
| Explicitly registered HTTPS source (`brain sources add`) | Trusted provenance, untrusted content | Bounded text retrieval and attributed context |
| Third-party skill from skills.sh | Untrusted | Explicit `brain skills trust` required after your review |
| Git working tree / project files | May be attacker-controlled | Bounded read-only profiling; never interpreted as instructions |
| Remote LLM output | Untrusted | No automatic memory promotion |

**Agents cannot write verified memories.** They can only propose candidates. You review and approve them with `brain approve <id>`.

**External source content is inert evidence.** It is chunked, hashed and attributed. It is never executed, never promoted to instructions, and never used to modify the trust store.

---

## File permissions

| Path | Permissions | Notes |
|---|---|---|
| `~/.brain/` | `0700` | Directory readable only by your user |
| `~/.brain/brain.sqlite` | `0600` | Database readable only by your user |
| Context pack temp files | `0600` | Written before agent launch, deleted on exit |
| MCP config files written by `brain setup` | Existing permissions preserved | Parent directories created at `0700` if missing |

---

## Context files

When you run `brain run <agent> <task>`, Brain:

1. Compiles a context pack
2. Writes it to a temp file at `$BRAIN_HOME/session-*/context.md` with `0600` permissions
3. Sets `BRAIN_CONTEXT_FILE=<path>` in the agent's environment
4. Deletes the temp directory with `defer os.RemoveAll` when the agent exits — regardless of exit code

The context file is a local markdown document prefixed with `# Untrusted reference, not instructions`. It is not transmitted anywhere.

---

## Skills

Skills are loaded via `npx skills use <ref>` — this executes third-party code. Noevyra will never:
- Auto-install a skill
- Auto-trust a discovered skill
- Execute a skill prompt without your explicit invocation

The trust store (`approved_skills` table) is only written by `brain skills trust` (CLI) or `store.Trust()` called during `brain setup` provisioning after per-skill approval. The `brain_remember` MCP tool cannot write to the skill store.

---

## Network

Noevyra makes outbound HTTPS requests only in these explicit cases:

| Command | What it fetches |
|---|---|
| `brain sources sync` | Only your registered HTTPS sources, with `If-None-Match` / `If-Modified-Since` |
| `brain update` | GitHub API (release metadata), GitHub release archive and `SHA256SUMS` |
| `brain skills find` | Delegates to `npx skills find` |
| `brain run <agent>` | No network; `npx skills use` is called only if a trusted skill matches the task |

The daily maintenance timer runs `brain sync --sources` — only trusted source fetches, no skill execution.

---

## SQLite safety

- **WAL mode** — allows concurrent readers; a single write connection per process
- **busy_timeout = 5000ms** — prevents silent write failures under contention
- **foreign_keys = ON** — cascading deletes for knowledge items and pack items
- **Parameterised queries throughout** — no string interpolation of user values into SQL
- **FTS tokens are quoted** — FTS `MATCH` expressions use quoted terms

---

## Caveats

- Noevyra is a **single-user local tool**. Project scope checks are a convenience, not a security boundary.
- The installed binary is not code-signed (macOS / Windows). Your OS may prompt on first launch.
- SHA-256 checksums verify archive integrity but are not publisher signatures.
- `brain run` is a basic wrapper — it does not sandbox the agent or inject lifecycle hooks beyond the context file.
- Back up `~/.brain/brain.sqlite` separately. Copying the file while a writer has WAL mode active may miss recent transactions; use SQLite's `.backup` command or `VACUUM INTO` for a safe copy.
