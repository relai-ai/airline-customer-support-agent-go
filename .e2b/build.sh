#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
# CLI authentication remains on the host. No credentials enter the template.
project_info=$(npx --yes @e2b/cli@2.21.2 auth info)
if ! printf '%s' "$project_info" | grep -Fq 'playground-dev (2e588861-935b-4f73-a85b-cfc649151eda)'; then
  echo 'Select RELAI playground-dev with e2b auth configure before building.' >&2
  exit 1
fi
npx --yes @e2b/cli@2.21.2 template create relai-airline-dev \
  --dockerfile .e2b/Dockerfile --cpu-count 2 --memory-mb 4096 \
  --cmd /usr/local/bin/start-playground \
  --ready-cmd 'docker info >/dev/null'
