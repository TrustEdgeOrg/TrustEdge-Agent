"""Shared constants for Cursor governance hooks and MCP gateway config.

Values mirror the Go agent where noted:
  - internal/constants/telemetry.go
  - internal/agentaction/channel.go
  - internal/identity/catalog_cursor.go
  - internal/mcp/events/emit.go (inject dir layout)
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

# --- Telemetry event types (internal/constants/telemetry.go) ---
TYPE_AI_AGENT_ACTION = "ai_agent_action"
TYPE_PROCESS_START = "process_start"
TYPE_FILE_OPEN = "file_open"

# --- Agent action channels (internal/agentaction/channel.go) ---
CHANNEL_MCP = "MCP"
CHANNEL_SHELL = "SHELL"
CHANNEL_PROCESS = "PROCESS"
CHANNEL_NETWORK = "NETWORK"
CHANNEL_FILE = "FILE"
CHANNEL_SKILL = "SKILL"
CHANNEL_UNKNOWN = "UNKNOWN"

# --- Agent identity (internal/identity/catalog_cursor.go) ---
AGENT_TYPE_CURSOR = "cursor"

# --- Cursor hook evidence / bridge sources ---
EVIDENCE_CURSOR_HOOK = "cursor_hook"
EVIDENCE_HOST_BRIDGE = "cursor_hook_host_bridge"

# --- Cursor hook event names ---
HOOK_BEFORE_SUBMIT_PROMPT = "beforeSubmitPrompt"
HOOK_POST_TOOL_USE = "postToolUse"
HOOK_POST_TOOL_USE_FAILURE = "postToolUseFailure"

# --- Prompt / tool labels for ai_agent_action payloads ---
TOOL_PROMPT = "prompt"
TOOL_SKILL = "skill"
OPERATION_PROMPT_SUBMIT = "submit"

# --- MCP gateway defaults (cmd/trustedge-mcp-gateway, policy-github.json) ---
MCP_SERVER_GITHUB = "github"
MCP_AGENT_TYPE_CURSOR = AGENT_TYPE_CURSOR
GITHUB_TOOLSETS_DEFAULT = "repos,issues,pull_requests"
POLICY_GITHUB_REL = "testdata/mcp/policy-github.json"
GATEWAY_BIN_REL = "bin/trustedge-mcp-gateway"
GITHUB_MCP_SCRIPT_REL = "scripts/run-github-mcp.sh"

# --- Environment overrides ---
ENV_INJECT_DIR = "TRUSTEDGE_INJECT_DIR"
ENV_STATE_PATH = "TRUSTEDGE_AGENT_STATE_PATH"
ENV_LEGACY_STATE_PATH = "TRUSTTWIN_STATE_PATH"

# --- Cursor built-in tool names ---
READ_TOOLS = frozenset({"Read", "Grep", "Glob", "FetchMcpResource"})
WRITE_TOOLS = frozenset({"Write", "StrReplace", "Delete", "Edit", "EditNotebook"})
SHELL_TOOLS = frozenset({"Shell", "Bash"})
NETWORK_TOOLS = frozenset({"WebFetch", "WebSearch"})
TASK_TOOLS = frozenset({"Task"})
FILE_HOST_TOOLS = READ_TOOLS | WRITE_TOOLS

CHANNEL_BY_TOOL = {
    **{t: CHANNEL_FILE for t in READ_TOOLS | WRITE_TOOLS},
    **{t: CHANNEL_SHELL for t in SHELL_TOOLS},
    **{t: CHANNEL_NETWORK for t in NETWORK_TOOLS},
    **{t: CHANNEL_PROCESS for t in TASK_TOOLS},
}

CLASSIFICATION_BY_TOOL = {
    **{t: "read_only" for t in READ_TOOLS},
    **{t: "write" for t in WRITE_TOOLS},
    **{t: "external_communication" for t in SHELL_TOOLS | NETWORK_TOOLS},
    **{t: "unknown" for t in TASK_TOOLS},
}

# Mirrors internal/collect/process/secret_paths.go and detection-engine catalog.
SECRET_PATH_MARKERS = (
    "/.ssh/",
    "\\.ssh\\",
    "~/.ssh",
    "/.aws/",
    "\\.aws\\",
    "~/.aws",
    "/.kube/",
    "\\.kube\\",
    "~/.kube",
    "/.gnupg/",
    "id_rsa",
    "id_ed25519",
    "id_ecdsa",
    "credentials",
    ".env",
    "secrets.json",
    "secret.yaml",
    "secret.yml",
    "kubeconfig",
    "service_account.json",
    "application_default_credentials",
)

RISK_SECRET_READ = "secret_read"
RISK_SECRETS_GIT_PUSH = "secrets_git_push"
RISK_SECRET_WRITE = "secret_write"

CURSOR_APP_EXECUTABLE = "/Applications/Cursor.app/Contents/MacOS/Cursor"
ACTIVE_TURN_FILENAME = ".active-turn.json"

# Hook commands for Cursor hooks.json (project vs user scope).
HOOK_SCRIPT_REL = "scripts/cursor-governance/emit-tool-use.py"
HOOK_COMMAND = f"{sys.executable} {HOOK_SCRIPT_REL}"


def hook_script_path() -> Path:
    return repo_root() / HOOK_SCRIPT_REL


def hook_command_user() -> str:
    """Absolute command for ~/.cursor/hooks.json (runs from any workspace)."""
    return f"{sys.executable} {hook_script_path()}"


def hook_command_project() -> str:
    """Relative command for <repo>/.cursor/hooks.json (project root cwd)."""
    return HOOK_COMMAND


def _state_path_candidates(home: Path) -> list[Path]:
    """Match internal/config/config.go resolveStatePath ordering."""
    return [
        home / "Library/Application Support/TrustEdge Agent/state.json",
        home / "Library/Application Support/TrustTwin/state.json",
        home / ".local/share/TrustEdge Agent/state.json",
        home / ".local/share/TrustTwin/state.json",
    ]


def resolve_state_path() -> Path:
    for key in (ENV_STATE_PATH, ENV_LEGACY_STATE_PATH):
        raw = os.environ.get(key, "").strip()
        if raw:
            return Path(raw).expanduser()
    home = Path.home()
    for candidate in _state_path_candidates(home):
        if candidate.is_file():
            return candidate
    return _state_path_candidates(home)[0]


def default_inject_dir() -> Path:
    """Match internal/mcp/events.DefaultInjectDir (<stateDir>/inject)."""
    raw = os.environ.get(ENV_INJECT_DIR, "").strip()
    if raw:
        return Path(raw).expanduser()
    return resolve_state_path().parent / "inject"


def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent.parent
