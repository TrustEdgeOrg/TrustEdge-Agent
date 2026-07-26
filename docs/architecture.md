# <img src="assets/icons/architecture.svg" width="28" height="28" align="absmiddle" alt="" /> Architecture

The `trustedge-agent` binary collects endpoint telemetry and POSTs it to [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API). The API can publish to a stream for TrustEdge detection.

> **In one breath:** Collect on the device → durable batch → optional zstd → HTTPS upload → ingest → stream → detect → alert.

<p align="center">
  <img src="assets/icons/flow.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#high-level-flow">Flow</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#agent-lifecycle">Lifecycle</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#collectors">Collectors</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/queue.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#batching">Batching</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/compress.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#compression">Compression</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/lock.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#authentication">Auth</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/layout.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#project-layout">Layout</a>
</p>

> **Also see**
> <img src="assets/icons/platforms.svg" width="16" height="16" align="absmiddle" alt="" /> [Platform watchers](watchers-overview.md)
> · <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" /> [Collection](collection.md)
> · <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" /> [Agent guide](agent.md)
> · <img src="assets/icons/config.svg" width="16" height="16" align="absmiddle" alt="" /> [Configuration](configuration.md)
> · <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" /> [Docs hub](README.md)

---

## Ecosystem

| | Component | Repo | Role |
|---|-----------|------|------|
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | `trustedge-agent` | [TrustEdge-Agent](https://github.com/TrustEdgeOrg/TrustEdge-Agent) | Collect · batch · compress · secure upload |
| <img src="assets/icons/upload.svg" width="18" height="18" align="absmiddle" alt="" /> | `trustedge-agent-api` | [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) | Register · ingest · optional Kafka publish |
| <img src="assets/icons/flow.svg" width="18" height="18" align="absmiddle" alt="" /> | TrustEdge | [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) | Detection · alerts · UI |

---

## <img src="assets/icons/flow.svg" width="22" height="22" align="absmiddle" alt="" /> High-level flow

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart LR
    subgraph EP ["Endpoint"]
        DEV(["Device"])
    end

    subgraph AG ["TrustEdge Agent"]
        direction LR
        COL["Collect"] --> BAT["Batch"] --> ZIP["Compress"] --> SND["Secure upload"]
    end

    subgraph CL ["TrustEdge"]
        direction LR
        API["Agent API"] --> STR["Stream"] --> DET["Detection"] --> ALT["Alert"]
    end

    DEV --> COL
    SND -->|HTTPS + zstd| API

    classDef endpoint fill:#F8FAFC,stroke:#475569,stroke-width:2px,color:#0F172A
    classDef agent fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef cloud fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95
    classDef stream fill:#FEF9C3,stroke:#CA8A04,stroke-width:2px,color:#713F12
    classDef compress fill:#FFEDD5,stroke:#EA580C,stroke-width:2px,color:#7C2D12

    class DEV endpoint
    class COL,BAT,SND agent
    class ZIP compress
    class API,DET,ALT cloud
    class STR stream
```

| Stage | What happens |
|-------|--------------|
| **Collect** | Five concurrent sources: device, network, activity, processes, security |
| **Batch** | Events land in a **durable ring**; flushed by size, timer, or shutdown |
| **Compress** | Optional **zstd** when smaller than raw JSON |
| **Secure upload** | HTTPS `POST /v1/events` with the device bearer token |
| **Agent API** | Decompress · validate · accept (`202`) |
| **Stream** | Optional publish (e.g. Kafka `trustedge.agent.events`) |
| **Detection → Alert** | TrustEdge rules raise alerts |

Timers and flush edge cases: [Collection & batching](collection.md).

---

## <img src="assets/icons/agent.svg" width="22" height="22" align="absmiddle" alt="" /> Agent lifecycle

### Startup

1. `cmd/trustedge-agent` loads config, builds the API client, credentials store, and metrics.  
2. `EnsureRegistered()` loads device ID + token from disk/keyring, or calls `POST /v1/register`.  
3. `Agent.Run()` starts collectors, the batch flush loop, and optional periodic status logs.

### Telemetry path

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '14px', 'fontFamily': 'arial'}}}%%
flowchart LR
    C["Collect"] --> E["Enqueue"] --> R["Durable ring"]
    R --> F["Flush"] --> Z["Maybe zstd"] --> P["POST /v1/events"]
    P -->|202| A["Ack from ring"]
    P -->|fail| B["Backoff · keep queued"]

    classDef a fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef b fill:#FFEDD5,stroke:#EA580C,stroke-width:2px,color:#7C2D12
    classDef c fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95
    classDef d fill:#FEE2E2,stroke:#DC2626,stroke-width:2px,color:#7F1D1D

    class C,E,R a
    class F,Z b
    class P,A c
    class B d
```

1. **Collect** — sources emit typed payloads.  
2. **Enqueue** — `batcher.Enqueue(type, payload)`.  
3. **Buffer** — append `models.Event` records to the durable ring.  
4. **Flush** — pending ≥ `EventBatchSize` (32), `EventBatchFlush` (2s), or shutdown.  
5. **Compress** — marshal JSON → `codec.MaybeCompress()`.  
6. **Post** — `client.PostEvents()`; **Ack** only after success.  
7. **Ingest** — API decompresses if needed, validates, stores / publishes.  
8. **Response** — `202 Accepted` with `{ "status": "accepted", "accepted": N }`.

Failed uploads stay queued and retry with exponential backoff (capped by `TRUSTEDGE_AGENT_EVENT_RETRY_MAX`). When the ring is full, the **oldest** pending events are overwritten. Shutdown does a short best-effort flush (default **3s**); leftovers remain on disk for the next start.

| Operability | Knob |
|-------------|------|
| Log format | `TRUSTEDGE_AGENT_LOG_FORMAT=text\|json` |
| Status metrics | `TRUSTEDGE_AGENT_METRICS_INTERVAL` — upload counters, pending depth, `auth_recover_total` |

---

## <img src="assets/icons/collection.svg" width="22" height="22" align="absmiddle" alt="" /> Collectors

Five collectors run inside `Agent.Run()`:

| Collector | Event type | Trigger |
|-----------|------------|---------|
| Client details | `client_details` | Once at startup, then every `DetailsInterval` (60s) |
| Network monitor | `network_summary` | Link/IP change (debounced) + `NetworkInterval` heartbeat |
| Action tracker | `action_summary` | Sample every 5s; emit every 60s |
| Process monitor | `process_start` / `process_exit` | OS watcher + poll every 10s |
| Security monitor | `driver_load` / `service_install` / `registry_persistence` | Watcher wake + poll every 30s |

### Process monitoring (hybrid)

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart TB
    RT["Real-time OS notifications"]
    POLL["Periodic scan"]
    DEDUP["Deduplication"]
    OUT["Durable ring → batch"]

    RT --> DEDUP --> OUT
    POLL --> DEDUP

    classDef rt fill:#F1F5F9,stroke:#64748B,stroke-width:2px,color:#0F172A
    classDef dedup fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef out fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95

    class RT,POLL rt
    class DEDUP dedup
    class OUT out
```

| Path | Role |
|------|------|
| **Real-time** | OS notifies on start/exit when the platform watcher is available |
| **Periodic scan** | Reconciles the process table and catches misses |
| **Dedup** | Same PID transition is not emitted twice |

Disable processes: `TRUSTEDGE_AGENT_PROCESS_INTERVAL=0`.  
Platform matrix: [Platform watchers](watchers-overview.md) · CGO notes: [Agent guide](agent.md#platforms).

---

## <img src="assets/icons/queue.svg" width="22" height="22" align="absmiddle" alt="" /> Batching

`EventBatcher` (`internal/agent/batcher.go`) writes through a durable ring (`internal/agent/ring.go`) before upload.

| Flush trigger | Default | Config |
|---------------|---------|--------|
| Pending size | 32 | `TRUSTEDGE_AGENT_EVENT_BATCH_SIZE` |
| Time interval | 2s | `TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH` |
| Shutdown | ~3s best-effort | — |

| Queue knob | Default |
|------------|---------|
| Capacity | 4096 (`TRUSTEDGE_AGENT_EVENT_QUEUE_CAPACITY`) |
| Path | `events.queue.json` beside the state file |
| Max backoff | 60s (`TRUSTEDGE_AGENT_EVENT_RETRY_MAX`) |

Wire shapes: one event → plain `Event` JSON · several → `{"events":[...]}`.

---

## <img src="assets/icons/compress.svg" width="22" height="22" align="absmiddle" alt="" /> Compression

`internal/codec` applies **zstd** only when it wins:

| Outcome | Behavior |
|---------|----------|
| Smaller than raw JSON | Send compressed + `Content-Encoding: zstd` |
| Otherwise | Send plain JSON |
| API | Accepts both (backward compatible) |

---

## <img src="assets/icons/lock.svg" width="22" height="22" align="absmiddle" alt="" /> Authentication

```mermaid
sequenceDiagram
    participant A as Agent
    participant API as Agent API

    A->>API: POST /v1/register (optional enroll bearer)
    API-->>A: device_id + device_token
    A->>API: POST /v1/events (device bearer)
    API-->>A: 202 Accepted
    Note over A,API: On 401: clear token, re-register once (serialized), retry batch
```

| Step | Behavior |
|------|----------|
| **Register** | `POST /v1/register` (optional enroll bearer) |
| **Telemetry** | `POST /v1/events` with device bearer |
| **Recovery** | On `401`, clear token, re-register (mutex), retry the batch once |

Device ID lives on disk; the token lives in the OS keyring. Details: [Agent guide](agent.md#credentials-and-state).

---

## <img src="assets/icons/upload.svg" width="22" height="22" align="absmiddle" alt="" /> API persistence

Ingest storage and Kafka publishing are documented in [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API).

---

## <img src="assets/icons/layout.svg" width="22" height="22" align="absmiddle" alt="" /> Project layout

```text
cmd/trustedge-agent/     Entrypoint — config, slog, signal lifecycle
internal/agent/          Runtime, durable ring, batcher, auth, metrics
internal/collect/        OS collectors + platform watchers
internal/api/            HTTPS client (register + events)
internal/credentials/    Device ID file + keyring tokens
internal/codec/          Optional zstd
internal/config/         Env-based configuration
internal/models/         Event envelopes and payloads
docs/                    Architecture · watchers · collection · agent · config
```

---

## Interview talking points

1. **Thin edge, thick pipeline** — agent stays small; detection lives in TrustEdge.  
2. **Durability before cleverness** — ring + Ack-after-success beats “fire and forget”.  
3. **Auth that survives concurrency** — one serialized re-register for many `401`s.  
4. **Hybrid collection** — see [Platform watchers](watchers-overview.md) for the OS story.
