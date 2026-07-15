# Configuration

All settings are environment variables. Copy [.env.example](../.env.example) as a starting point.

## Agent (`trustedge-agent`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_API_URL` | _(required)_ | Ingest API base URL (no trailing slash). Local example: `http://127.0.0.1:8080`. See [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) |
| `TRUSTEDGE_AGENT_ENROLL_TOKEN` | _(empty)_ | Bearer token for `POST /v1/register`; required when `TRUSTEDGE_AGENT_PRODUCTION=1` |
| `TRUSTEDGE_AGENT_PRODUCTION` | `0` | `1` requires HTTPS API URL and enroll token |
| `TRUSTEDGE_AGENT_COMPRESS` | `1` | `0` disables zstd on `/v1/events` (compat with older ingest images) |
| `TRUSTEDGE_AGENT_BATCH` | `1` | `0` sends one Event object per POST instead of `{"events":[...]}` (compat with older ingest images) |
| `TRUSTEDGE_AGENT_STATE_PATH` | Platform default | Device ID state file path (see [Agent guide](agent.md)) |
| `TRUSTEDGE_AGENT_DETAILS_INTERVAL` | `60` | `client_details` interval (seconds or Go duration) |
| `TRUSTEDGE_AGENT_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat interval |
| `TRUSTEDGE_AGENT_NETWORK_DEBOUNCE` | `2` | Debounce for network change events |
| `TRUSTEDGE_AGENT_ACTION_INTERVAL` | `60` | How often to emit `action_summary` |
| `TRUSTEDGE_AGENT_ACTION_SAMPLE_INTERVAL` | `5` | How often to sample the foreground app inside each action window |
| `TRUSTEDGE_AGENT_PROCESS_INTERVAL` | `10` | Process poll interval; `0` disables process monitoring |
| `TRUSTEDGE_AGENT_EVENT_BATCH_SIZE` | `32` | Max events per batch before flush |
| `TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH` | `2` | Max seconds between batch flushes |
| `TRUSTEDGE_AGENT_EVENT_QUEUE_CAPACITY` | `4096` | Durable offline ring size; when full, oldest events are overwritten |
| `TRUSTEDGE_AGENT_EVENT_QUEUE_PATH` | next to state file | Path for persisted pending events (`events.queue.json`) |
| `TRUSTEDGE_AGENT_EVENT_RETRY_MAX` | `60` | Max backoff between retries after a failed upload |
| `TRUSTEDGE_AGENT_LOG_FORMAT` | `text` | `text` or `json` structured logs |
| `TRUSTEDGE_AGENT_METRICS_INTERVAL` | `5m` | Periodic agent status log; `0` disables |
| `TRUSTEDGE_AGENT_PUBLIC_IP_URL` | provider default | Public IP lookup URL for `network_summary`; set to `off` to disable |

### Interval format

Duration env vars accept:

- A number of seconds: `60`, `30.5`
- A Go duration string: `2s`, `1m`, `500ms`

### Production checklist

```bash
export TRUSTEDGE_AGENT_PRODUCTION=1
export TRUSTEDGE_AGENT_API_URL=https://your-ingest.example
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<from your API>
```

Do not commit real API hosts, enroll tokens, or device tokens. Prefer placeholders in docs and `.env.example`.

## API server configuration

Redis, Kafka, and ingest API settings live in [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API).

## Legacy environment variables

`TRUSTTWIN_*` names remain supported as fallbacks during migration (for example `TRUSTTWIN_API_URL` → `TRUSTEDGE_AGENT_API_URL`). Prefer `TRUSTEDGE_AGENT_*` for new deployments.

## CI

Agent CI (`.github/workflows/agent-ci.yml`) runs `go test ./...` and builds on Linux, macOS, and Windows.
