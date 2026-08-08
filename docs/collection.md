# <img src="assets/icons/collection.svg" width="28" height="28" align="absmiddle" alt="" /> Collection and batching

How the agent notices change on a laptop, turns it into events, and delivers them reliably.

> **In one breath:** Collectors only **enqueue**. A durable ring owns retry. Watchers make it fast; polls make it correct.

<p align="center">
  <a href="#big-picture">Big picture</a>
  &nbsp;·&nbsp;
  <a href="#hybrid-design">Hybrid design</a>
  &nbsp;·&nbsp;
  <a href="#what-we-collect">What we collect</a>
  &nbsp;·&nbsp;
  <a href="#how-detection-works-per-os">Per OS</a>
  &nbsp;·&nbsp;
  <a href="#batch--upload">Batch &amp; upload</a>
</p>

---

## Big picture

<p align="center">
  <img src="assets/collection-pipeline.svg" alt="Collectors enqueue into a durable ring, then batch, compress, and upload to the Agent API" width="980" />
</p>

| Rule | Meaning |
|------|---------|
| Collectors never call the network | Isolation — one upload path |
| Mixed event types share a batch | Quiet and busy hosts both work |
| Failed uploads stay in the ring | Survive offline / flaky Wi‑Fi |
| Watcher dies → poll continues | Degrade, don’t fail |

---

## Hybrid design

<p align="center">
  <img src="assets/hybrid-model.svg" alt="OS watcher and periodic poll both feed a monitor, then the durable ring" width="980" />
</p>

Same pattern for **process**, **network summary**, and **security**:

1. **Watcher** (when the OS allows) → low-latency wake or typed event  
2. **Poll** → silent baseline, catch misses, work alone if needed  
3. **Monitor** → dedupe / fingerprint / enrich  
4. **Ring** → batch → optional zstd → HTTPS  

**Connection samples** (`network_connection`) are poll-only: silent baseline, then newly seen ESTABLISHED TCP sockets (capped per poll).

Deep OS detail is in the diagrams below.

---

## What we collect

| <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" /> Signal | Events | Default cadence |
|---|--------|-----------------|
| Host | `client_details` | Once at start + every 60s |
| Network | `network_summary` | On change (debounced) + 60s heartbeat |
| Connections | `network_connection` | 15s poll · new ESTABLISHED TCP (pid, ports, remote) |
| Activity | `action_summary` | Sample 5s · emit 60s |
| Processes | `process_start` / `process_exit` | Watcher + 10s poll |
| Security | `driver_load` / `service_install` / `registry_persistence` | Watcher wake + 30s poll |
| AI inventory | `known_ai_app` | 60s poll + wake from process events |

Turn off: processes `PROCESS_INTERVAL=0` · security `SECURITY_INTERVAL=0` · AI inventory `KNOWN_AI_INTERVAL=0` · connections `CONNECTION_INTERVAL=0` (all `TRUSTEDGE_AGENT_*`).

AI inventory covers verified catalog products across:

| Category | Examples |
|----------|----------|
| GUI apps | Cursor, Claude |
| CLI agents | Claude Code, Codex, Gemini CLI, Copilot CLI |
| Local model runtimes | Ollama (app or Docker), llama.cpp-family |
| IDE extensions | GitHub Copilot, Continue, Cline, Roo (Cursor / VS Code hosts) |

Identity uses catalog matching with evidence (path, bundle ID, signing, package, extension ID, Docker image). Collectors stay local — only the upload path sends data over the network.

---

## How detection works per OS

### <img src="assets/icons/concurrency.svg" width="20" height="20" align="absmiddle" alt="" /> Processes

<p align="center">
  <img src="assets/detect-process.svg" alt="Linux netlink, Windows ETW, and macOS Endpoint Security feeding ProcessMonitor" width="980" />
</p>

| OS | How we hear about start / exit |
|----|--------------------------------|
| **Linux** | Netlink process connector → enrich from `/proc` |
| **Windows** | ETW Microsoft-Windows-Kernel-Process (admin recommended) |
| **macOS** | Endpoint Security `EXEC` / `EXIT` (CGO + entitlement) |

