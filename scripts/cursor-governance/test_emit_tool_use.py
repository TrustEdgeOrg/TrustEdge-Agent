#!/usr/bin/env python3
"""Unit tests for emit-tool-use.py (run: python3 scripts/cursor-governance/test_emit_tool_use.py)."""

from __future__ import annotations

import json
import os
import tempfile
import unittest
from pathlib import Path

import importlib.util
import sys

HOOK_DIR = Path(__file__).resolve().parent
if str(HOOK_DIR) not in sys.path:
    sys.path.insert(0, str(HOOK_DIR))
_spec = importlib.util.spec_from_file_location(
    "emit_tool_use", HOOK_DIR / "emit-tool-use.py"
)
assert _spec and _spec.loader
emit_tool_use = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(emit_tool_use)


class EmitToolUseTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmpdir = tempfile.TemporaryDirectory()
        os.environ["TRUSTEDGE_INJECT_DIR"] = self._tmpdir.name

    def tearDown(self) -> None:
        self._tmpdir.cleanup()
        os.environ.pop("TRUSTEDGE_INJECT_DIR", None)

    def test_skips_mcp_tools(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "MCP:github/list_issues",
                "tool_input": {},
                "tool_use_id": "abc",
            },
            failed=False,
        )
        self.assertIsNone(payload)

    def test_shell_tool_payload(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Shell",
                "tool_input": {"command": "npm test"},
                "tool_use_id": "call-1",
                "conversation_id": "conv-1",
                "duration": 1200,
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["channel"], "SHELL")
        self.assertEqual(payload["destination"], "npm test")
        self.assertEqual(payload["session_id"], "conv-1")
        self.assertEqual(payload.get("turn_id"), None)
        self.assertFalse(payload["governed"])

    def test_tool_turn_id(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Grep",
                "tool_input": {"pattern": "foo"},
                "tool_use_id": "call-2",
                "conversation_id": "conv-1",
                "generation_id": "gen-1",
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["session_id"], "conv-1")
        self.assertEqual(payload["turn_id"], "gen-1")

    def test_skill_definition_read_emits_metadata_only_usage(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Read",
                "tool_input": {"path": "/Users/me/project/.cursor/skills/review/SKILL.md"},
                "tool_use_id": "skill-read-1",
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["channel"], "SKILL")
        self.assertEqual(payload["tool"], "skill")
        self.assertEqual(payload["operation"], "load")
        self.assertEqual(payload["skill_name"], "review")
        self.assertEqual(payload["skill_source"], "cursor")
        self.assertEqual(payload["skill_outcome"], "succeeded")
        self.assertNotIn("/Users/me", payload["destination"])
        self.assertNotIn("private instructions", json.dumps(payload))

    def test_explicit_skill_tool_emits_invocation(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "UseSkill",
                "tool_input": {"skill_name": "incident-triage"},
                "tool_use_id": "skill-use-1",
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["channel"], "SKILL")
        self.assertEqual(payload["operation"], "invoke")
        self.assertEqual(payload["skill_name"], "incident-triage")
        self.assertEqual(payload["confidence"], "high")

    def test_failed_skill_load_is_not_reported_as_successful(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Read",
                "tool_input": {"path": "/repo/.agents/skills/review/SKILL.md"},
                "tool_use_id": "skill-read-failed",
            },
            failed=True,
        )
        assert payload is not None
        self.assertEqual(payload["operation"], "load")
        self.assertEqual(payload["skill_outcome"], "failed")

    def test_skill_definition_read_does_not_emit_absolute_file_bridge(self) -> None:
        emit_tool_use.handle_event(
            {
                "hook_event_name": "postToolUse",
                "tool_name": "Read",
                "tool_input": {"path": "/Users/me/project/.agents/skills/review/SKILL.md"},
                "tool_use_id": "skill-read-2",
            }
        )
        files = list(Path(self._tmpdir.name).glob("hook-*.json"))
        self.assertEqual(len(files), 1)
        event = json.loads(files[0].read_text(encoding="utf-8"))
        self.assertEqual(event["type"], "ai_agent_action")
        self.assertEqual(event["payload"]["skill_name"], "review")
        self.assertNotIn("/Users/me", json.dumps(event))

    def test_write_inject_file(self) -> None:
        emit_tool_use.write_inject(
            {
                "agent_type": "cursor",
                "channel": "FILE",
                "tool": "Read",
                "correlation_id": "x1",
            },
            "tool",
        )
        files = list(Path(self._tmpdir.name).glob("hook-*.json"))
        self.assertEqual(len(files), 1)
        data = json.loads(files[0].read_text(encoding="utf-8"))
        self.assertEqual(data["type"], "ai_agent_action")
        self.assertEqual(data["payload"]["tool"], "Read")

    def test_prompt_submit(self) -> None:
        payload = emit_tool_use.build_prompt_payload(
            {
                "prompt": "List files in src",
                "generation_id": "gen-9",
                "conversation_id": "conv-9",
                "hook_event_name": "beforeSubmitPrompt",
            }
        )
        self.assertEqual(payload["tool"], "prompt")
        self.assertEqual(payload["operation"], "submit")
        self.assertEqual(payload["turn_id"], "gen-9")
        self.assertIn("List files", payload["destination"])

    def test_write_active_turn(self) -> None:
        emit_tool_use.write_active_turn(
            {
                "conversation_id": "conv-9",
                "generation_id": "gen-9",
            }
        )
        path = Path(self._tmpdir.name) / ".active-turn.json"
        self.assertTrue(path.is_file())
        data = json.loads(path.read_text(encoding="utf-8"))
        self.assertEqual(data["session_id"], "conv-9")
        self.assertEqual(data["turn_id"], "gen-9")

    def test_long_prompt_kept_for_asked_ui(self) -> None:
        long = "Hey — check the README.\n\nAlso run git status." + (" more words." * 40)
        payload = emit_tool_use.build_prompt_payload(
            {
                "prompt": long,
                "generation_id": "gen-long",
                "conversation_id": "conv-long",
            }
        )
        self.assertEqual(payload["destination"], long)
        self.assertEqual(payload["prompt_text"], long)
        self.assertIn("\n\n", payload["destination"])
        self.assertTrue(payload["destination"].startswith("Hey —"))
        self.assertNotIn("…", payload["destination"])

    def test_read_env_marks_high_risk(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Read",
                "tool_input": {"path": "/tmp/trustedge-git-test/.env"},
                "tool_use_id": "read-env",
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["risk"], "high")
        self.assertEqual(payload["risk_type"], "secret_read")
        self.assertEqual(payload["classification"], "sensitive_read")

    def test_git_push_env_marks_high_risk(self) -> None:
        payload = emit_tool_use.build_tool_payload(
            {
                "tool_name": "Shell",
                "tool_input": {
                    "command": "git add .env && git push origin main",
                },
                "tool_use_id": "push-env",
            },
            failed=False,
        )
        assert payload is not None
        self.assertEqual(payload["risk"], "high")
        self.assertEqual(payload["risk_type"], "secrets_git_push")
        self.assertEqual(payload["classification"], "destructive")
        self.assertEqual(payload["decision"], "allowed")

    def test_shell_host_bridge_emits_process_start(self) -> None:
        emit_tool_use.handle_event(
            {
                "hook_event_name": "postToolUse",
                "tool_name": "Shell",
                "tool_input": {"command": "ls -la /tmp"},
                "tool_use_id": "shell-corr-1",
                "conversation_id": "conv-h",
                "generation_id": "gen-h",
            }
        )
        files = list(Path(self._tmpdir.name).glob("hook-*.json"))
        types = {
            f.name: json.loads(f.read_text(encoding="utf-8"))["type"] for f in files
        }
        self.assertIn("ai_agent_action", types.values())
        self.assertIn("process_start", types.values())
        proc = next(
            json.loads(f.read_text(encoding="utf-8"))
            for f in files
            if json.loads(f.read_text(encoding="utf-8"))["type"] == "process_start"
        )
        self.assertEqual(proc["payload"]["source"], "cursor_hook_host_bridge")
        self.assertEqual(proc["payload"]["cmdline"], "ls -la /tmp")
        self.assertEqual(proc["payload"]["correlation_id"], "shell-corr-1")
        self.assertEqual(proc["payload"]["comm"], "ls")
        self.assertTrue(any(n.endswith("-tool.json") for n in types))
        self.assertTrue(any(n.endswith("-host-proc.json") for n in types))

    def test_read_host_bridge_emits_file_open(self) -> None:
        emit_tool_use.handle_event(
            {
                "hook_event_name": "postToolUse",
                "tool_name": "Read",
                "tool_input": {"path": "/Users/me/README.md"},
                "tool_use_id": "read-corr-1",
            }
        )
        files = list(Path(self._tmpdir.name).glob("hook-*-host-file.json"))
        self.assertEqual(len(files), 1)
        data = json.loads(files[0].read_text(encoding="utf-8"))
        self.assertEqual(data["type"], "file_open")
        self.assertEqual(data["payload"]["path"], "/Users/me/README.md")
        self.assertEqual(data["payload"]["operation"], "open")
        self.assertEqual(data["payload"]["source"], "cursor_hook_host_bridge")

    def test_write_host_bridge_uses_file_open_with_write_op(self) -> None:
        emit_tool_use.handle_event(
            {
                "hook_event_name": "postToolUse",
                "tool_name": "StrReplace",
                "tool_input": {"path": "/Users/me/README.md"},
                "tool_use_id": "write-corr-1",
            }
        )
        files = list(Path(self._tmpdir.name).glob("hook-*-host-file.json"))
        self.assertEqual(len(files), 1)
        data = json.loads(files[0].read_text(encoding="utf-8"))
        self.assertEqual(data["type"], "file_open")
        self.assertEqual(data["payload"]["operation"], "write")

    def test_mcp_tools_skip_host_bridge(self) -> None:
        emit_tool_use.handle_event(
            {
                "hook_event_name": "postToolUse",
                "tool_name": "MCP:github/list_issues",
                "tool_input": {},
                "tool_use_id": "mcp-1",
            }
        )
        files = list(Path(self._tmpdir.name).glob("hook-*.json"))
        self.assertEqual(files, [])


if __name__ == "__main__":
    unittest.main()
