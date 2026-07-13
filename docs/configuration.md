# Configuration

All settings are environment variables. Copy [.env.example](../.env.example) as a starting point.

## Agent (`trusttwin`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTTWIN_API_URL` | `http://127.0.0.1:8080` | Ingest API base URL (no trailing slash) |
| `TRUSTTWIN_ENROLL_TOKEN` | _(empty)_ | Bearer token for `POST /v1/register`; required when `TRUSTTWIN_PRODUCTION=1` |
| `TRUSTTWIN_PRODUCTION` | `0` | `1` requires HTTPS API URL and enroll token |
| `TRUSTTWIN_STATE_PATH` | Platform default | Device ID state file path (see [Agent](agent.md)) |
| `TRUSTTWIN_DETAILS_INTERVAL` | `60` | `client_details` interval (seconds or Go duration) |
| `TRUSTTWIN_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat interval |
| `TRUSTTWIN_NETWORK_DEBOUNCE` | `2` | Debounce for network change events |
| `TRUSTTWIN_ACTION_INTERVAL` | `60` | `action_summary` sampling interval |
| `TRUSTTWIN_PROCESS_INTERVAL` | `10` | Process poll interval; `0` disables process monitoring |
| `TRUSTTWIN_EVENT_BATCH_SIZE` | `32` | Max events per batch before flush |
| `TRUSTTWIN_EVENT_BATCH_FLUSH` | `2` | Max seconds between batch flushes |
| `TRUSTTWIN_PUBLIC_IP_URL` | ipify default | Public IP lookup URL; set to `off` to disable |

### Interval format

Duration env vars accept:

- A number of seconds: `60`, `30.5`
- A Go duration string: `2s`, `1m`, `500ms`

### Production agent checklist

```bash
export TRUSTTWIN_PRODUCTION=1
export TRUSTTWIN_API_URL=https://api.example.com
export TRUSTTWIN_ENROLL_TOKEN=<from server>
```

## API (`trusttwin-api`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTTWIN_LISTEN` | `:8080` | HTTP listen address |
| `TRUSTTWIN_ENROLL_TOKEN` | _(empty)_ | Required for registration when set; required in production |
| `TRUSTTWIN_PRODUCTION` | `0` | `1` requires enroll token and Redis |
| `TRUSTTWIN_DATA_DIR` | `data` | Dev fallback directory for `devices.json` / `events.jsonl` |
| `TRUSTTWIN_PERSIST_FILES` | auto | `1`/`0` to force disk writes on or off |
| `TRUSTTWIN_REDIS_URL` | _(empty)_ | Redis URL for live device state |
| `REDIS_URL` | _(empty)_ | Fallback Redis URL (same as TrustEdge stack) |
| `KAFKA_BROKERS` | _(empty)_ | Comma-separated brokers; unset disables Kafka publish |
| `KAFKA_TOPIC` | `trusttwin.events` | Kafka topic for ingested events |

### Persistence modes

| `TRUSTTWIN_PRODUCTION` | `TRUSTTWIN_REDIS_URL` | Behavior |
|------------------------|----------------------|----------|
| `0` | unset | In-memory + optional disk (`data/`) |
| `0` | set | In-memory + Redis mirror + optional disk |
| `1` | **required** | Redis only; disk persistence disabled |

Override disk writes in dev with `TRUSTTWIN_PERSIST_FILES=0` or `=1`.

### Production API checklist

```bash
export TRUSTTWIN_PRODUCTION=1
export TRUSTTWIN_ENROLL_TOKEN=<generate-secure-token>
export REDIS_URL=redis://127.0.0.1:6379/0
export KAFKA_BROKERS=redpanda:9092
export KAFKA_TOPIC=trusttwin.events
```

## Docker / EC2 (TrustEdge compose)

The TrustEdge `docker-compose.yml` profile `trusttwin` runs `trusttwin-api` from ECR. Environment is injected by compose — see [AWS deploy](../aws/README.md).

Typical EC2 env (set in TrustEdge compose or `/etc/trustedge/`):

- `TRUSTTWIN_PRODUCTION=1`
- `TRUSTTWIN_ENROLL_TOKEN` — from `/etc/trustedge/trusttwin-enroll.token`
- `REDIS_URL` — shared with TrustEdge backend
- `KAFKA_BROKERS` — Redpanda broker inside compose network

## CI

Agent CI (`.github/workflows/agent-ci.yml`) runs `go test ./...` and builds on Linux, macOS, and Windows.

API deploy CI (`.github/workflows/deploy-api.yml`) builds the Docker image, pushes to ECR, and deploys to EC2 on push to `develop` or `main`.
