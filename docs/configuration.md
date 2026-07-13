# Configuration

All settings are environment variables. Copy [.env.example](../.env.example) as a starting point.

## Agent (`trustedge-agent`)

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_API_URL` | `http://127.0.0.1:8080` | Ingest API base URL (no trailing slash) — see [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API) |
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

## API server configuration

Redis, Kafka, and ingest API settings live in [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API).

## Legacy environment variables

`TRUSTTWIN_*` environment variables remain supported as fallbacks during migration (for example `TRUSTTWIN_API_URL` → `TRUSTEDGE_AGENT_API_URL`). Prefer the `TRUSTEDGE_AGENT_*` names for new deployments.

## CI

Agent CI (`.github/workflows/agent-ci.yml`) runs `go test ./...` and builds on Linux, macOS, and Windows.
