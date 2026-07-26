# <img src="assets/icons/platforms.svg" width="28" height="28" align="absmiddle" alt="" /> Platform watchers

How `trustedge-agent` gets low-latency change signals on Linux, Windows, and macOS — without sacrificing correctness when the OS API is unavailable.

> **Design in one sentence:** Watchers for latency, polls for correctness; if a watcher dies we degrade, we don’t fail.

<p align="center">
  <img src="assets/icons/flow.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#shared-model">Model</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/platforms.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#platform-matrix">Matrix</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#how-each-domain-uses-the-signal">Domains</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#talking-points-60-seconds">Talking points</a>
</p>

> **Also see**
> <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" /> [Collection](collection.md) — visual per-OS detection diagrams
> · <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" /> [Agent guide](agent.md#platforms)
> · <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" /> [Docs hub](README.md)

---

## <img src="assets/icons/flow.svg" width="22" height="22" align="absmiddle" alt="" /> Shared model

Every monitored domain follows the same hybrid loop:

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart LR
    subgraph OS ["Host OS"]
        API["Kernel / platform API"]
    end

    subgraph AGENT ["TrustEdge Agent"]
        direction TB
        W["Event-driven watcher"]
        P["Periodic poll"]
        M["Monitor<br/>dedupe · baseline · enrich"]
        R["Durable ring → batch → API"]
    end

    API -->|"start / link / persistence change"| W
    W -->|"Change or wake"| M
    P -->|"reconcile misses"| M
    M --> R

    classDef os fill:#F1F5F9,stroke:#64748B,stroke-width:2px,color:#0F172A
    classDef watch fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef poll fill:#FFEDD5,stroke:#EA580C,stroke-width:2px,color:#7C2D12
    classDef mon fill:#E0E7FF,stroke:#4F46E5,stroke-width:2px,color:#312E81
    classDef out fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95

    class API os
    class W watch
    class P poll
    class M mon
    class R out
```

| Path | Role |
|------|------|
| **Watcher** | Near-real-time signal when the platform exposes one |
| **Poll** | Source of truth — silent baseline, catch misses, works alone if watcher is `nil` |
| **Monitor** | Dedup, enrich, and decide what to enqueue |

---

## <img src="assets/icons/platforms.svg" width="22" height="22" align="absmiddle" alt="" /> Platform matrix

| Domain | Linux | Windows | macOS |
|--------|-------|---------|-------|
| **Process** | Netlink connector (`PROC_EVENT_EXEC` / `EXIT`) | ETW kernel-process provider | Endpoint Security (`NOTIFY_EXEC` / `EXIT`, CGO) |
| **Network** | Netlink route (`NEW/DEL LINK` · `ADDR`) | Interface fingerprint poll (~2s) | AF_ROUTE socket |
| **Security** | Poll only | `RegNotifyChangeKeyValue` on Run/RunOnce + Services | `kqueue` `EVFILT_VNODE` on LaunchAgents / LaunchDaemons |

Unavailable watcher (permissions, CGO off, missing entitlement) → **poll-only**; collection keeps running.

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '13px', 'fontFamily': 'arial'}}}%%
flowchart TB
    subgraph L ["Linux"]
        LP["Process: netlink connector"]
        LN["Network: netlink route"]
        LS["Security: poll only"]
    end
    subgraph W ["Windows"]
        WP["Process: ETW"]
        WN["Network: fingerprint poll"]
        WS["Security: registry notify"]
    end
    subgraph M ["macOS"]
        MP["Process: Endpoint Security"]
        MN["Network: AF_ROUTE"]
        MS["Security: kqueue"]
    end

    classDef linux fill:#ECFDF5,stroke:#059669,stroke-width:2px,color:#064E3B
    classDef win fill:#EFF6FF,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef mac fill:#FAF5FF,stroke:#7C3AED,stroke-width:2px,color:#4C1D95

    class LP,LN,LS linux
    class WP,WN,WS win
    class MP,MN,MS mac
```

---

## <img src="assets/icons/collection.svg" width="22" height="22" align="absmiddle" alt="" /> How each domain uses the signal

### Process — emit typed events

```text
OS notify  →  ProcessWatcher  →  Change{process_start|process_exit}
                                      ↓
                               ProcessMonitor.Observe()  →  enqueue
Periodic ProcessInterval poll  →  ProcessMonitor.Poll()  →  enqueue (misses)
```

- Watcher carries PID / image / cmdline when the OS provides them.
- Poll diffs the process table; first poll seeds `seen` **silently**.

### Network — wake, then snapshot

```text
OS link/addr change  →  platform watcher  →  link_change signal
                                              ↓
                                    debounce → summary fingerprint → enqueue
Heartbeat every NetworkInterval  →  always enqueue (liveness)
```

Windows has no socket watcher here — the same `link_change` path is driven by a short fingerprint poll.

### Security — wake only, poll decides

```text
Registry / plist change  →  SecurityWatcher  →  wake (debounced ~250ms)
                                                    ↓
                                         SecurityMonitor.Poll()  →  enqueue diffs
Periodic SecurityInterval poll   →  same Poll() path
```

The watcher never invents `driver_load` / `service_install` / `registry_persistence` — it only asks Poll to re-scan.

---

## <img src="assets/icons/agent.svg" width="22" height="22" align="absmiddle" alt="" /> Talking points (60 seconds)

1. **Hybrid by design** — event path for speed, poll path for correctness and cold start.  
2. **Degrade, don’t fail** — watcher setup errors log and continue; intervals still fire.  
3. **Platform-native APIs** — netlink / ETW / Endpoint Security / kqueue / registry notify.  
4. **One enqueue path** — all collectors feed the durable ring; upload and auth recovery stay shared.

---

## <img src="assets/icons/architecture.svg" width="22" height="22" align="absmiddle" alt="" /> Related docs

| | Doc | Purpose |
|---|-----|---------|
| <img src="assets/icons/collection.svg" width="18" height="18" align="absmiddle" alt="" /> | [Collection & batching](collection.md) | Collector payloads, flush rules, concurrency |
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | [Agent guide](agent.md) | CGO builds, entitlements, credentials |
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Architecture](architecture.md) | Lifecycle and upload path |
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Docs hub](README.md) | Interview start path |
