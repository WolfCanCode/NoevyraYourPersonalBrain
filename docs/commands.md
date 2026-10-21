# Command reference

`BRAIN_HOME` overrides `~/.brain` for the store location.

---

## brain init

```sh
brain init [--schedule]
```

Creates `~/.brain/brain.sqlite` and **auto-seeds 2 737 embedded skills** from the bundled AAS catalog. `--schedule` registers a daily sync timer (macOS `launchd` / Linux `systemd`).

---

## brain setup

```sh
brain setup [--refresh] [--dry-run] [--status] [--no-live-skills]
```

Detects project stack → searches skill catalog per technology → interactive TUI selection → provisions MCPs and skills → persists profile.

| Flag | Effect |
|---|---|
| `--status` | Show stored profile; no detection or provisioning |
| `--refresh` | Re-detect without overwrite confirmation |
| `--dry-run` | Preview only — no writes |
| `--no-live-skills` | Skip catalog search (useful offline) |

---

## brain run

```sh
brain run claude|codex|cursor|kiro [task]
```

Compiles a context pack, writes it to a `0600` temp file, sets `BRAIN_CONTEXT_FILE`, and launches the agent. File is deleted on agent exit.

---

## brain advise

```sh
brain advise "task" [--budget N]
```

Returns a full JSON intelligence pack: profile, reasoning frame, ranked evidence, capability recommendations. `--budget` range: 128–16 000 tokens (default adapts from past outcomes).

---

## brain remember

```sh
brain remember "text" [--type TYPE] [--scope project|personal]
```

Types: `decision` · `identity` · `semantic` · `episodic` · `project` · `procedural`

---

## brain recall / brain context

```sh
brain recall "query"
brain context "task"
```

FTS5 search over verified memories. `context` wraps the result in a task-labelled pack.

---

## brain feedback

```sh
brain feedback <pack-id> --outcome helpful|partial|harmful [--tokens N]
```

Records pack quality. After ≥ 3 helpful outcomes the compiler adapts its default budget.

---

## brain profile

```sh
brain profile
```

Read-only project detection. Returns detected languages, stacks, signals, and file counts as JSON.

---

## brain skills

```sh
brain skills find "query"       # search 2 737 embedded skills
brain skills trust owner/repo@skill --keyword TERM
brain skills list
```

`find` searches the local embedded catalog instantly. Falls back to `npx skills find` if the store is unavailable. `trust` registers a skill for keyword-based matching in `brain run`.

---

## brain sources

```sh
brain sources seed              # register AAS + Go, OWASP, MCP starter sources
brain sources add URL --name N  # trust an additional HTTPS source
brain sources sync              # fetch all registered sources + refresh catalog
brain sources list
brain sources pause <id>
```

---

## brain update

```sh
brain update          # download latest release, verify SHA-256, replace binary
brain update --check  # report available version without installing
```

Atomically replaces the running binary (write-then-rename). Restart `brain mcp` processes after update.

---

## brain approve

```sh
brain approve <memory-id>
```

Promotes an agent-proposed candidate memory to verified.

---

## brain sync

```sh
brain sync [--sources] [--skills] [--online]
```

Local maintenance (expire stale memories). `--sources` fetches registered sources. `--online` = `--sources --skills`.

---

## brain mcp / brain mcp-config

```sh
brain mcp         # start MCP server over stdio
brain mcp-config  # print JSON config for agent clients
```

---

## brain doctor

```sh
brain doctor
```

Shows version, platform, tool availability (git, claude, codex, cursor-agent, kiro, npx), scheduler status.

---

## brain status / brain digest / brain version

```sh
brain status          # memory counts, project ID, scheduler
brain digest [--days N]
brain version
```
