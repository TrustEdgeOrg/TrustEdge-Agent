#!/usr/bin/env python3
"""Emit Cursor hook events as TrustEdge inject files.

Built-in Cursor tools (Shell, Read, Write, …) bypass the MCP gateway. This hook
writes ai_agent_action envelopes for the Asked/Happened UI, and also bridges
Shell/file tools into process_start / file_open inject events so short-lived
Cursor sandbox commands still appear under On the host (OS poll often misses them).

MCP tools are skipped here when the trustedge-mcp-gateway already governs them.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from constants import (
    ACTIVE_TURN_FILENAME,
    AGENT_TYPE_CURSOR,
    CHANNEL_BY_TOOL,
    CHANNEL_SKILL,
    CHANNEL_UNKNOWN,
    CLASSIFICATION_BY_TOOL,
    CURSOR_APP_EXECUTABLE,
    EVIDENCE_CURSOR_HOOK,
    EVIDENCE_HOST_BRIDGE,
    FILE_HOST_TOOLS,
    HOOK_BEFORE_SUBMIT_PROMPT,
    HOOK_POST_TOOL_USE,
    HOOK_POST_TOOL_USE_FAILURE,
    NETWORK_TOOLS,
    OPERATION_PROMPT_SUBMIT,
    READ_TOOLS,
    RISK_SECRET_READ,
    RISK_SECRETS_GIT_PUSH,
    RISK_SECRET_WRITE,
    SECRET_PATH_MARKERS,
    SHELL_TOOLS,
    TASK_TOOLS,
    TOOL_PROMPT,
    TOOL_SKILL,
    TYPE_AI_AGENT_ACTION,
    TYPE_FILE_OPEN,
    TYPE_PROCESS_START,
    WRITE_TOOLS,
    default_inject_dir,
)

READ_TOOLS_SET = READ_TOOLS
WRITE_TOOLS_SET = WRITE_TOOLS
SHELL_TOOLS_SET = SHELL_TOOLS


def inject_dir() -> Path:
    return default_inject_dir()


def utc_now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def is_mcp_tool(tool_name: str) -> bool:
    name = (tool_name or "").strip()
    if not name:
        return False
    if name.startswith("MCP:"):
        return True
    if name.startswith("mcp__"):
        return True
    return False


def conversation_id(event: dict[str, Any]) -> str | None:
    for key in ("conversation_id", "session_id"):
        val = event.get(key)
        if isinstance(val, str) and val.strip():
            return val.strip()
    return None


def turn_id(event: dict[str, Any]) -> str | None:
    val = event.get("generation_id")
    if isinstance(val, str) and val.strip():
        return val.strip()
    return None


def truncate(text: str, limit: int = 240) -> str:
    text = re.sub(r"\s+", " ", (text or "").strip())
    if len(text) <= limit:
        return text
    return text[: limit - 1] + "…"


def full_prompt_text(text: str, limit: int = 100_000) -> str:
    """Keep user prompt text intact for the Asked UI (preserve newlines)."""
    text = (text or "").strip()
    if len(text) <= limit:
        return text
    return text[: limit - 1] + "…"


def destination(tool_name: str, tool_input: dict[str, Any]) -> str:
    if tool_name in SHELL_TOOLS_SET:
        cmd = tool_input.get("command") or tool_input.get("cmd")
        if isinstance(cmd, str) and cmd.strip():
            return truncate(cmd)

    for key in (
        "path",
        "target_file",
        "file_path",
        "pattern",
        "glob_pattern",
        "url",
        "search_term",
        "description",
        "subagent_type",
        "prompt",
    ):
        val = tool_input.get(key)
        if isinstance(val, str) and val.strip():
            return truncate(val)

    return ""


def operation(tool_name: str) -> str:
    if tool_name in READ_TOOLS_SET:
        return "read"
    if tool_name in WRITE_TOOLS_SET:
        return "write"
    if tool_name in SHELL_TOOLS_SET:
        return "execute"
    if tool_name in NETWORK_TOOLS:
        return "fetch"
    if tool_name in TASK_TOOLS:
        return "delegate"
    return "invoke"


def channel_for(tool_name: str) -> str:
    return CHANNEL_BY_TOOL.get(tool_name, CHANNEL_UNKNOWN)


def classification_for(tool_name: str) -> str:
    return CLASSIFICATION_BY_TOOL.get(tool_name, "unknown")


def path_looks_secret(path: str) -> bool:
    p = (path or "").lower()
    return any(m.lower() in p for m in SECRET_PATH_MARKERS)


def apply_risk(payload: dict[str, Any], tool_name: str, tool_input: dict[str, Any]) -> None:
    dest = str(payload.get("destination") or "")
    risk_type = None
    if tool_name in SHELL_TOOLS_SET:
        cmd = dest.lower()
        if "git" in cmd and "push" in cmd and (".env" in cmd or "secret" in cmd):
            risk_type = RISK_SECRETS_GIT_PUSH
        elif path_looks_secret(dest):
            risk_type = RISK_SECRET_READ
    elif tool_name in WRITE_TOOLS_SET and path_looks_secret(dest):
        risk_type = RISK_SECRET_WRITE
    elif tool_name in READ_TOOLS_SET and path_looks_secret(dest):
        risk_type = RISK_SECRET_READ

    if not risk_type:
        return
    payload["risk"] = "high"
    payload["risk_type"] = risk_type
    evidence = payload.get("evidence")
    if not isinstance(evidence, list):
        evidence = []
    evidence = list(evidence)
    evidence.append(f"risk:{risk_type}")
    payload["evidence"] = evidence
    if risk_type == RISK_SECRETS_GIT_PUSH:
        payload["classification"] = "destructive"
        payload["operation"] = "secrets_git_push"
    elif risk_type == RISK_SECRET_READ:
        payload["classification"] = "sensitive_read"
    elif risk_type == RISK_SECRET_WRITE:
        payload["classification"] = "write"


def write_envelope(event_type: str, payload: dict[str, Any], suffix: str) -> Path:
    out_dir = inject_dir()
    out_dir.mkdir(parents=True, exist_ok=True)
    corr = payload.get("correlation_id") or payload.get("tool_use_id") or "x"
    safe = re.sub(r"[^a-zA-Z0-9._-]+", "_", str(corr))[:64]
    name = f"hook-{time.time_ns()}-{safe}-{suffix}.json"
    path = out_dir / name
    tmp = path.with_suffix(".tmp")
    envelope = {"type": event_type, "payload": payload}
    tmp.write_text(json.dumps(envelope, separators=(",", ":")), encoding="utf-8")
    tmp.rename(path)
    return path


def write_inject(payload: dict[str, Any], suffix: str) -> None:
    write_envelope(TYPE_AI_AGENT_ACTION, payload, suffix)


def resolve_cursor_pid() -> int:
    """Best-effort live Cursor PID for process/file ancestry; 1 if unknown."""
    try:
        import subprocess

        out = subprocess.check_output(
            ["ps", "-axo", "pid=,args="],
            text=True,
            stderr=subprocess.DEVNULL,
        )
        for line in out.splitlines():
            line = line.strip()
            if not line:
                continue
            if CURSOR_APP_EXECUTABLE not in line:
                continue
            if "Helper" in line:
                continue
            pid_s = line.split(None, 1)[0]
            return int(pid_s)
    except Exception:  # noqa: BLE001
        pass
    return 1


def synthetic_pid(seed: str) -> int:
    digest = hashlib.sha1(seed.encode("utf-8")).hexdigest()
    return 800_000_000 + (int(digest[:8], 16) % 100_000_000)


def shell_comm(command: str) -> str:
    cmd = (command or "").strip()
    if not cmd:
        return "shell"
    for token in cmd.split():
        if "=" in token and not token.startswith("-"):
            continue
        base = Path(token).name
        return base or "shell"
    return "shell"


def build_host_process_payload(
    *,
    command: str,
    correlation_id: str,
    cursor_pid: int,
) -> dict[str, Any]:
    cmdline = truncate(command, 4000)
    pid = synthetic_pid(f"shell:{correlation_id}:{cmdline}")
    return {
        "pid": pid,
        "ppid": cursor_pid,
        "comm": shell_comm(cmdline),
        "executable": shell_comm(cmdline),
        "cmdline": cmdline,
        "user": os.environ.get("USER") or os.environ.get("USERNAME") or "",
        "source": EVIDENCE_HOST_BRIDGE,
        "correlation_id": correlation_id,
    }


def build_host_file_payload(
    *,
    path: str,
    tool_name: str,
    correlation_id: str,
    cursor_pid: int,
) -> dict[str, Any]:
    op = "write" if tool_name in WRITE_TOOLS_SET else "open"
    return {
        "pid": cursor_pid,
        "ppid": 1,
        "comm": "Cursor",
        "executable": CURSOR_APP_EXECUTABLE,
        "path": path,
        "operation": op,
        "source": EVIDENCE_HOST_BRIDGE,
        "correlation_id": correlation_id,
    }


def file_path_from_input(tool_input: dict[str, Any]) -> str:
    for key in ("path", "target_file", "file_path"):
        val = tool_input.get(key)
        if isinstance(val, str) and val.strip():
            return val.strip()
    return ""


def skill_action_details(tool_name: str, tool_input: dict[str, Any]) -> tuple[str, str, str] | None:
    separated = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", tool_name.strip())
    normalized = re.sub(r"[^a-z0-9]+", "_", separated.lower()).strip("_")
    if normalized in {"skill", "use_skill", "invoke_skill", "run_skill"}:
        for key in ("skill_name", "skill", "name"):
            value = tool_input.get(key)
            if isinstance(value, str) and value.strip():
                return value.strip()[:128], "invoke", "skill_tool_use"

    if tool_name not in READ_TOOLS_SET:
        return None
    path = file_path_from_input(tool_input).replace("\\", "/")
    parts = [part for part in path.split("/") if part]
    if len(parts) < 3 or parts[-1].lower() != "skill.md" or parts[-3].lower() != "skills":
        return None
    source_dir = parts[-4].lower() if len(parts) >= 4 else ""
    source = {
        ".agents": "agents",
        ".claude": "claude",
        ".copilot": "copilot",
        ".cursor": "cursor",
        ".github": "github",
    }.get(source_dir, "unknown")
    return parts[-2][:128], "load", f"skill_definition_read:{source}"


def emit_host_bridge(tool_name: str, tool_input: dict[str, Any], action_payload: dict[str, Any]) -> None:
    corr = str(action_payload.get("correlation_id") or f"hook-{time.time_ns()}")
    cursor_pid = resolve_cursor_pid()

    if tool_name in SHELL_TOOLS_SET:
        command = str(tool_input.get("command") or tool_input.get("cmd") or "").strip()
        if not command:
            command = str(action_payload.get("destination") or "").strip()
        if not command:
            return
        write_envelope(
            TYPE_PROCESS_START,
            build_host_process_payload(
                command=command,
                correlation_id=corr,
                cursor_pid=cursor_pid,
            ),
            "host-proc",
        )
        return

    if tool_name in FILE_HOST_TOOLS:
        path = file_path_from_input(tool_input)
        if not path:
            return
        write_envelope(
            TYPE_FILE_OPEN,
            build_host_file_payload(
                path=path,
                tool_name=tool_name,
                correlation_id=corr,
                cursor_pid=cursor_pid,
            ),
            "host-file",
        )


def build_tool_payload(event: dict[str, Any], *, failed: bool) -> dict[str, Any] | None:
    tool_name = str(event.get("tool_name") or "").strip()
    if not tool_name or is_mcp_tool(tool_name):
        return None

    tool_input = event.get("tool_input")
    if not isinstance(tool_input, dict):
        tool_input = {}

    corr = event.get("tool_use_id")
    corr_str = corr.strip() if isinstance(corr, str) else ""
    if not corr_str:
        corr_str = f"hook-{time.time_ns()}"

    payload: dict[str, Any] = {
        "agent_type": AGENT_TYPE_CURSOR,
        "channel": channel_for(tool_name),
        "tool": tool_name,
        "operation": operation(tool_name),
        "destination": destination(tool_name, tool_input),
        "correlation_id": corr_str,
        "timestamp": utc_now(),
        "decision": "allowed" if not failed else "unknown",
        "classification": classification_for(tool_name),
        "governed": False,
        "evidence": [EVIDENCE_CURSOR_HOOK],
        "confidence": "high",
    }

    sid = conversation_id(event)
    if sid:
        payload["session_id"] = sid
    tid = turn_id(event)
    if tid:
        payload["turn_id"] = tid

    cwd = event.get("cwd")
    if isinstance(cwd, str) and cwd.strip():
        payload["process"] = truncate(cwd, 512)

    duration = event.get("duration")
    if isinstance(duration, (int, float)) and duration >= 0:
        payload["evidence"] = [EVIDENCE_CURSOR_HOOK, f"duration_ms={int(duration)}"]

    skill = skill_action_details(tool_name, tool_input)
    if skill:
        skill_name, skill_operation, skill_evidence = skill
        payload.update(
            {
                "channel": CHANNEL_SKILL,
                "tool": TOOL_SKILL,
                "operation": skill_operation,
                "destination": skill_name,
                "skill_name": skill_name,
                "skill_source": skill_evidence.rsplit(":", 1)[-1]
                if skill_evidence.startswith("skill_definition_read:")
                else "unknown",
                "skill_outcome": "failed" if failed else "succeeded",
                "evidence": [EVIDENCE_CURSOR_HOOK, skill_evidence],
                "confidence": "high" if skill_operation == "invoke" else "medium",
                "classification": "read_only" if skill_operation == "load" else "unknown",
            }
        )

    apply_risk(payload, tool_name, tool_input)
    return payload


def build_prompt_payload(event: dict[str, Any]) -> dict[str, Any]:
    prompt = event.get("prompt")
    prompt_text = prompt if isinstance(prompt, str) else ""
    attachments = event.get("attachments")
    attach_count = len(attachments) if isinstance(attachments, list) else 0

    corr = event.get("generation_id")
    corr_str = corr.strip() if isinstance(corr, str) and corr.strip() else f"prompt-{time.time_ns()}"

    dest_parts = []
    if prompt_text.strip():
        dest_parts.append(full_prompt_text(prompt_text))
    if attach_count:
        dest_parts.append(f"{attach_count} attachment(s)")

    payload: dict[str, Any] = {
        "agent_type": AGENT_TYPE_CURSOR,
        "channel": CHANNEL_UNKNOWN,
        "tool": TOOL_PROMPT,
        "operation": OPERATION_PROMPT_SUBMIT,
        "destination": "\n\n".join(dest_parts) if dest_parts else "user prompt",
        "correlation_id": corr_str,
        "timestamp": utc_now(),
        "decision": "allowed",
        "classification": "unknown",
        "governed": False,
        "evidence": [EVIDENCE_CURSOR_HOOK, "prompt_submit"],
        "confidence": "high",
        "prompt_text": full_prompt_text(prompt_text) if prompt_text.strip() else "",
    }

    sid = conversation_id(event)
    if sid:
        payload["session_id"] = sid
    tid = turn_id(event)
    if tid:
        payload["turn_id"] = tid

    return payload


def write_active_turn(event: dict[str, Any]) -> None:
    """Record the current Cursor turn so the MCP gateway can correlate tool calls."""
    sid = conversation_id(event)
    tid = turn_id(event)
    if not sid or not tid:
        return
    out_dir = inject_dir()
    out_dir.mkdir(parents=True, exist_ok=True)
    path = out_dir / ACTIVE_TURN_FILENAME
    tmp = path.with_suffix(".tmp")
    body = json.dumps(
        {
            "session_id": sid,
            "turn_id": tid,
            "timestamp": utc_now(),
        },
        separators=(",", ":"),
    )
    tmp.write_text(body, encoding="utf-8")
    tmp.rename(path)


def handle_event(event: dict[str, Any]) -> None:
    hook = str(event.get("hook_event_name") or "").strip()

    if hook == HOOK_BEFORE_SUBMIT_PROMPT:
        write_active_turn(event)
        write_inject(build_prompt_payload(event), "prompt")
        return

    if hook in (HOOK_POST_TOOL_USE, HOOK_POST_TOOL_USE_FAILURE):
        failed = hook == HOOK_POST_TOOL_USE_FAILURE
        payload = build_tool_payload(event, failed=failed)
        if not payload:
            return
        write_inject(payload, "tool-fail" if failed else "tool")
        tool_name = str(event.get("tool_name") or "").strip()
        tool_input = event.get("tool_input") if isinstance(event.get("tool_input"), dict) else {}
        if not payload.get("skill_name"):
            emit_host_bridge(tool_name, tool_input, payload)


def main() -> int:
    try:
        raw = sys.stdin.read()
        if not raw.strip():
            return 0
        event = json.loads(raw)
        if not isinstance(event, dict):
            return 0
        handle_event(event)
        return 0
    except Exception as exc:  # noqa: BLE001 — hook must fail open
        print(f"emit-tool-use: {exc}", file=sys.stderr)
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
