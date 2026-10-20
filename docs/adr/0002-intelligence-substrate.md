# ADR 0002: Deterministic intelligence substrate

- Status: proposed for v0.2
- Date: 2026-10-20

## Context

Memory alone still leaves every agent to rediscover the project, select tools and expand large amounts of context. The Brain should carry that repeated reasoning across agents without becoming another hosted chatbot or requiring an expensive model call.

## Decision

Build a local, deterministic intelligence layer around four components:

1. **Source registry and harvester.** The user explicitly trusts an HTTPS source once. Online sync retrieves bounded text with conditional requests, records provenance and hashes, and never executes remote content.
2. **Project profiler.** A bounded scan detects languages, manifests, stacks and operational signals. Repository content is evidence, not instruction text.
3. **Context compiler.** Verified memories and relevant source chunks are ranked, deduplicated and truncated to an explicit token budget. Every pack has an ID and auditable metrics.
4. **Capability router.** Task and project signals produce explainable skill, MCP and CLI recommendations. Recommendations do not install or connect anything automatically.

Agents access the same result through `brain advise`, the `brain run` wrappers or `brain_advise` over MCP. They report outcomes through `brain feedback` or `brain_feedback`. Helpful or harmful feedback adjusts bounded knowledge quality scores for later packs.

## Trust and safety invariants

- Network sync is opt-in: the user registers sources and explicitly installs the daily schedule. The timer only contacts those registered sources.
- Sources require HTTPS, except localhost for development tests; credentials and cross-host redirects are rejected.
- Responses are capped at 2 MiB and stored as inert text with URL and content hash provenance.
- Remote text cannot become a command, skill installation, verified personal memory or higher-priority instruction.
- Scheduled refresh never invokes the skills CLI; third-party code checks remain a separate explicit action.
- An agent cannot add a knowledge source through MCP.
- Tool recommendations explain their matching signals and require normal user/client setup.
- Pack size is enforced before output; estimated token use and dropped items are returned.

## Efficiency loop

For each pack the Brain records candidate count, selected count, deduplication, drops, budget and estimated output tokens. Feedback links the outcome to the included knowledge. Ranking changes are clamped so one result cannot permanently dominate. This is online optimization of retrieval, not model training.

Initial success criteria:

- No model or embedding service is required.
- The same CLI binary works for any Git project and any MCP-compatible agent.
- A pack never exceeds its requested evidence token budget.
- Repeated external knowledge is deduplicated.
- Every external fragment retains its source URL.
- All trust transitions and feedback remain auditable in SQLite.

## Consequences

Lexical retrieval will miss some semantic matches, and static capability rules need curation. These limits are preferable to opaque, high-token orchestration in the first iteration. Embeddings or a small reranker may later be added behind benchmarks, but they must beat the deterministic baseline on quality, latency and token use.
