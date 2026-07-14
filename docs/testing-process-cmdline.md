# Test process command lines locally

This guide captures live agent telemetry into `events.json` so you can verify the new `cmdline` field.

## What you need

- Go 1.22+
- Two terminals
- Branch: `feature/process-cmdline`

## 1. Build the agent

```bash
cd TrustEdge-Agent
make build
# → bin/trustedge-agent
```

Or without installing a binary:

```bash
go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent
```

On macOS CI-style builds use poll-only process monitoring:

```bash
CGO_ENABLED=0 go build -buildvcs=false -o bin/trustedge-agent ./cmd/trustedge-agent
```

With CGO enabled (default on macOS), the Endpoint Security watcher may fail without entitlements — the agent then falls back to poll. That is fine for cmdline testing; poll still fills `cmdline` from `ps`.

## 2. Start the capture server (Terminal 1)

```bash
cd TrustEdge-Agent
go run ./scripts/capture-events
```

You should see:

```text
capture server listening on http://127.0.0.1:8080
writing events to .../TrustEdge-Agent/events.json
```

This writes accepted events to **`events.json`** in the repo root (gitignored).

Example shape (also in [`testdata/events.example.json`](../testdata/events.example.json)):

```json
{
  "type": "process_start",
  "payload": {
    "pid": 4242,
    "ppid": 1000,
    "user": "elad",
    "comm": "curl",
    "executable": "/usr/bin/curl",
    "cmdline": "curl https://example.com"
  }
}
```

## 3. Run the agent (Terminal 2)

Use short intervals so process polls happen quickly:

```bash
cd TrustEdge-Agent

export TRUSTEDGE_AGENT_API_URL=http://127.0.0.1:8080
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
trustedge-agent: reporting to http://127.0.0.1:8080
trustedge-agent: posted batch (N events)
```

## 4. Generate a process with a clear cmdline (Terminal 3)

```bash
curl -sS -o /dev/null https://example.com
# or:
python3 -c 'import time; time.sleep(2)'
```

Wait a few seconds for the next process poll / watcher event and batch flush.

## 5. Inspect `events.json`

```bash
# all process events that include cmdline
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

**Success looks like:** a `process_start` whose `payload.cmdline` contains your command (e.g. `curl https://example.com` or `python3 -c ...`).

## 6. Reset and re-run

Stop the capture server (Ctrl+C) and start it again — it recreates an empty `events.json`.

Or:

```bash
echo '[]' > events.json
```

## Unit tests (no agent run)

```bash
CGO_ENABLED=0 go test ./internal/collect/ -count=1
```

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| No `events.json` growth | Capture server running? Agent `API_URL` correct? |
| Events but no `process_*` | `PROCESS_INTERVAL` not `0`? Spawn a short-lived process |
| Process events without `cmdline` | Wrong branch? Rebuild agent; on Linux poll uses `/proc` |
| macOS ES watcher errors | Expected without entitlements; poll still provides cmdline |
| Permission / keyring prompts | Local env is fine; use empty enroll token for capture server |
