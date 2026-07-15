# Agent guide

The `trustedge-agent` binary runs on each endpoint (laptop, workstation, or server) and reports device posture to [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) over HTTPS. **No VPN required.**

| You are here for… | Jump to |
|-------------------|---------|
| Install & run | [Installation](#installation) |
| Where credentials live | [Credentials and state](#credentials-and-state) |
| What leaves the device | [Telemetry](#telemetry) · [Privacy](#privacy) |
| OS / watcher notes | [Platforms](#platforms) |
| Auth after `401` | [Auth recovery](#auth-recovery) |

Related: [Architecture](architecture.md) · [Collection & batching](collection.md) · [Configuration](configuration.md)

---

## Platforms

| OS | Default local build | Richer process watcher | Credential store |
|----|---------------------|------------------------|------------------|
| **macOS** (arm64 / amd64) | `CGO=0` poll mode | Endpoint Security (needs `make build-cgo` + entitlement) | Keychain |
| **Linux** (amd64) | `CGO=0` poll mode | Netlink PROC connector | Secret Service |
| **Windows** (amd64) | Poll / ETW depending on build | ETW kernel process (admin) | Credential Manager |

CI builds and tests all three platforms (`.github/workflows/agent-ci.yml`). Default `make build` / `make test` use **`CGO_ENABLED=0`** for portable poll-mode binaries.

### Watcher privileges

| Platform | Needed for the event-driven watcher |
|----------|-------------------------------------|
| Linux | Root or `CAP_NET_ADMIN` |
| Windows | Administrator |
| macOS | Endpoint Security entitlement, signed binary, user approval |

If the watcher cannot start, the agent **falls back to poll-only** process monitoring — no hard failure.

---

## Installation

### Build

```bash
cd TrustEdge-Agent
make build          # → bin/trustedge-agent  (CGO=0)
make build-cgo      # richer macOS / Windows watchers where available
make build-all      # cross-compile → bin/trustedge-agent-{darwin,linux,windows}-*
```

Requires **Go 1.22+**.

### Run (local API)

```bash
export TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080
# export TRUSTEDGE_AGENT_ENROLL_TOKEN=...   # if your API requires enroll

make build
./bin/trustedge-agent
```

One-liner: `make run-agent-local`.

### Run (remote ingest)

```bash
export TRUSTEDGE_AGENT_API_URL=https://your-ingest.example
export TRUSTEDGE_AGENT_ENROLL_TOKEN=your-enroll-token
./bin/trustedge-agent
```

Production checklist: set `TRUSTEDGE_AGENT_PRODUCTION=1` so the agent requires **HTTPS** and an enroll token.

> `TRUSTEDGE_AGENT_API_URL` is **required** — there is no baked-in default host.

---

## Credentials and state

On first successful register, the agent stores:

| Item | Where |
|------|--------|
| **Device ID** | State JSON (platform path below) |
| **Device token** | OS keyring (Keychain / Secret Service / Credential Manager) |
| **Pending events** | Durable ring file next to state (`events.queue.json` by default) |

### Default state paths

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/TrustEdge Agent/state.json` |
| Linux | `~/.local/share/TrustEdge Agent/state.json` |
| Windows | `%APPDATA%\TrustEdge Agent\state.json` |

Override with `TRUSTEDGE_AGENT_STATE_PATH`. Queue path override: `TRUSTEDGE_AGENT_EVENT_QUEUE_PATH`.

With `TRUSTEDGE_AGENT_PRODUCTION=1`, tokens stay in the **keyring only** — not written into `state.json`.

---

## Telemetry

| Event type | What it reports |
|------------|-----------------|
| `client_details` | Device identity, OS, arch, agent version, uptime (heartbeat) |
| `network_summary` | Public IP, interface type, socket counts, top remote ports |
| `action_summary` | Foreground focus, idle vs active, app switches |
| `process_start` | New process: pid, ppid, user, name, path, cmdline |
| `process_exit` | Exit lifecycle (enriched from start when available) |

Payload schemas: [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md).  
Timers, flush rules, concurrency: [Collection & batching](collection.md).

### How collectors behave

| Collector | Behavior |
|-----------|----------|
| **Client details** | Once at startup, then on `TRUSTEDGE_AGENT_DETAILS_INTERVAL` |
| **Network** | On interface/address change (debounced) + periodic heartbeat |
| **Actions** | Sample foreground every ~5s; emit one summary per ~60s window |
| **Processes** | Watcher (when available) + poll reconcile + dedup |

Public IP comes from a configurable lookup URL (default: ipify). Disable with `TRUSTEDGE_AGENT_PUBLIC_IP_URL=off`.

---

## Privacy

The agent does **not** collect:

- Window titles or URLs  
- Keystrokes or clipboard  
- Screenshots  
- Raw Wi‑Fi SSIDs  
- Full remote IP connection tables  
- File contents  

Process monitoring includes metadata **and command line** (truncated at 4 KiB). Turn processes off with:

```bash
export TRUSTEDGE_AGENT_PROCESS_INTERVAL=0
```

---

## Auth recovery

Uploads carry the device token. On **`401 Unauthorized`**:

1. Serialize recovery so concurrent 401s only re-register once  
2. Clear the stored device token  
3. `POST /v1/register` (skipped if another goroutine already refreshed)  
4. Retry the failed batch once  

Failed uploads otherwise stay in the durable ring and retry with exponential backoff.

---

## Local stack with TrustEdge

```bash
# Terminal 1 — TrustEdge compose / scripts (Redis, stream, API)
cd ~/Desktop/TrustEdge && ./scripts/dev-up.sh

# Terminal 2 — this agent
cd ~/Desktop/TrustEdge-Agent
export TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080
go run ./cmd/trustedge-agent
```

Or point at any local [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) instance on `:8080`.

---

## See also

| Doc | Purpose |
|-----|---------|
| [Architecture](architecture.md) | Lifecycle, upload path, durable ring |
| [Collection & batching](collection.md) | Collectors, flush triggers, concurrency |
| [Configuration](configuration.md) | Every environment variable |
| [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) | HTTP schemas |
