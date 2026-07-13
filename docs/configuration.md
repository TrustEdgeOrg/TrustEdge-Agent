# Configuration

All settings are environment variables. Copy [.env.example](../.env.example) as a starting point.

## Agent (`trustedge-agent`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_API_URL` | `http://127.0.0.1:8080` | Ingest API base URL (no trailing slash) |
| `TRUSTEDGE_AGENT_ENROLL_TOKEN` | _(empty)_ | Bearer token for `POST /v1/register`; required when `TRUSTEDGE_AGENT_PRODUCTION=1` |
| `TRUSTEDGE_AGENT_PRODUCTION` | `0` | `1` requires HTTPS API URL and enroll token |
| `TRUSTEDGE_AGENT_STATE_PATH` | Platform default | Device ID state file path (see [Agent](agent.md)) |
| `TRUSTEDGE_AGENT_DETAILS_INTERVAL` | `60` | `client_details` interval (seconds or Go duration) |
| `TRUSTEDGE_AGENT_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat interval |
| `TRUSTEDGE_AGENT_NETWORK_DEBOUNCE` | `2` | Debounce for network change events |
| `TRUSTEDGE_AGENT_ACTION_INTERVAL` | `60` | `action_summary` sampling interval |
| `TRUSTEDGE_AGENT_PROCESS_INTERVAL` | `10` | Process poll interval; `0` disables process monitoring |
| `TRUSTEDGE_AGENT_EVENT_BATCH_SIZE` | `32` | Max events per batch before flush |
| `TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH` | `2` | Max seconds between batch flushes |
| `TRUSTEDGE_AGENT_PUBLIC_IP_URL` | ipify default | Public IP lookup URL; set to `off` to disable |

### Interval format

Duration env vars accept:

- A number of seconds: `60`, `30.5`
- A Go duration string: `2s`, `1m`, `500ms`

### Production agent checklist

```bash
export TRUSTEDGE_AGENT_PRODUCTION=1
export TRUSTEDGE_AGENT_API_URL=https://api.example.com
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<from server>
```

## API (`trustedge-agent-api`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_LISTEN` | `:8080` | HTTP listen address |
| `TRUSTEDGE_AGENT_ENROLL_TOKEN` | _(empty)_ | Required for registration when set; required in production |
| `TRUSTEDGE_AGENT_PRODUCTION` | `0` | `1` requires enroll token and Redis |
| `TRUSTEDGE_AGENT_DATA_DIR` | `data` | Dev fallback directory for `devices.json` / `events.jsonl` |
| `TRUSTEDGE_AGENT_PERSIST_FILES` | auto | `1`/`0` to force disk writes on or off |
| `TRUSTEDGE_AGENT_REDIS_URL` | _(empty)_ | Redis URL for live device state |
| `REDIS_URL` | _(empty)_ | Fallback Redis URL (same as TrustEdge stack) |
| `KAFKA_BROKERS` | _(empty)_ | Comma-separated brokers; unset disables Kafka publish |
| `KAFKA_TOPIC` | `trustedge.agent.events` | Kafka topic for ingested events |

### Persistence modes

| `TRUSTEDGE_AGENT_PRODUCTION` | `TRUSTEDGE_AGENT_REDIS_URL` | Behavior |
|------------------------|----------------------|----------|
| `0` | unset | In-memory + optional disk (`data/`) |
| `0` | set | In-memory + Redis mirror + optional disk |
| `1` | **required** | Redis only; disk persistence disabled |

Override disk writes in dev with `TRUSTEDGE_AGENT_PERSIST_FILES=0` or `=1`.

### Production API checklist

```bash
export TRUSTEDGE_AGENT_PRODUCTION=1
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<generate-secure-token>
export REDIS_URL=redis://127.0.0.1:6379/0
export KAFKA_BROKERS=redpanda:9092
export KAFKA_TOPIC=trustedge.agent.events
```

## Docker / EC2 (TrustEdge compose)

The TrustEdge `docker-compose.yml` profile `agent` runs `trustedge-agent-api` from ECR. Environment is injected by compose — see [AWS deploy](../aws/README.md).

Typical EC2 env (set in TrustEdge compose or `/etc/trustedge/`):

- `TRUSTEDGE_AGENT_PRODUCTION=1`
- `TRUSTEDGE_AGENT_ENROLL_TOKEN` — from `/etc/trustedge/agent-enroll.token`
- `REDIS_URL` — shared with TrustEdge backend
- `KAFKA_BROKERS` — Redpanda broker inside compose network

## Legacy environment variables

`TRUSTTWIN_*` environment variables remain supported as fallbacks during migration (for example `TRUSTTWIN_API_URL` → `TRUSTEDGE_AGENT_API_URL`). Prefer the `TRUSTEDGE_AGENT_*` names for new deployments.

## CI

Agent CI (`.github/workflows/agent-ci.yml`) runs `go test ./...` and builds on Linux, macOS, and Windows.

API deploy CI (`.github/workflows/deploy-api.yml`) builds the Docker image, pushes to ECR, and deploys to EC2 on push to `develop` or `main`.
