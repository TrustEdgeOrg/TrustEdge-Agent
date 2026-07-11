# TrustTwin

**TrustTwin** is a macOS digital twin agent written in Go. It reports three live pillars to an ingest API (local or EC2) — **no VPN**.

| Pillar | What it is |
|---|---|
| `client_details` | Device identity + online heartbeat |
| `network_summary` | Coarse network posture (counts, top ports) |
| `action_summary` | Short-window app focus + idle/active |

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg). Companion to [TrustEdgeClient](https://github.com/TrustEdgeOrg/TrustEdgeClient) (VPN client) — TrustTwin is telemetry-only.

## Privacy

TrustTwin does **not** collect window titles, URLs, keystrokes, screenshots, raw Wi‑Fi SSIDs, or full remote IP connection lists.

## Quick start

### 1. Run the API

```bash
cd ~/Desktop/TrustTwin
go run ./cmd/trusttwin-api
```

Listens on `:8080` by default. Data is stored under `./data/`.

### 2. Run the agent

```bash
go run ./cmd/trusttwin
```

Collects real hostname/OS, public IP, port summaries (`netstat`), foreground app (`lsappinfo` / AppleScript), and idle time (`ioreg`).

**network_summary** posts on agent start, **60s heartbeat**, and **link changes** (route socket on macOS when available). Socket posture is not polled between heartbeats.

### 3. Inspect a client

```bash
curl -s http://127.0.0.1:8080/v1/clients/<device_id> | python3 -m json.tool
```

The agent prints `device_id` on register. On macOS, `device_id` is saved at:

`~/Library/Application Support/TrustTwin/state.json`

The device token is stored in **macOS Keychain** (service `TrustTwin`), not in `state.json`. On first upgrade from older agents, any token in `state.json` is migrated to Keychain automatically.

On Linux/Windows, the token remains in `state.json` with a startup warning (Keychain support is macOS-first).

## Authentication

1. **Register** — `POST /v1/register` with optional `Authorization: Bearer <enroll_token>` when `TRUSTTWIN_ENROLL_TOKEN` is set.
2. **Telemetry** — `POST /v1/events` with `Authorization: Bearer <device_token>`.
3. **401 recovery** — if the API rejects the device token (e.g. server data wiped), the agent clears credentials, re-registers once, and retries the event.

### Production mode

Set `TRUSTTWIN_PRODUCTION=1` on both agent and API:

| Requirement | Agent | API |
|---|---|---|
| `TRUSTTWIN_ENROLL_TOKEN` | required | required |
| `TRUSTTWIN_API_URL` | must be `https://` | — |

## Configuration

| Variable | Default | Used by |
|---|---|---|
| `TRUSTTWIN_API_URL` | `http://127.0.0.1:8080` | agent |
| `TRUSTTWIN_ENROLL_TOKEN` | _(empty)_ | agent + API |
| `TRUSTTWIN_STATE_PATH` | `~/Library/Application Support/TrustTwin/state.json` | agent |
| `TRUSTTWIN_LISTEN` | `:8080` | API |
| `TRUSTTWIN_DATA_DIR` | `data` | API |
| `TRUSTTWIN_REDIS_URL` / `REDIS_URL` | _(empty)_ | API (live twin state for TrustEdge) |
| `TRUSTTWIN_DETAILS_INTERVAL` | `60` (seconds) | agent (`client_details`) |
| `TRUSTTWIN_NETWORK_INTERVAL` | `60` | agent (`network_summary` heartbeat) |
| `TRUSTTWIN_NETWORK_DEBOUNCE` | `2` | agent (coalesce rapid link changes) |
| `TRUSTTWIN_ACTION_INTERVAL` | `60` | agent (`action_summary`) |
| `TRUSTTWIN_PUBLIC_IP_URL` | ipify JSON endpoint | agent (`network_summary.public_ip`; `off` disables lookup) |
| `TRUSTTWIN_PRODUCTION` | `0` | agent + API (enforces HTTPS + enroll token) |

## Build

```bash
make build
./bin/trusttwin-api
./bin/trusttwin
```

# TrustTwin standalone repository (agent + trusttwin-api ingest).

Part of [TrustEdgeOrg](https://github.com/TrustEdgeOrg). Pairs with [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) — the dashboard reads live device state from Redis; it does not call this API directly.

## Components

| Binary | Role |
|--------|------|
| `trusttwin` | macOS endpoint agent (runs on each laptop) |
| `trusttwin-api` | Ingest + auth server (Docker on EC2) |

## Quick start (local)

```bash
# Terminal 1 — API
go run ./cmd/trusttwin-api

# Terminal 2 — agent
go run ./cmd/trusttwin
```

With TrustEdge dev stack (`make dev-up` in TrustEdge repo), point the agent at `http://127.0.0.1:8080`.

## Deploy trusttwin-api (ECR)

CI (`.github/workflows/deploy-api.yml`) builds and pushes `trustedge-trusttwin-api` to ECR on `main` and `develop`.

TrustEdge `docker-compose.yml` pulls that image — no TrustTwin source on EC2.

```bash
TRUSTTWIN_API_IMAGE=804012660077.dkr.ecr.us-east-1.amazonaws.com/trustedge-trusttwin-api:latest
```

## Local dev with TrustEdge compose

Clone this repo next to TrustEdge:

```text
~/Desktop/TrustEdge
~/Desktop/TrustTwin   ← this repo
```

```bash
cd TrustEdge && ./scripts/dev-up.sh
cd ../TrustTwin && TRUSTTWIN_API_URL=http://127.0.0.1:8080 go run ./cmd/trusttwin
```

Compose builds the API image from `../TrustTwin` via `TRUSTTWIN_BUILD_CONTEXT`.

## Privacy

TrustTwin does **not** collect window titles, URLs, keystrokes, screenshots, raw Wi‑Fi SSIDs, or full remote IP connection lists.

## API

See [docs/api.md](docs/api.md).


## Deploy API to EC2 (binary, legacy)

1. `GOOS=linux GOARCH=amd64 go build -o trusttwin-api ./cmd/trusttwin-api`
2. Copy binary to the instance
3. Open port 80/443 (or your chosen port)
4. Run with the **same Redis** as TrustEdge:

```bash
TRUSTTWIN_LISTEN=:8080 \
TRUSTTWIN_DATA_DIR=/var/lib/trusttwin \
REDIS_URL=redis://127.0.0.1:6379/0 \
./trusttwin-api
```

5. Point agents at `TRUSTTWIN_API_URL=https://your-ec2-or-alb` with `TRUSTTWIN_PRODUCTION=1` and a shared `TRUSTTWIN_ENROLL_TOKEN`.

The API keeps local files under `TRUSTTWIN_DATA_DIR` for auth/tokens, and mirrors live device state to Redis keys `twin:devices` and `twin:device:{id}:latest` for the TrustEdge digital twin graph.

## Project layout

```text
cmd/trusttwin/
cmd/trusttwin-api/
internal/
docs/api.md
```

## API

See [docs/api.md](docs/api.md).
