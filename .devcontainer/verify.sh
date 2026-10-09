#!/usr/bin/env bash
# Environment smoke test only: no login, credentials, or model calls.
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$HOME/.local/bin:$PATH"
for tool in go node python3 docker codex relai uv; do
  command -v "$tool" >/dev/null || { printf 'Missing tool: %s\n' "$tool" >&2; exit 1; }
done
# Docker-in-Docker may need a moment after the container starts.
ready=false
for attempt in {1..30}; do
  if docker info >/dev/null 2>&1; then ready=true; break; fi
  sleep 2
done
if [[ "$ready" != true ]]; then
  printf 'Docker did not become ready within 60 seconds. Inspect container startup logs.\n' >&2
  exit 1
fi
go build ./...
go vet ./...
go test ./...
printf '\nWorkspace checks passed at revision '
git rev-parse HEAD
printf 'Authentication and the live RELAI optimization demo require separate verification.\n'
