# <img src="assets/icons/agent.svg" width="28" height="28" align="absmiddle" alt="" /> Agent guide

The `trustedge-agent` binary runs on each endpoint and reports device posture to [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) over HTTPS. **No VPN required.**

> **In one breath:** Build once, run on macOS / Linux / Windows, register for a device token in the OS keyring, and keep uploading through offline gaps via a durable ring.

<p align="center">
  <img src="assets/icons/platforms.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#platforms">Platforms</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/install.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#installation">Install</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/lock.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#credentials-and-state">Credentials</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#telemetry">Telemetry</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/privacy.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#privacy">Privacy</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#see-also">Related</a>
</p>

> **Also see**
> <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" /> [Collection](collection.md)
> · <img src="assets/icons/config.svg" width="16" height="16" align="absmiddle" alt="" /> [Configuration](configuration.md)
> · <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" /> [Docs hub](README.md)

---

## <img src="assets/icons/platforms.svg" width="22" height="22" align="absmiddle" alt="" /> Platforms

| OS | Default local build | Richer process watcher | Credential store |
|----|---------------------|------------------------|------------------|
| **macOS** (arm64 / amd64) | `CGO=0` poll mode | Endpoint Security (`make build-cgo` + entitlement) | Keychain |
| **Linux** (amd64) | `CGO=0` poll mode | Netlink PROC connector | Secret Service |
| **Windows** (amd64) | Poll / ETW depending on build | ETW kernel process (admin) | Credential Manager |

CI builds and tests all three platforms (`.github/workflows/agent-ci.yml`). Default `make build` / `make test` use **`CGO_ENABLED=0`** for portable poll-mode binaries.

### Watcher privileges

| Platform | Needed for the event-driven watcher |
|----------|-------------------------------------|
| Linux | Root or `CAP_NET_ADMIN` |
| Windows | Administrator |
| macOS | Endpoint Security entitlement, signed binary, user approval |

If the watcher cannot start → **poll-only** — no hard failure.  
Hybrid design + OS matrix: [Collection](collection.md#how-detection-works-per-os).

---

## <img src="assets/icons/install.svg" width="22" height="22" align="absmiddle" alt="" /> Installation

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

| Mode | Requirement |
|------|-------------|
| Local / demo | `TRUSTEDGE_AGENT_API_URL` required (no baked-in default host) |
| Production | `TRUSTEDGE_AGENT_PRODUCTION=1` → **HTTPS** + enroll token |

---

## <img src="assets/icons/lock.svg" width="22" height="22" align="absmiddle" alt="" /> Credentials and state

On first successful register, the agent stores:

| Item | Where |
|------|--------|
| **Device ID** | State JSON (platform path below) |
| **Device token** | OS keyring (Keychain / Secret Service / Credential Manager) |
| **Pending events** | Durable ring beside state (`events.queue.json`) |

### Default state paths

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/TrustEdge Agent/state.json` |
| Linux | `~/.local/share/TrustEdge Agent/state.json` |
| Windows | `%APPDATA%\TrustEdge Agent\state.json` |

Override: `TRUSTEDGE_AGENT_STATE_PATH` · queue: `TRUSTEDGE_AGENT_EVENT_QUEUE_PATH`.  
With `TRUSTEDGE_AGENT_PRODUCTION=1`, tokens stay in the **keyring only** — not written into `state.json`.

---

## <img src="assets/icons/collection.svg" width="22" height="22" align="absmiddle" alt="" /> Telemetry

| Event type | What it reports |
|------------|-----------------|
| `client_details` | Device identity, OS, arch, agent version, uptime |
| `network_summary` | Public IP, interface type, socket counts, top remote ports |
| `action_summary` | Foreground focus, idle vs active, app switches |
| `process_start` | New process: pid, ppid, user, name, path, cmdline |
| `process_exit` | Exit lifecycle (enriched from start when available) |
| `driver_load` | Newly observed loaded driver/kext |
| `service_install` | Newly observed Windows service or macOS LaunchDaemon |
| `registry_persistence` | New/changed Windows Run key or macOS LaunchAgent |
| `known_ai_app` | AI tools inventory upsert/removal (apps, CLI agents, local model runtimes, IDE extensions) |

| Collector | Behavior |
|-----------|----------|
| **Client details** | Once at startup, then on `DETAILS_INTERVAL` |
| **Network** | On interface/address change (debounced) + heartbeat |
| **Actions** | Sample foreground ~5s; emit one summary per ~60s window |
| **Processes** | Watcher (when available) + poll reconcile + dedup |
| **Security** | Watcher wake + poll reconcile with silent baseline |
| **AI inventory** | Timed poll + optional wake from process RuntimeFeed |

Payload schemas: [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md).  
Timers / flush: [Collection & batching](collection.md).  
Public IP: configurable URL; disable with `TRUSTEDGE_AGENT_PUBLIC_IP_URL=off`.

---

## <img src="assets/icons/privacy.svg" width="22" height="22" align="absmiddle" alt="" /> Privacy

The agent does **not** collect:

- Window titles or URLs  
- Keystrokes or clipboard  
- Screenshots  
- Raw Wi‑Fi SSIDs  
- Full remote IP connection tables  
- File contents  

Process monitoring includes metadata **and command line** (truncated at 4 KiB):

```bash
export TRUSTEDGE_AGENT_PROCESS_INTERVAL=0      # disable processes
export TRUSTEDGE_AGENT_SECURITY_INTERVAL=0     # disable security lifecycle
export TRUSTEDGE_AGENT_KNOWN_AI_INTERVAL=0     # disable AI inventory
```

---

## <img src="assets/icons/lock.svg" width="22" height="22" align="absmiddle" alt="" /> Auth recovery

Uploads carry the device token. On **`401 Unauthorized`**:

1. Serialize recovery so concurrent 401s only re-register once  
2. Clear the stored device token  
3. `POST /v1/register` (skipped if another goroutine already refreshed)  
4. Retry the failed batch once  

Failed uploads otherwise stay in the durable ring and retry with exponential backoff.

---

## <img src="assets/icons/flow.svg" width="22" height="22" align="absmiddle" alt="" /> Local stack with TrustEdge

```bash
# Terminal 1 — TrustEdge compose / scripts (Redis, stream, API)
cd TrustEdge && ./scripts/dev-up.sh

# Terminal 2 — this agent
cd TrustEdge-Agent
export TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080
go run ./cmd/trustedge-agent
```

Or point at any local [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) instance on `:8080`.

---

## <img src="assets/icons/layout.svg" width="22" height="22" align="absmiddle" alt="" /> See also

| | Doc | Purpose |
|---|-----|---------|
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Architecture](architecture.md) | Lifecycle, upload path, durable ring |
| <img src="assets/icons/collection.svg" width="18" height="18" align="absmiddle" alt="" /> | [Collection & batching](collection.md) | Collectors, flush triggers, concurrency |
| <img src="assets/icons/config.svg" width="18" height="18" align="absmiddle" alt="" /> | [Configuration](configuration.md) | Every environment variable |
| <img src="assets/icons/upload.svg" width="18" height="18" align="absmiddle" alt="" /> | [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) | HTTP schemas |
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Docs hub](README.md) | Index of agent docs |
