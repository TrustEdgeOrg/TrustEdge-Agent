# <img src="assets/agent-icon.svg" alt="" width="32" height="32" align="absmiddle" /> TrustEdge Agent docs

Interview-ready map of how the endpoint agent collects, buffers, and uploads telemetry.

> **Elevator pitch:** A thin cross-platform Go agent — hybrid OS watchers + poll reconciliation, durable offline ring, keyring-backed auth, and zstd when it wins — feeding TrustEdge detection over HTTPS.

---

## Start here (2-minute path)

| Order | Open this | Why |
|------:|-----------|-----|
| 1 | [Architecture](architecture.md) | End-to-end pipeline + auth recovery |
| 2 | [Platform watchers](watchers-overview.md) | The hybrid design story (best interview slide) |
| 3 | [Collection & batching](collection.md) | Collectors, flush rules, concurrency |
| 4 | [Agent guide](agent.md) | Install, platforms, privacy, credentials |

---

## All docs

| | Doc | Audience |
|---|-----|----------|
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Architecture](architecture.md) | System design walkthrough |
| <img src="assets/icons/platforms.svg" width="18" height="18" align="absmiddle" alt="" /> | [Platform watchers](watchers-overview.md) | Linux · Windows · macOS signal paths |
| <img src="assets/icons/collection.svg" width="18" height="18" align="absmiddle" alt="" /> | [Collection & batching](collection.md) | Collectors → ring → upload |
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | [Agent guide](agent.md) | Run locally · credentials · privacy |
| <img src="assets/icons/config.svg" width="18" height="18" align="absmiddle" alt="" /> | [Configuration](configuration.md) | Every env var |
| <img src="assets/icons/test.svg" width="18" height="18" align="absmiddle" alt="" /> | [Test process cmdline](testing-process-cmdline.md) | Local capture of `cmdline` |

API schemas live in [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md).

---

## Design principles (say these out loud)

1. **Watchers for latency, polls for correctness** — degrade to poll-only, never hard-fail collection.  
2. **Collectors only enqueue** — one durable ring owns retry, backoff, and offline survival.  
3. **Tokens in the OS keyring** — device ID on disk; concurrent `401`s share one re-register.  
4. **Privacy by omission** — no titles, keystrokes, screenshots, or full connection tables.

Repo root: [README](../README.md).
