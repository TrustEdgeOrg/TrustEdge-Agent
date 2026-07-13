# TrustEdge Agent

**TrustEdge Agent** is the EDR-lite cross-platform endpoint agent (Go) plus an ingest API for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform. The agent reports device posture to your server — **no VPN required**.

Supported platforms: **macOS**, **Linux**, **Windows**.

| Event type | What it is |
|---|---|
| `client_details` | Device identity + online heartbeat |
| `network_summary` | Coarse network posture (counts, top ports) |
| `action_summary` | Short-window app focus + idle/active |
| `process_start` / `process_exit` | EDR-lite process visibility (pid, parent, user, comm) |

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg). Pairs with [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) (dashboard, detection engine, policy) and [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) (optional VPN enroll).

Events flow to Redis and Kafka (`trustedge.agent.events`) for live observability and rules-based detection (process chains, network drift).

## Documentation

| Guide | Description |
|-------|-------------|
| [docs/](docs/README.md) | Documentation index |
| [Architecture](docs/architecture.md) | Telemetry flow, batching, compression |
| [Agent](docs/agent.md) | Installation, platforms, collectors |
| [Configuration](docs/configuration.md) | Environment variables |
| [API reference](docs/api.md) | HTTP endpoints and event payloads |
| [AWS deploy](aws/README.md) | ECR build and EC2 deploy |

## Privacy

TrustEdge Agent does **not** collect window titles, URLs, keystrokes, screenshots, raw Wi‑Fi SSIDs, or full remote IP connection lists.

Process monitoring collects **metadata only** (pid, parent pid, user, process name) — not command lines or file contents. Disable with `TRUSTEDGE_AGENT_PROCESS_INTERVAL=0`.

## Components

| Binary | Role | Where it runs |
|--------|------|----------------|
| `trustedge-agent` | Endpoint agent | Each laptop / workstation |
| `trustedge-agent-api` | Ingest + auth + Redis/Kafka | EC2 Docker (ECR image) |

Production API uses **Redis + Kafka** only (no `devices.json` / `events.jsonl` on disk). Local `./data/` is optional dev fallback when running the API without production mode.

## Quick start — agent only (typical)

Point the agent at your EC2 ingest API:

```bash
cd ~/Desktop/TrustEdge-Agent
export TRUSTEDGE_AGENT_API_URL=http://YOUR_EC2_HOST:8080
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<from /etc/trustedge/agent-enroll.token on EC2>
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
make build       # → bin/trustedge-agent, bin/trustedge-agent-api
make build-all   # cross-platform agent binaries
make test
```

## Configuration (agent)

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

## Deploy trustedge-agent-api (ECR → EC2)

CI (`.github/workflows/deploy-api.yml`) builds and pushes `trustedge-agent-api` to ECR. TrustEdge `docker-compose.yml` pulls that image on EC2.

One-time setup: [aws/README.md](aws/README.md) (`AWS_ROLE_ARN`, OIDC trust policy).

## Local dev with TrustEdge stack

```text
~/Desktop/TrustEdge
~/Desktop/TrustEdge-Agent
```

```bash
cd TrustEdge && ./scripts/dev-up.sh
cd ../TrustEdge-Agent && TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080 go run ./cmd/trustedge-agent
```

Compose builds the API from `../TrustEdge-Agent` and uses a Docker volume for API state (not repo `data/`).

## Project layout

```text
cmd/trustedge-agent/          # agent
cmd/trustedge-agent-api/      # ingest API
internal/collect/       # platform telemetry collectors
internal/agent/         # agent runtime + batcher
internal/codec/         # zstd compression
internal/store/         # API persistence (memory, disk, Redis, Kafka)
docs/                   # documentation
```
