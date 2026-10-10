#!/usr/bin/env bash
set -euo pipefail
# E2B may restore an already-running daemon from the template snapshot.
# Reuse a healthy daemon and keep the start process tied to its health.
if sudo docker info >/dev/null 2>&1; then
  while sudo docker info >/dev/null 2>&1; do sleep 5; done
  exit 1
fi
# Otherwise keep our daemon in the foreground; readiness checks real Docker.
exec sudo dockerd
