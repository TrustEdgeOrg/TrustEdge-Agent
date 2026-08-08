# <img src="assets/trustedge-icon.svg" alt="" width="32" height="32" align="absmiddle" /> TrustEdge Agent docs

How the endpoint agent collects, buffers, and uploads telemetry.

> A thin cross-platform Go agent — hybrid OS watchers + poll reconciliation, durable offline ring, keyring-backed auth, and zstd when it wins — feeding TrustEdge detection over HTTPS.

---

## Docs

| | Doc | Purpose |
|---|-----|---------|
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Architecture](architecture.md) | Pipeline, lifecycle, auth recovery |
| <img src="assets/icons/collection.svg" width="18" height="18" align="absmiddle" alt="" /> | [Collection & batching](collection.md) | Collectors, per-OS detection, batching |
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | [Agent guide](agent.md) | Install, platforms, privacy, credentials |
| <img src="assets/icons/config.svg" width="18" height="18" align="absmiddle" alt="" /> | [Configuration](configuration.md) | Every environment variable |

API schemas: [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md).

---

## Design principles

1. **Watchers for latency, polls for correctness** — degrade to poll-only, never hard-fail collection.  
2. **Collectors only enqueue** — one durable ring owns retry, backoff, and offline survival.  
3. **Tokens in the OS keyring** — device ID on disk; concurrent `401`s share one re-register.  
4. **Privacy by omission** — no titles, keystrokes, screenshots, or full connection tables.

Repo root: [README](../README.md).
