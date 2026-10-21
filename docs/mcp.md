# MCP integration

Noevyra exposes an MCP (Model Context Protocol) server over stdio. Connect it once per agent client and every session gets the full intelligence layer.

---

## Get the configuration

```sh
brain mcp-config
```

Output:

```json
{
  "mcpServers": {
    "brain": {
      "command": "/Users/you/.local/bin/brain",
      "args": ["mcp"]
    }
  }
}
```

The `command` is the absolute path of your installed `brain` binary. Always use an absolute path — relative paths break when the agent starts from a different directory.

---

## Connect by agent

### Claude

```sh
claude mcp add --scope user brain -- /absolute/path/to/brain mcp
```

Or add manually to `~/.claude.json`:

```json
{
  "mcpServers": {
    "brain": {
      "command": "/absolute/path/to/brain",
      "args": ["mcp"]
    }
  }
}
```

### Kiro

Run `brain mcp-config` and paste the output into `~/.kiro/settings/mcp.json`. If the file already has other entries, merge the `mcpServers` key.

`brain setup` writes this file automatically when it detects `kiro` in `PATH`.

### Cursor

Add the `brain mcp-config` output to your Cursor MCP configuration file (`~/.cursor/mcp.json`).

### Codex

```sh
codex mcp add brain -- /absolute/path/to/brain mcp
```

---

## Project identity

The MCP server uses the **working directory** of the process that launched it to determine project scope. Start the Brain MCP process from your project's Git root — or ensure your agent client does so — to correctly scope project memories.

---

## Available tools

### `brain_advise(task, budgetTokens?)`

The most powerful tool. Returns a full intelligence pack:

```json
{
  "id": "pack-id-for-feedback",
  "task": "add rate limiting",
  "profile": { "languages": ["go"], "stacks": ["gin"], "signals": ["database", "api"] },
  "reasoning": {
    "approach": ["…"],
    "risks": ["…"],
    "verificationGates": ["…"]
  },
  "context": [
    { "kind": "memory", "content": "…", "score": 98.0 },
    { "kind": "knowledge", "title": "…", "content": "…", "source": "https://…" }
  ],
  "recommendations": [
    { "kind": "mcp", "ref": "github", "reason": "…", "confidence": 0.88 }
  ],
  "metrics": { "budgetTokens": 1200, "estimatedTokens": 840 }
}
```

`budgetTokens` — optional integer 128–16000. Omit to use the adaptive budget.

### `brain_profile()`

Returns the detected project profile (languages, stacks, signals, file counts).

### `brain_context(task, limit?)`

Returns verified memories relevant to the task. Simpler and faster than `brain_advise` when you only need memories.

### `brain_recall(query, limit?)`

Full-text search across verified memories in the current project and personal scope.

### `brain_remember(content, type?)`

Proposes a project-scoped memory as a **candidate** — it is not verified until you run `brain approve <id>` from the CLI. Agents cannot write verified memories directly.

`type` — one of: `decision` · `identity` · `semantic` · `episodic` · `project` · `procedural`

### `brain_feedback(packId, outcome, tokensUsed?, notes?)`

Records the usefulness of a pack returned by `brain_advise`. 

`outcome` — `helpful` | `partial` | `harmful`

After ≥3 helpful outcomes, the compiler adapts its default token budget for future packs.

### `brain_finish_task(summary, status, evidence?)`

Logs a task outcome event. Does not automatically create a memory — that requires explicit `brain_remember` or `brain approve`.

`status` — `completed` | `blocked` | `failed`

### `brain_handoff(objective, completed?, remaining?, evidence?)`

Records a structured handoff. Useful when an agent suspends mid-task and another agent resumes.

---

## Protocol notes

- Transport: newline-delimited JSON-RPC 2.0 over stdio
- No `console.log` on stdout — all log output goes to stderr
- Supports protocol versions: `2024-11-05`, `2025-03-26`, `2025-06-18`, `2025-11-25`
- Agent writes (`brain_remember`) create candidates only — they require explicit CLI approval
- The implementation intentionally supports a small, stable subset of MCP methods

---

## Updating MCP config after a `brain update`

The `command` path in your MCP config must point to the actual binary. Because `brain update` replaces the binary at the same path, no config change is needed after an update. Restart any running `brain mcp` processes to pick up the new binary.
