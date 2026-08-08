# <img src="assets/icons/config.svg" width="28" height="28" align="absmiddle" alt="" /> Configuration

All settings are environment variables. Copy [`.env.example`](../.env.example) as a starting point.

> **In one breath:** One required URL (`TRUSTEDGE_AGENT_API_URL`). Everything else has a sensible default — production mode tightens HTTPS + enroll.

<p align="center">
  <img src="assets/icons/lock.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#connection--auth">Auth</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#collectors">Collectors</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/queue.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#batching--reliability">Batching</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/flow.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#interval-format">Intervals</a>
  &nbsp;·&nbsp;
  <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" />
  &nbsp;<a href="#related">Related</a>
</p>

> **Also see**
> <img src="assets/icons/agent.svg" width="16" height="16" align="absmiddle" alt="" /> [Agent guide](agent.md)
> · <img src="assets/icons/collection.svg" width="16" height="16" align="absmiddle" alt="" /> [Collection](collection.md)
> · <img src="assets/icons/architecture.svg" width="16" height="16" align="absmiddle" alt="" /> [Docs hub](README.md)

---

## <img src="assets/icons/lock.svg" width="22" height="22" align="absmiddle" alt="" /> Connection & auth

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_API_URL` | _(required)_ | Ingest API base URL (no trailing slash). Example: `http://127.0.0.1:8080` |
| `TRUSTEDGE_AGENT_ENROLL_TOKEN` | _(empty)_ | Bearer for `POST /v1/register`; required when `PRODUCTION=1` |
| `TRUSTEDGE_AGENT_PRODUCTION` | `0` | `1` requires HTTPS API URL and enroll token |
| `TRUSTEDGE_AGENT_STATE_PATH` | Platform default | Device ID state file ([paths](agent.md#default-state-paths)) |

### Production checklist

```bash
export TRUSTEDGE_AGENT_PRODUCTION=1
export TRUSTEDGE_AGENT_API_URL=https://your-ingest.example
export TRUSTEDGE_AGENT_ENROLL_TOKEN=<from your API>
```

> Do not commit real API hosts, enroll tokens, or device tokens.

---

## <img src="assets/icons/compress.svg" width="22" height="22" align="absmiddle" alt="" /> Wire format

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_COMPRESS` | `1` | `0` disables zstd on `/v1/events` |
| `TRUSTEDGE_AGENT_BATCH` | `1` | `0` sends one Event object per POST instead of `{"events":[...]}` |

---

## <img src="assets/icons/collection.svg" width="22" height="22" align="absmiddle" alt="" /> Collectors

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_DETAILS_INTERVAL` | `60` | `client_details` interval |
| `TRUSTEDGE_AGENT_NETWORK_INTERVAL` | `60` | `network_summary` heartbeat |
| `TRUSTEDGE_AGENT_NETWORK_DEBOUNCE` | `2` | Debounce for network change events |
| `TRUSTEDGE_AGENT_ACTION_INTERVAL` | `60` | How often to emit `action_summary` |
| `TRUSTEDGE_AGENT_ACTION_SAMPLE_INTERVAL` | `5` | Foreground sample rate inside each action window |
| `TRUSTEDGE_AGENT_PROCESS_INTERVAL` | `10` | Process poll; `0` disables process monitoring |
| `TRUSTEDGE_AGENT_SECURITY_INTERVAL` | `30` | Security lifecycle poll; `0` disables |
| `TRUSTEDGE_AGENT_KNOWN_AI_INTERVAL` | `60` | AI tools inventory poll; `0` disables |
| `TRUSTEDGE_AGENT_PUBLIC_IP_URL` | provider default | Public IP lookup; set `off` to disable |

---

## <img src="assets/icons/queue.svg" width="22" height="22" align="absmiddle" alt="" /> Batching & reliability

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_EVENT_BATCH_SIZE` | `32` | Max events per batch before flush |
| `TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH` | `2` | Max seconds between batch flushes |
| `TRUSTEDGE_AGENT_EVENT_QUEUE_CAPACITY` | `4096` | Durable offline ring size; overwrite oldest when full |
| `TRUSTEDGE_AGENT_EVENT_QUEUE_PATH` | next to state file | Persisted pending events (`events.queue.json`) |
| `TRUSTEDGE_AGENT_EVENT_RETRY_MAX` | `60` | Max backoff between retries after a failed upload |

---

## <img src="assets/icons/agent.svg" width="22" height="22" align="absmiddle" alt="" /> Operability

| Variable | Default | Description |
|----------|---------|-------------|
| `TRUSTEDGE_AGENT_LOG_FORMAT` | `text` | `text` or `json` structured logs |
| `TRUSTEDGE_AGENT_METRICS_INTERVAL` | `5m` | Periodic agent status log; `0` disables |

---

## <img src="assets/icons/flow.svg" width="22" height="22" align="absmiddle" alt="" /> Interval format

Duration env vars accept:

| Form | Example |
|------|---------|
| Seconds as a number | `60`, `30.5` |
| Go duration string | `2s`, `1m`, `500ms` |

---

## <img src="assets/icons/upload.svg" width="22" height="22" align="absmiddle" alt="" /> API server configuration

Redis, Kafka, and ingest API settings live in [TrustEdge-Agent-API](https://github.com/TrustEdgeOrg/TrustEdge-Agent-API).

---

## <img src="assets/icons/layout.svg" width="22" height="22" align="absmiddle" alt="" /> Legacy environment variables

`TRUSTTWIN_*` names remain supported as fallbacks during migration (for example `TRUSTTWIN_API_URL` → `TRUSTEDGE_AGENT_API_URL`). Prefer `TRUSTEDGE_AGENT_*` for new deployments.

---

## <img src="assets/icons/platforms.svg" width="22" height="22" align="absmiddle" alt="" /> CI

Agent CI (`.github/workflows/agent-ci.yml`) runs `go test ./...` and builds on Linux, macOS, and Windows.

---

## Related

| | Doc |
|---|-----|
| <img src="assets/icons/agent.svg" width="18" height="18" align="absmiddle" alt="" /> | [Agent guide](agent.md) |
| <img src="assets/icons/collection.svg" width="18" height="18" align="absmiddle" alt="" /> | [Collection & batching](collection.md) |
| <img src="assets/icons/architecture.svg" width="18" height="18" align="absmiddle" alt="" /> | [Docs hub](README.md) |
