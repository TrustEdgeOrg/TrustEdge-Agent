# Test process command lines locally

This guide captures live agent telemetry into `events.json` so you can verify the new `cmdline` field.

## What you need

- Go 1.22+
- Two terminals
- Branch: `feature/process-cmdline`

## 1. Build the agent

Default `make build` uses **`CGO_ENABLED=0`** (poll-only). That avoids the
`EndpointSecurity` linker error on machines without the full macOS SDK.

```bash
cd TrustEdge-Agent
make build
# → bin/trustedge-agent
```

Poll still collects `cmdline` via `ps` on macOS — enough to verify this feature.

Only if you have entitlements / SDK for Endpoint Security:

```bash
make build-cgo
```

## 2. Start the capture server (Terminal 1)

Capture defaults to **`:18080`** so it does not clash with Docker Compose / TrustEdge on `:8080`.

```bash
cd TrustEdge-Agent
make capture-events
# or: go run ./scripts/capture-events
```

You should see:

```text
capture server listening on http://127.0.0.1:18080
writing events to .../TrustEdge-Agent/events.json
```

If the port is busy:

```bash
TRUSTEDGE_CAPTURE_ADDR=:18081 go run ./scripts/capture-events
```

## 3. Run the agent (Terminal 2)

Use short intervals so process polls happen quickly:

```bash
cd TrustEdge-Agent

export TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:18080
export TRUSTEDGE_AGENT_PROCESS_INTERVAL=2
export TRUSTEDGE_AGENT_EVENT_BATCH_FLUSH=1
export TRUSTEDGE_AGENT_DETAILS_INTERVAL=30
export TRUSTEDGE_AGENT_NETWORK_INTERVAL=30
export TRUSTEDGE_AGENT_ACTION_INTERVAL=30

./bin/trustedge-agent
```

Look for log lines like:

```text
trustedge-agent: registered device ...
trustedge-agent: reporting to http://127.0.0.1:18080
trustedge-agent: posted batch (N events)
```

## 4. Generate a process with a clear cmdline (Terminal 3)

```bash
curl -sS -o /dev/null https://example.com
# or:
python3 -c 'import time; time.sleep(2)'
```

Wait a few seconds for the next process poll and batch flush.

## 5. Inspect `events.json`

```bash
python3 - <<'PY'
import json
from pathlib import Path
events = json.loads(Path("events.json").read_text())
procs = [e for e in events if e.get("type", "").startswith("process_")]
print(f"total events={len(events)} process={len(procs)}")
for e in procs[-10:]:
    p = e.get("payload") or {}
    print(f"{e['type']:14} pid={p.get('pid')} cmdline={p.get('cmdline')!r}")
PY
```

Or with `jq` if installed:

```bash
jq '[.[] | select(.type|startswith("process_"))] | .[-5:] | .[] | {type, pid: .payload.pid, cmdline: .payload.cmdline}' events.json
```

**Success looks like:** a `process_start` whose `payload.cmdline` contains your command (e.g. `curl https://example.com`).

## 6. Reset and re-run

Stop the capture server (Ctrl+C) and start it again — it recreates an empty `events.json`.

Or:

```bash
echo '[]' > events.json
```

## Unit tests (no agent run)

```bash
make test
```

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| `bind: address already in use` on 8080 | Use default `:18080` (Docker often owns 8080) |
| `framework 'EndpointSecurity' not found` | Use `make build` (`CGO_ENABLED=0`), not plain CGO `go build` |
| No `events.json` growth | Capture running? Agent `API_URL` matches port `18080`? |
| Events but no `process_*` | `PROCESS_INTERVAL` not `0`? Spawn a short-lived process |
| Process events without `cmdline` | Rebuild on this branch; macOS poll uses `ps` args |
| Permission / keyring prompts | Local env is fine; empty enroll token works with capture |