**Always:** a timed poll reconciles the process table (first poll is silent). Cap: 100 starts per poll.

### <img src="assets/icons/flow.svg" width="20" height="20" align="absmiddle" alt="" /> Network

<p align="center">
  <img src="assets/detect-network.svg" alt="Linux netlink route, Windows fingerprint poll, and macOS AF_ROUTE feeding a debounced network summary" width="980" />
</p>

| OS | How we notice link / IP change |
|----|--------------------------------|
| **Linux** | Netlink route (`NEW/DEL LINK` · `ADDR`) |
| **Windows** | Interface fingerprint every ~2s (same `link_change` path) |
| **macOS** | AF_ROUTE socket (add / delete / addr / ifinfo) |

Then: **debounce 2s** → summary fingerprint → emit. Heartbeats **always** post so liveness is visible even when posture is unchanged.

### <img src="assets/icons/flow.svg" width="20" height="20" align="absmiddle" alt="" /> Connection samples

Poll-only (`TRUSTEDGE_AGENT_CONNECTION_INTERVAL`, default **15s**):

| Behavior | Detail |
|----------|--------|
| Scope | Newly observed **ESTABLISHED** TCP sockets with pid + remote |
| Baseline | First poll is silent (seeds seen set) |
| Cap | At most **40** new sockets per poll |
| Payload | `pid`, `comm`, local/remote addr+port, optional reverse-DNS hostname |
| Disable | `TRUSTEDGE_AGENT_CONNECTION_INTERVAL=0` |

This is **not** a full connection-table dump — only incremental samples after the baseline.

### <img src="assets/icons/lock.svg" width="20" height="20" align="absmiddle" alt="" /> Security lifecycle

<p align="center">
  <img src="assets/detect-security.svg" alt="Linux poll-only, Windows registry notify, and macOS kqueue feeding SecurityMonitor fingerprint diffs" width="980" />
</p>

| OS | Event-driven wake | What Poll() reports |
|----|-------------------|---------------------|
| **Linux** | — (poll only) | Platform artifacts on interval |
| **Windows** | Registry notify on Run/RunOnce + Services | Drivers, services, Run keys |
| **macOS** | `kqueue` on LaunchAgents / LaunchDaemons dirs | kexts, LaunchDaemons, LaunchAgents |

The watcher **never invents** event types — it only wakes `Poll()`. First poll seeds a silent baseline.

---

## Batch & upload

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '14px', 'fontFamily': 'arial', 'lineColor': '#64748B'}}}%%
flowchart LR
    R[("Durable ring")] --> F{"Flush?"}
    F -->|size ≥ 32| P["POST /v1/events"]
    F -->|every 2s| P
    F -->|shutdown ~3s| P
    P -->|202| A["Ack · remove"]
    P -->|fail| B["Backoff · keep"]
    P -->|401| AUTH["Re-register once · retry"]

    classDef ring fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef decide fill:#FEF3C7,stroke:#D97706,stroke-width:2px,color:#78350F
    classDef post fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95
    classDef ok fill:#D1FAE5,stroke:#059669,stroke-width:2px,color:#065F46
    classDef err fill:#FEE2E2,stroke:#DC2626,stroke-width:2px,color:#7F1D1D

    class R ring
    class F decide
    class P post
    class A ok
    class B,AUTH err
```

| Topic | Behavior |
|-------|----------|
| **Compress** | zstd only when smaller than raw JSON |
| **Auth** | Device token in OS keyring; concurrent `401`s share one re-register |
| **Capacity** | Ring holds 4096; overwrite oldest when full |
| **Privacy** | No titles, keystrokes, screenshots, or SSIDs; connection samples are incremental/capped |

Concurrency: each collector runs in its own loop — a slow public-IP lookup never blocks process events.

---

## Dig deeper

| | Doc |
|---|-----|
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Architecture](architecture.md) — lifecycle & auth sequence |
| <img src="assets/icons/config.svg" width="18" height="18" align="absmiddle" alt="" /> | [Configuration](configuration.md) — every env var |
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | [Agent guide](agent.md) — install, credentials, privacy |
| <img src="assets/icons/upload.svg" width="18" height="18" align="absmiddle" alt="" /> | [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) — payload fields |
