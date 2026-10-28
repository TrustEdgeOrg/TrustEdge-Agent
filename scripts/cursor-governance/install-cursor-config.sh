#!/usr/bin/env bash
# Install Cursor governance config.
#
#   bash scripts/cursor-governance/install-cursor-config.sh          # default: global hooks + project MCP
#   bash scripts/cursor-governance/install-cursor-config.sh --user   # ~/.cursor/hooks.json only
#   bash scripts/cursor-governance/install-cursor-config.sh --project  # repo .cursor/ hooks + MCP (no user hooks)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GOV="$ROOT/scripts/cursor-governance"
CURSOR_DIR="$ROOT/.cursor"
USER_CURSOR_DIR="${HOME}/.cursor"

MODE="${1:-global}"
case "$MODE" in
  --global | global | "") MODE=global ;;
  --user | user) MODE=user ;;
  --project | project) MODE=project ;;
  *)
    echo "usage: $0 [--global|--user|--project]" >&2
    exit 1
    ;;
esac

inject_dir() {
  if [[ -n "${TRUSTEDGE_INJECT_DIR:-}" ]]; then
    printf '%s' "$TRUSTEDGE_INJECT_DIR"
    return
  fi
  python3 -c 'import sys; sys.path.insert(0, "'"$GOV"'"); from constants import default_inject_dir; print(default_inject_dir())'
}

write_hooks() {
  local dest="$1"
  local hook_cmd="$2"
  mkdir -p "$(dirname "$dest")"
  sed "s|HOOK_COMMAND|$hook_cmd|g" "$GOV/hooks.json.example" > "$dest"
}

install_user_hooks() {
  local hook_cmd
  hook_cmd="$(python3 -c "import sys; sys.path.insert(0, '$GOV'); from constants import hook_command_user; print(hook_command_user())")"
  write_hooks "$USER_CURSOR_DIR/hooks.json" "$hook_cmd"
  echo "  $USER_CURSOR_DIR/hooks.json  (all Cursor workspaces)"
}

install_project_hooks() {
  local hook_cmd
  hook_cmd="$(python3 -c "import sys; sys.path.insert(0, '$GOV'); from constants import hook_command_project; print(hook_command_project())")"
  mkdir -p "$CURSOR_DIR"
  write_hooks "$CURSOR_DIR/hooks.json" "$hook_cmd"
  echo "  $CURSOR_DIR/hooks.json  (TrustEdge-Agent project only)"
}

install_project_mcp() {
  local inject
  inject="$(inject_dir)"
  mkdir -p "$CURSOR_DIR"
  sed \
    -e "s|REPO_ROOT|$ROOT|g" \
    -e "s|INJECT_DIR|$inject|g" \
    "$GOV/mcp.json.example" > "$CURSOR_DIR/mcp.json"
  echo "  $CURSOR_DIR/mcp.json  (inject_dir=$inject)"
}

INJECT="$(inject_dir)"

echo "Installing Cursor governance ($MODE)..."

case "$MODE" in
  global)
    install_user_hooks
    install_project_mcp
    if [[ -f "$CURSOR_DIR/hooks.json" ]]; then
      rm -f "$CURSOR_DIR/hooks.json"
      echo "  removed $CURSOR_DIR/hooks.json (use user hooks to avoid duplicate prompts)"
    fi
    ;;
  user)
    install_user_hooks
    ;;
  project)
    install_project_hooks
    install_project_mcp
    ;;
esac

cat <<EOF

Reload Cursor hooks/MCP (Cmd+Shift+P → Hooks: Reload / MCP: Reload).
Ensure trustedge-agent is running with inject watcher active.
EOF
