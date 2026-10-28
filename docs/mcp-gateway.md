# MCP Gateway (AI Agent Action Governance)

Local stdio MCP gateway that sits between an AI client (e.g. Claude Code) and a downstream MCP server.

```text
Claude Code  →  trustedge-mcp-gateway  →  downstream MCP server
                      │
                      ▼
              inject/*.json (ai_agent_action)
                      │
                      ▼
              TrustEdge Agent EventBatcher → Agent-API → detection-engine
```

## Limitation (MVP)

- **Transport:** stdio JSON-RPC with `Content-Length` framing only.
- SSE / streamable HTTP MCP are **not** supported yet.

## Build

```bash
make build-mcp-gateway
# → bin/trustedge-mcp-gateway
```

## Run

```bash
./bin/trustedge-mcp-gateway \
  -server customer-db \
  -agent-type claude_code \
  -policy testdata/mcp/policy.json \
  -inject-dir .local-state/inject \
  -downstream ./bin/mock-customer-db-mcp
```

Point Claude Code / Cursor `mcp.json` at the gateway binary instead of the raw server.

## Policy

Deterministic JSON rules classify tools (`sensitive_read`, `write`, …) and decide `allowed` / `blocked`. See [testdata/mcp/policy.json](../testdata/mcp/policy.json).

Tool **argument values are never** written to telemetry by default (keys-only sanitizer).

## Wire Cursor → GitHub MCP (real server)

TrustEdge gateway is **stdio-only**, so use the **local** GitHub MCP (Docker), not the remote `https://api.githubcopilot.com/mcp/` HTTP endpoint.

1. Create a GitHub PAT: https://github.com/settings/personal-access-tokens/new  
   (repo read is enough for a first test; grant only what you need.)
2. Put the token in Cursor MCP settings for server **github** → env `GITHUB_PERSONAL_ACCESS_TOKEN`  
   (or edit `~/.cursor/mcp.json` `env` field — do not commit the token.)
3. Config already points at:
   - gateway: `bin/trustedge-mcp-gateway`
   - policy: `testdata/mcp/policy-github.json`
   - downstream: `scripts/run-github-mcp.sh` → `ghcr.io/github/github-mcp-server`
4. Reload Cursor MCP. Ask: “Using the github MCP server, list issues in OWNER/REPO” or “get_file_contents for README”.
5. Check TrustEdge → **AI agent actions** for governed `github/...` tool calls.

Sensitive reads (e.g. `get_file_contents`, `search_code`) are classified in the GitHub policy for correlation alerts.

## Cursor built-in tools (hooks)

The MCP gateway only sees MCP `tools/call` traffic. Built-in Cursor tools (Shell, Read, Write, Grep, Task, …) are observed via project hooks:

```text
User prompt  →  beforeSubmitPrompt  →  inject (prompt turn)
Agent tools  →  postToolUse         →  inject (ai_agent_action)
                      │
                      ▼
              TrustEdge Agent inject watcher (same path as MCP gateway)
```

Setup (one-time, local):

```bash
# Recommended: global hooks (any Cursor project) + project MCP config
bash scripts/cursor-governance/install-cursor-config.sh

# Or install only user-level hooks (~/.cursor/hooks.json)
bash scripts/cursor-governance/install-cursor-config.sh --user
```

**Global hooks** (`~/.cursor/hooks.json`) emit prompts and built-in tool use from **any**
workspace into the same agent inject dir. The install script removes project-level
`TrustEdge-Agent/.cursor/hooks.json` when using global mode so prompts are not duplicated.

Project-only mode (repo-scoped hooks):

```bash
bash scripts/cursor-governance/install-cursor-config.sh --project
```

Shared hook/MCP constants live in `scripts/cursor-governance/constants.py` (mirrors
Go telemetry and agentaction packages).

Files:

- `scripts/cursor-governance/hooks.json.example` → `~/.cursor/hooks.json` (global) or `.cursor/hooks.json` (project)
- `scripts/cursor-governance/mcp.json.example` → `.cursor/mcp.json`
- `scripts/cursor-governance/emit-tool-use.py` — writes inject JSON for hooks
- `scripts/cursor-governance/constants.py` — shared event/channel/path constants

Default inject dir: resolved from agent state path (legacy `TrustTwin` or
`TrustEdge Agent`). Override with `TRUSTEDGE_INJECT_DIR`.

Reload Cursor hooks after editing, then run the agent with inject watcher active. Check TrustEdge → agent detail → **Agent actions**.

Run hook unit tests:

```bash
python3 scripts/cursor-governance/test_emit_tool_use.py
```
