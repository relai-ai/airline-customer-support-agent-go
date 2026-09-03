#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
RELAI_DIR="$ROOT_DIR/.relai"
SIM_DIR="$ROOT_DIR/.relai/simulator"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "RELAI Go simulator dependency check requires $1, but it was not found on PATH." >&2
    exit 1
  fi
}

if ! command -v go >/dev/null 2>&1 && [ -x /usr/local/go/bin/go ]; then
  export PATH="/usr/local/go/bin:$PATH"
fi

require_command go

if [ ! -f "$RELAI_DIR/go.mod" ]; then
  echo "RELAI Go simulator requires go.mod in $RELAI_DIR. Run .relai/simulator/install.sh" >&2
  exit 1
fi

if ! grep -q 'github.com/relai-ai/relai-cli/sdk/go' "$RELAI_DIR/go.mod"; then
  echo "RELAI Go simulator go.mod is missing the RELAI Go SDK. Run .relai/simulator/install.sh" >&2
  exit 1
fi

GOFMT_FILES=(
  "$SIM_DIR/src/relai_simulator/adapter_contract.go"
  "$SIM_DIR/src/relai_simulator/adapter.go"
  "$SIM_DIR/src/relai_simulator/runner.go"
)
if [ -f "$SIM_DIR/smoke_learning_env.go" ]; then
  GOFMT_FILES+=("$SIM_DIR/smoke_learning_env.go")
fi
if [ -d "$ROOT_DIR/.relai/learning-envs" ]; then
  while IFS= read -r -d '' file; do
    GOFMT_FILES+=("$file")
  done < <(find "$ROOT_DIR/.relai/learning-envs" -name '*.go' -print0)
fi

test "$(gofmt -l "${GOFMT_FILES[@]}")" = ""

if [ -f "$SIM_DIR/smoke_learning_env.go" ]; then
  "$SIM_DIR/metadata.sh" --learning-env "$SIM_DIR/smoke_learning_env.go" >/dev/null
fi
