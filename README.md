# TrustTwin

**TrustTwin** is the EDR-lite macOS endpoint agent (Go) plus an ingest API for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform. The agent reports device posture to your server — **no VPN required**.

| Event type | What it is |
|---|---|
| `client_details` | Device identity + online heartbeat |
| `network_summary` | Coarse network posture (counts, top ports) |
| `action_summary` | Short-window app focus + idle/active |
| `process_start` / `process_exit` | EDR-lite process visibility (pid, parent, user, comm) |

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg). Pairs with [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) (dashboard, detection engine, policy) and [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) (optional VPN enroll).

Events flow to Redis and Kafka (`trusttwin.events`) for live observability and rules-based detection (process chains, network drift).

## Privacy

TrustTwin does **not** collect window titles, URLs, keystrokes, screenshots, raw Wi‑Fi SSIDs, or full remote IP connection lists.

Process monitoring collects **metadata only** (pid, parent pid, user, process name) — not command lines or file contents. Disable with `TRUSTTWIN_PROCESS_INTERVAL=0`.

## Components

| Binary | Role | Where it runs |
|--------|------|----------------|
| `trusttwin` | Endpoint agent | Each laptop |
| `trusttwin-api` | Ingest + auth + Redis/Kafka | EC2 Docker (ECR image) |

Production API uses **Redis + Kafka** only (no `devices.json` / `events.jsonl` on disk). Local `./data/` is optional dev fallback when running the API without production mode.

## Quick start — agent only (typical)

Point the agent at your EC2 ingest API:

```bash
cd ~/Desktop/TrustTwin
export TRUSTTWIN_API_URL=http://YOUR_EC2_HOST:8080
export TRUSTTWIN_ENROLL_TOKEN=<from /etc/trustedge/trusttwin-enroll.token on EC2>
go run ./cmd/trusttwin
```

Or build and run:

```bash
make build
./bin/trusttwin
```

Device ID: `~/Library/Application Support/TrustTwin/state.json`  
Device token: macOS **Keychain** (service `TrustTwin`).

## Build

```bash
make build    # → bin/trusttwin, bin/trusttwin-api
make test
```

## Configuration (agent)

| Variable | Default | Purpose |
|---|---|---|
| `TRUSTTWIN_API_URL` | `http://127.0.0.1:8080` | Ingest API URL |
| `TRUSTTWIN_ENROLL_TOKEN` | _(empty)_ | Required for EC2 when API enforces enroll |
| `TRUSTTWIN_STATE_PATH` | `~/Library/.../TrustTwin/state.json` | Agent device id file |
| `TRUSTTWIN_DETAILS_INTERVAL` | `60` | `client_details` interval (seconds) |
| `TRUSTTWIN_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat |
| `TRUSTTWIN_ACTION_INTERVAL` | `60` | `action_summary` interval |
| `TRUSTTWIN_PROCESS_INTERVAL` | `10` | Process polling; `0` disables |
| `TRUSTTWIN_PRODUCTION` | `0` | `1` requires HTTPS + enroll token on agent |

## Authentication

1. **Register** — `POST /v1/register` (+ optional enroll bearer token)
2. **Telemetry** — `POST /v1/events` with device token
3. **401 recovery** — agent re-registers once if token rejected

## Deploy trusttwin-api (ECR → EC2)

CI (`.github/workflows/deploy-api.yml`) builds and pushes `trustedge-trusttwin-api` to ECR. TrustEdge `docker-compose.yml` pulls that image on EC2.

One-time setup: [aws/README.md](aws/README.md) (`AWS_ROLE_ARN`, OIDC trust policy).

## Local dev with TrustEdge stack

```text
~/Desktop/TrustEdge
~/Desktop/TrustTwin
```

```bash
cd TrustEdge && ./scripts/dev-up.sh
cd ../TrustTwin && TRUSTTWIN_API_URL=http://127.0.0.1:8080 go run ./cmd/trusttwin
```

Compose builds the API from `../TrustTwin` and uses a Docker volume for API state (not repo `data/`).

## API reference

See [docs/api.md](docs/api.md).

## Project layout

```text
cmd/trusttwin/          # agent
cmd/trusttwin-api/      # ingest API
internal/collect/       # telemetry collectors
internal/agent/         # agent runtime
internal/store/         # API persistence (memory, optional disk, Redis, Kafka)
docs/api.md
```
