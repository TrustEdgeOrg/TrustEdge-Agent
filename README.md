# TrustEdge Agent

**TrustEdge Agent** is the EDR-lite cross-platform endpoint agent (Go) for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform. The agent reports device posture to [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) — **no VPN required**.

Supported platforms: **macOS**, **Linux**, **Windows**.

| Event type | What it is |
|---|---|
| `client_details` | Device identity + online heartbeat |
| `network_summary` | Coarse network posture (counts, top ports) |
| `action_summary` | Short-window app focus + idle/active |
| `process_start` / `process_exit` | EDR-lite process visibility (pid, parent, user, comm) |

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg). Pairs with [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) (ingest), [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) (dashboard, detection), and [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) (optional VPN enroll).

## Documentation

| Guide | Description |
|-------|-------------|
| [docs/](docs/README.md) | Documentation index |
| [Architecture](docs/architecture.md) | Telemetry flow, batching, compression |
| [Agent](docs/agent.md) | Installation, platforms, collectors |
| [Configuration](docs/configuration.md) | Environment variables |
| [API reference](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API/blob/main/docs/api.md) | HTTP endpoints (TrustEdge-Agent-API repo) |

## Privacy

TrustEdge Agent does **not** collect window titles, URLs, keystrokes, screenshots, raw Wi‑Fi SSIDs, or full remote IP connection lists.

Process monitoring collects **metadata only** (pid, parent pid, user, process name) — not command lines or file contents. Disable with `TRUSTEDGE_AGENT_PROCESS_INTERVAL=0`.

## Quick start

Point the agent at your ingest API:

```bash
cd ~/Desktop/TrustEdge-Agent
export TRUSTEDGE_AGENT_API_URL=http://YOUR_API_HOST:8080
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<from server>
go run ./cmd/trustedge-agent
```

Or build and run:

```bash
make build
./bin/trustedge-agent
```

**Credentials:**

| OS | Device ID | Device token |
|----|-----------|--------------|
| macOS | `~/Library/Application Support/TrustEdge Agent/state.json` | Keychain |
| Linux | `~/.local/share/TrustEdge Agent/state.json` | Secret Service |
| Windows | `%APPDATA%\TrustEdge Agent\state.json` | Credential Manager |

## Build

```bash
make build       # → bin/trustedge-agent
make build-all   # cross-platform agent binaries
make test
```

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `TRUSTEDGE_AGENT_API_URL` | `http://127.0.0.1:8080` | Ingest API URL |
| `TRUSTEDGE_AGENT_ENROLL_TOKEN` | _(empty)_ | Required for EC2 when API enforces enroll |
| `TRUSTEDGE_AGENT_STATE_PATH` | Platform default | Agent device ID file |
| `TRUSTEDGE_AGENT_DETAILS_INTERVAL` | `60` | `client_details` interval (seconds) |
| `TRUSTEDGE_AGENT_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat |
| `TRUSTEDGE_AGENT_ACTION_INTERVAL` | `60` | `action_summary` interval |
| `TRUSTEDGE_AGENT_PROCESS_INTERVAL` | `10` | Process polling; `0` disables |
| `TRUSTEDGE_AGENT_EVENT_BATCH_SIZE` | `32` | Events per upload batch |
| `TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH` | `2` | Max seconds between batch flushes |
| `TRUSTEDGE_AGENT_PRODUCTION` | `0` | `1` requires HTTPS + enroll token on agent |

Full reference: [docs/configuration.md](docs/configuration.md).

## Authentication

1. **Register** — `POST /v1/register` (+ optional enroll bearer token)
2. **Telemetry** — `POST /v1/events` with device token (batched, optionally zstd-compressed)
3. **401 recovery** — agent re-registers once if token rejected

See [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) for server-side ingest, Redis, and Kafka.

## Local dev with TrustEdge stack

```text
~/Desktop/TrustEdge
~/Desktop/TrustEdge-Agent
~/Desktop/TrustEdge-Agent-API
```

```bash
cd TrustEdge-Agent-API && go run ./cmd/trustedge-agent-api
cd ../TrustEdge-Agent && TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent
```

## Project layout

```text
cmd/trustedge-agent/     # agent entrypoint
internal/collect/        # platform telemetry collectors
internal/agent/          # agent runtime + batcher
internal/api/            # HTTP client to ingest API
internal/codec/          # zstd compression
docs/                    # documentation
```
