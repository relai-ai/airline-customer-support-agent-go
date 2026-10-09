#!/usr/bin/env bash
set -euo pipefail
# Keep the daemon as the foreground process. E2B's separate readiness check
# waits for Docker to respond; a failed daemon cannot leave a stale ready file.
exec sudo dockerd
