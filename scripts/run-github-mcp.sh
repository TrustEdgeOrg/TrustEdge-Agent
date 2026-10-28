#!/usr/bin/env bash
# Local GitHub MCP (stdio) for TrustEdge gateway downstream.
# Requires: Docker + GITHUB_PERSONAL_ACCESS_TOKEN in the environment.
set -euo pipefail

if [[ -z "${GITHUB_PERSONAL_ACCESS_TOKEN:-}" ]]; then
  echo "GITHUB_PERSONAL_ACCESS_TOKEN is required for GitHub MCP" >&2
  echo "Create a fine-grained PAT: https://github.com/settings/personal-access-tokens/new" >&2
  exit 1
fi

# stdio MCP: keep -i, no TTY. Token passed from env (do not hardcode).
exec docker run -i --rm \
  -e GITHUB_PERSONAL_ACCESS_TOKEN \
  -e GITHUB_TOOLSETS="${GITHUB_TOOLSETS:-repos,issues,pull_requests}" \
  ghcr.io/github/github-mcp-server
