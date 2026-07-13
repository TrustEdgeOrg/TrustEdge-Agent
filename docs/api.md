# TrustTwin API

Ingest API for the [TrustEdge](https://github.com/TrustEdgeOrg/TrustEdge) security observability platform. Agents POST endpoint telemetry; production mode mirrors to Redis and publishes to Kafka for detection.

Base URL example: `http://127.0.0.1:8080`

## Event envelope

All telemetry uses one envelope:

```json
{
  "event_id": "evt_...",
  "device_id": "dev_...",
  "type": "client_details | network_summary | action_summary | process_start | process_exit",
  "ts": "2026-07-03T21:00:00Z",
  "payload": {}
}
```

## Endpoints

### `GET /healthz`

Health check.

### `POST /v1/register`

Register a client (device). When `TRUSTTWIN_ENROLL_TOKEN` is set (required in production), send `Authorization: Bearer <enroll_token>`.

Request:

```json
{
  "device_id": "optional-existing-id",
  "hostname": "elad-mbp",
  "os": "darwin",
  "os_version": "15.5",
  "arch": "arm64",
  "agent_version": "0.1.0"
}
```

Response:

```json
{
  "device_id": "dev_...",
  "device_token": "tok_..."
}
```

### `POST /v1/events`

Ingest one event. Requires `Authorization: Bearer <device_token>`.

### Production

When `TRUSTTWIN_PRODUCTION=1`:

- API refuses to start without `TRUSTTWIN_ENROLL_TOKEN`.
- Agents refuse to start without `TRUSTTWIN_ENROLL_TOKEN` and an `https://` API URL.
- On macOS, device tokens are stored in Keychain, not `state.json`.

### `GET /v1/clients/{id}`

Return latest client details and recent events (for demos / debugging).

## Pillars

### `client_details`

Identity + presence heartbeat.

| Field | Description |
|---|---|
| `hostname` | Machine name |
| `os` | `darwin`, `linux`, … |
| `os_version` | OS version string |
| `arch` | `arm64`, `amd64` |
| `agent_version` | Agent version |
| `timezone` | Local timezone abbreviation |
| `status` | `online` |
| `uptime_sec` | Agent uptime |

### `network_summary`

Coarse network posture (no raw connection tables).

| Field | Description |
|---|---|
| `public_ip` | Public IPv4 |
| `network_type` | `wifi`, `ethernet`, `unknown` |
| `listening_count` | Listening sockets |
| `established_count` | Established TCP |
| `top_remote_ports` | `[{port, count}, …]` |
| `foreground_app_connections` | Optional app-linked count |

### `action_summary`

Short-window behavior (no daily rollup in v1).

| Field | Description |
|---|---|
| `window_start` / `window_end` | Window bounds (RFC3339) |
| `focus` | App focus entries |
| `presence` | `active` or `idle` |
| `idle_sec` | Seconds since last input |
| `app_switches` | Focus changes in window |

### `process_start` / `process_exit`

EDR-lite process visibility (metadata only).

| Field | Description |
|---|---|
| `pid` | Process ID |
| `ppid` | Parent process ID |
| `user` | Owning user |
| `comm` | Short process name |
| `executable` | Binary path or comm |

## Privacy

TrustTwin does **not** collect window titles, URLs, keystrokes, screenshots, raw SSIDs, or full remote IP connection lists.
