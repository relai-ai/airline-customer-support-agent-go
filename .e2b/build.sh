#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# CLI authentication remains on the host. No credentials enter the template.
npx --yes @e2b/cli@2.21.2 template create relai-airline-dev \
  --dockerfile .e2b/Dockerfile --cpu-count 2 --memory-mb 4096 \
  --cmd /usr/local/bin/start-playground \
  --ready-cmd 'docker info >/dev/null'
