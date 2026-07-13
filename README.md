# TrustEdge Agent

A lightweight cross-platform endpoint agent for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security platform.

TrustEdge Agent runs on laptops, workstations, and servers. It collects device posture — OS info, network activity, user presence, and process lifecycle — and sends it securely to TrustEdge for detection and alerting. **No VPN required.**

**Platforms:** macOS · Linux · Windows

## How it fits in TrustEdge

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart LR
    A["TrustEdge Agent"] -->|HTTPS telemetry| B["Ingest API"]
    B -->|Event stream| C["TrustEdge Platform"]

    classDef agent fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef api fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95
    classDef platform fill:#DCFCE7,stroke:#16A34A,stroke-width:2px,color:#14532D

    class A agent
    class B api
    class C platform
```

| Repo | Role |
|------|------|
| **[TrustEdge Agent](https://github.com/TrustEdgeOrg/TrustEdge-Agent)** (this repo) | Collects endpoint telemetry and uploads it |
| **[TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API)** | Ingest API — validates events, publishes to Kafka |
| **[TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge)** | Dashboard, rules engine, alerts |
| **[TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient)** | Optional VPN enroll client |

## How it works

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart LR
    subgraph EP ["Endpoint"]
        DEV(["Device"])
    end

    subgraph AG ["TrustEdge Agent"]
        direction LR
        COL["Collect"] --> BAT["Batch"] --> ZIP["Compress"] --> SND["Send"]
    end

    subgraph CL ["TrustEdge Cloud"]
        direction LR
        API["Ingest API"] --> KFK[("Kafka")] --> DET["Detection"]
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
    class API,DET cloud
    class KFK stream
```

| Stage | Description |
|-------|-------------|
| **Collect** | Gather device, network, activity, and process telemetry from the OS |
| **Batch** | Buffer events in memory (up to 32 events or every 2 seconds) |
| **Compress** | JSON batches are optionally compressed with zstd when smaller than raw JSON |
| **Send** | Upload the batch to the ingest API over HTTPS |
| **Ingest** | API decompresses if needed, validates, and accepts events |
| **Kafka** | Events are published to the event stream |
| **Detection** | TrustEdge applies rules and raises alerts |

## What the agent watches

```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'fontSize': '15px', 'fontFamily': 'arial'}}}%%
flowchart TB
    subgraph SRC ["Data Sources"]
        direction LR
        D1["Device Info"]
        D2["Network"]
        D3["User Activity"]
        D4["Processes"]
    end

    BAT["Event Batcher"]
    ZIP["zstd Compress"]
    API["Ingest API"]

    D1 --> BAT
    D2 --> BAT
    D3 --> BAT
    D4 --> BAT
    BAT --> ZIP
    ZIP -->|HTTPS upload| API

    classDef source fill:#F1F5F9,stroke:#64748B,stroke-width:2px,color:#0F172A
    classDef batch fill:#DBEAFE,stroke:#2563EB,stroke-width:2px,color:#1E3A8A
    classDef compress fill:#FFEDD5,stroke:#EA580C,stroke-width:2px,color:#7C2D12
    classDef api fill:#EDE9FE,stroke:#7C3AED,stroke-width:2px,color:#4C1D95

    class D1,D2,D3,D4 source
    class BAT batch
    class ZIP compress
    class API api
```

| Area | What is collected |
|------|-------------------|
| **Device** | Hostname, OS, architecture, agent version, uptime |
| **Network** | Public IP, connection counts, top remote ports |
| **User activity** | Foreground app focus, idle/active presence, app switches |
| **Processes** | Process starts and exits — pid, parent, user, name, executable path |

Details: [Collection and batching](docs/collection.md) · [Architecture](docs/architecture.md)

## Privacy

TrustEdge Agent is designed for **metadata only**. It does **not** collect:

- Window titles or URLs
- Keystrokes or clipboard contents
- Screenshots
- Raw Wi‑Fi SSIDs
- Full remote IP connection tables
- Command lines or file contents

Process monitoring can be disabled entirely: `TRUSTEDGE_AGENT_PROCESS_INTERVAL=0`

## Quick start

```bash
git clone https://github.com/TrustEdgeOrg/TrustEdge-Agent.git
cd TrustEdge-Agent

export TRUSTEDGE_AGENT_API_URL=https://your-api-host
export TRUSTEDGE_AGENT_ENROLL_TOKEN=your-enroll-token

make build
./bin/trustedge-agent
```

For local development without building:

```bash
TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent
```

On first run the agent registers with the API and stores credentials locally (device ID on disk, token in the OS keyring). See [Agent guide](docs/agent.md) for platform paths and permissions.

## Documentation

| Guide | Description |
|-------|-------------|
| [docs/](docs/README.md) | Documentation index |
| [Architecture](docs/architecture.md) | End-to-end flow, compression, auth |
| [Collection and batching](docs/collection.md) | Collectors, flush triggers, upload |
| [Agent](docs/agent.md) | Installation, platforms, credentials |
| [Configuration](docs/configuration.md) | All environment variables |
| [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) | HTTP endpoints and payload schemas |

## For developers

### Build and test

```bash
make build       # → bin/trustedge-agent
make build-all   # cross-platform binaries
make test
```

### Local dev with TrustEdge stack

```bash
# Terminal 1 — ingest API
cd TrustEdge-Agent-API && go run ./cmd/trustedge-agent-api

# Terminal 2 — agent
cd TrustEdge-Agent && TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent
```

### Project layout

```text
cmd/trustedge-agent/     # agent entrypoint
internal/collect/        # platform telemetry collectors
internal/agent/          # agent runtime + batcher
internal/api/            # HTTP client to ingest API
internal/codec/          # zstd compression
docs/                    # documentation
```

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg).
