# Agent

The `trusttwin` binary runs on each endpoint (laptop, workstation, server) and reports device posture to the ingest API. No VPN is required.

## Supported platforms

| OS | Build | Process watcher | Credentials |
|----|-------|-----------------|-------------|
| **macOS** (arm64, amd64) | CGO enabled (default) | Endpoint Security (CGO) | Keychain |
| **macOS** (CGO=0) | Poll-only process monitoring | — | Keychain |
| **Linux** (amd64) | CGO optional | Netlink PROC connector | Secret Service keyring |
| **Windows** (amd64) | CGO enabled | ETW kernel process | Windows Credential Manager |

CI builds and tests all three platforms (`.github/workflows/agent-ci.yml`).

### Platform requirements

| Platform | Process watcher needs |
|----------|----------------------|
| Linux | Root or `CAP_NET_ADMIN` for netlink connector |
| Windows | Administrator for ETW |
| macOS | Apple Endpoint Security entitlement, signed binary, user approval |

When the watcher is unavailable, the agent falls back to poll-only process monitoring.

## Installation

### From source

```bash
cd TrustTwin
make build          # → bin/trusttwin
```

Cross-platform binaries:

```bash
make build-all      # → bin/trusttwin-{darwin,linux,windows}-*
```

### Run

```bash
export TRUSTTWIN_API_URL=http://YOUR_API_HOST:8080
export TRUSTTWIN_ENROLL_TOKEN=<enroll-token>   # required in production
./bin/trusttwin
```

Or during development:

```bash
TRUSTTWIN_API_URL=http://127.0.0.1:8080 go run ./cmd/trusttwin
```

## Credentials and state

On first run the agent registers with the API and stores credentials locally:

| Item | Storage |
|------|---------|
| Device ID | State JSON file (platform default path) |
| Device token | OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager) |

### Default state paths

| OS | Path |
|----|------|
| macOS | `~/Library/Application Support/TrustTwin/state.json` |
| Linux | `~/.local/share/TrustTwin/state.json` |
| Windows | `%APPDATA%\TrustTwin\state.json` |

Override with `TRUSTTWIN_STATE_PATH`.

In production mode (`TRUSTTWIN_PRODUCTION=1`), tokens are stored in the keyring only — not in `state.json`.

## Telemetry types

| Event type | What it reports |
|------------|-----------------|
| `client_details` | Device identity, OS, arch, agent version, uptime — online heartbeat |
| `network_summary` | Coarse network posture: public IP, interface type, socket counts, top remote ports |
| `action_summary` | Short-window app focus, idle/active presence, app switch count |
| `process_start` | New process: pid, ppid, user, comm, executable path |
| `process_exit` | Process exit: pid, comm |

See [API reference](api.md) for payload field details.

## Privacy

TrustTwin does **not** collect:

- Window titles or URLs
- Keystrokes or clipboard
- Screenshots
- Raw Wi‑Fi SSIDs
- Full remote IP connection tables
- Command lines or file contents

Process monitoring collects **metadata only** (pid, parent pid, user, process name, executable path).

Disable process monitoring with `TRUSTTWIN_PROCESS_INTERVAL=0`.

## Collectors in detail

### Client details

Sent immediately on startup, then on a fixed interval. Provides presence heartbeat for the TrustEdge twin graph.

### Network summary

`NetworkMonitor` watches OS network state. Emits when interfaces or addresses change (debounced by `TRUSTTWIN_NETWORK_DEBOUNCE`, default 2s) and on a periodic heartbeat (`TRUSTTWIN_NETWORK_INTERVAL`).

Public IP is fetched from a configurable URL (default: ipify). Set `TRUSTTWIN_PUBLIC_IP_URL=off` to disable outbound lookup.

### Action summary

`ActionTracker` samples foreground application and idle state over each interval window, then emits one summary event with focus durations and presence.

### Process monitor

Hybrid event-driven + poll model:

1. **Watcher** (when available) streams `process_start` / `process_exit` in real time.
2. **Poll** every `ProcessInterval` reconciles the process table and enqueues missed transitions.
3. **Dedup** via `Observe()` prevents duplicate events for the same PID transition.

## Batching and upload

Events are not sent individually. The `EventBatcher` buffers them and flushes when:

- Buffer reaches `TRUSTTWIN_EVENT_BATCH_SIZE` (default 32), or
- `TRUSTTWIN_EVENT_BATCH_FLUSH` elapses (default 2s), or
- Agent shuts down

Batches are JSON-marshaled and optionally zstd-compressed before `POST /v1/events`. See [Architecture](architecture.md).

## Auth recovery

If the API returns `401 Unauthorized` on a telemetry upload, the agent:

1. Clears the stored device token
2. Re-registers via `POST /v1/register`
3. Retries the failed batch once

## Local dev with TrustEdge

```bash
# Terminal 1 — TrustEdge stack (Redis, Redpanda, API)
cd ~/Desktop/TrustEdge && ./scripts/dev-up.sh

# Terminal 2 — agent
cd ~/Desktop/TrustTwin
TRUSTTWIN_API_URL=http://127.0.0.1:8080 go run ./cmd/trusttwin
```

TrustEdge `docker-compose.yml` can build the API from `../TrustTwin` for integrated local testing.
