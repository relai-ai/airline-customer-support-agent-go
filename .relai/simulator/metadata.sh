#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
RELAI_DIR="$ROOT_DIR/.relai"
SIM_DIR="$ROOT_DIR/.relai/simulator"
LEARNING_ENV=""
UPLOAD="false"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --learning-env)
      LEARNING_ENV="${2:-}"
      shift 2
      ;;
    --upload)
      UPLOAD="true"
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [ -z "$LEARNING_ENV" ]; then
  echo "Usage: .relai/simulator/metadata.sh --learning-env <learning-env-path>" >&2
  exit 2
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "RELAI Go simulator metadata extraction requires $1, but it was not found on PATH." >&2
    exit 1
  fi
}

if ! command -v go >/dev/null 2>&1 && [ -x /usr/local/go/bin/go ]; then
  export PATH="/usr/local/go/bin:$PATH"
fi

# BEGIN RELAI CLI SDK INSTALL - do not edit
read_relai_config_value() {
  local section="$1"
  local key="$2"
  local config_path="$ROOT_DIR/.relai/config.toml"
  if [ ! -f "$config_path" ]; then
    echo "missing RELAI config at $config_path" >&2
    exit 1
  fi
  awk -v section="$section" -v key="$key" -v path="$config_path" '
    BEGIN { in_section = 0; found = 0 }
    /^[[:space:]]*\[/ {
      in_section = ($0 ~ "^[[:space:]]*\\[" section "\\][[:space:]]*$")
      next
    }
    in_section && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" {
      value = $0
      sub(/^[^=]*=[[:space:]]*/, "", value)
      sub(/[[:space:]]*(#.*)?$/, "", value)
      if (value ~ /^".*"$/) {
        sub(/^"/, "", value)
        sub(/"$/, "", value)
      }
      if (value != "") {
        print value
        found = 1
        exit
      }
    }
    END {
      if (!found) {
        printf "missing %s.%s in %s\n", section, key, path > "/dev/stderr"
        exit 1
      }
    }
  ' "$config_path"
}

read_optional_relai_config_value() {
  local section="$1"
  local key="$2"
  local config_path="$ROOT_DIR/.relai/config.toml"
  if [ ! -f "$config_path" ]; then
    return 0
  fi
  awk -v section="$section" -v key="$key" '
    BEGIN { in_section = 0 }
    /^[[:space:]]*\[/ {
      in_section = ($0 ~ "^[[:space:]]*\\[" section "\\][[:space:]]*$")
      next
    }
    in_section && $0 ~ "^[[:space:]]*" key "[[:space:]]*=" {
      value = $0
      sub(/^[^=]*=[[:space:]]*/, "", value)
      sub(/[[:space:]]*(#.*)?$/, "", value)
      if (value ~ /^".*"$/) {
        sub(/^"/, "", value)
        sub(/"$/, "", value)
      }
      print value
      exit
    }
  ' "$config_path"
}

relai_go_sdk_local_path_configured() {
  [ -n "${RELAI_SDK_LOCAL_PATH:-$(read_optional_relai_config_value local relai_sdk_path)}" ]
}

resolve_relai_go_sdk_local_path() {
  RELAI_SDK_LOCAL_PATH="${RELAI_SDK_LOCAL_PATH:-$(read_optional_relai_config_value local relai_sdk_path)}"
  if [ -z "$RELAI_SDK_LOCAL_PATH" ]; then
    return 1
  fi
  if [ ! -f "$RELAI_SDK_LOCAL_PATH/go.mod" ]; then
    echo "local RELAI Go SDK go.mod not found at $RELAI_SDK_LOCAL_PATH/go.mod" >&2
    exit 1
  fi
  if ! grep -q '^module github.com/relai-ai/relai-cli/sdk/go$' "$RELAI_SDK_LOCAL_PATH/go.mod"; then
    echo "local RELAI Go SDK module must be github.com/relai-ai/relai-cli/sdk/go: $RELAI_SDK_LOCAL_PATH/go.mod" >&2
    exit 1
  fi
  printf '%s\n' "$RELAI_SDK_LOCAL_PATH"
}

resolve_relai_go_sdk_release_path() {
  local sdk_version="$1"
  local sdk_cache_dir="$RELAI_DIR/sdk/go"
  local sdk_dir="$sdk_cache_dir/$sdk_version"
  local archive_path="$sdk_cache_dir/relai-go-sdk-$sdk_version.tar.gz"
  local extract_dir="$sdk_cache_dir/.extract-$sdk_version-$$"
  local api_url
  local api_key
  if [ -f "$sdk_dir/go.mod" ]; then
    printf '%s\n' "$sdk_dir"
    return
  fi
  require_command curl
  require_command tar
  api_url="$(read_relai_config_value api url)"
  api_key="$(read_relai_config_value api key)"
  mkdir -p "$sdk_cache_dir"
  rm -rf "$extract_dir"
  mkdir -p "$extract_dir"
  curl -fsSL \
    -H "Authorization: Token $api_key" \
    "${api_url%/}/sdk/go/files/relai-go-sdk-$sdk_version.tar.gz" \
    -o "$archive_path"
  tar -xzf "$archive_path" -C "$extract_dir"
  if [ ! -f "$extract_dir/relai-go-sdk-$sdk_version/go.mod" ]; then
    rm -rf "$extract_dir"
    echo "RELAI Go SDK archive did not contain relai-go-sdk-$sdk_version/go.mod" >&2
    exit 1
  fi
  rm -rf "$sdk_dir"
  mv "$extract_dir/relai-go-sdk-$sdk_version" "$sdk_dir"
  rm -rf "$extract_dir" "$archive_path"
  printf '%s\n' "$sdk_dir"
}

resolve_relai_go_sdk_path() {
  local sdk_version="$1"
  if relai_go_sdk_local_path_configured; then
    resolve_relai_go_sdk_local_path
  else
    resolve_relai_go_sdk_release_path "$sdk_version"
  fi
}

go_module_replace_path() {
  local module_dir="$1"
  local sdk_path="$2"
  case "$sdk_path" in
    "$module_dir"/*)
      printf './%s\n' "${sdk_path#"$module_dir"/}"
      ;;
    *)
      printf '%s\n' "$sdk_path"
      ;;
  esac
}

ensure_relai_go_sdk_module() {
  local relai_dir="$1"
  local sdk_version="0.1.5"
  if [ ! -f "$relai_dir/go.mod" ]; then
    cat > "$relai_dir/go.mod" <<EOF
module relai-simulator

go 1.24.4

require github.com/relai-ai/relai-cli/sdk/go v$sdk_version
EOF
  fi
  local sdk_path
  sdk_path="$(resolve_relai_go_sdk_path "$sdk_version")"
  go mod edit -modfile="$relai_dir/go.mod" -require=github.com/relai-ai/relai-cli/sdk/go@v"$sdk_version"
  go mod edit -modfile="$relai_dir/go.mod" -replace=github.com/relai-ai/relai-cli/sdk/go="$(go_module_replace_path "$relai_dir" "$sdk_path")"
  (cd "$relai_dir" && go mod tidy)
}

configure_relai_go_sdk_for_temporary_workspace() {
  local temp_dir="$1"
  local sdk_version="0.1.5"
  local sdk_path
  sdk_path="$(resolve_relai_go_sdk_path "$sdk_version")"
  go mod edit -modfile="$temp_dir/go.mod" -replace=github.com/relai-ai/relai-cli/sdk/go="$sdk_path"
}

add_root_module_replace() {
  local temp_dir="$1"
  if [ ! -f "$ROOT_DIR/go.mod" ]; then
    return 0
  fi
  local module_path
  module_path="$(awk '/^module[[:space:]]+/ { print $2; exit }' "$ROOT_DIR/go.mod")"
  if [ -z "$module_path" ]; then
    return 0
  fi
  go mod edit -modfile="$temp_dir/go.mod" -module="$module_path/relai_simulator"
  go mod edit -modfile="$temp_dir/go.mod" -require="$module_path@v0.0.0"
  go mod edit -modfile="$temp_dir/go.mod" -replace="$module_path=$ROOT_DIR"
}

prepare_relai_go_runner_workspace() {
  local temp_dir="$1"
  local learning_env="$2"
  mkdir -p "$temp_dir"
  cp "$RELAI_DIR/go.mod" "$temp_dir/go.mod"
  if [ -f "$RELAI_DIR/go.sum" ]; then
    cp "$RELAI_DIR/go.sum" "$temp_dir/go.sum"
  fi
  configure_relai_go_sdk_for_temporary_workspace "$temp_dir"
  cp "$learning_env" "$temp_dir/environment.go"
  cp "$SIM_DIR/src/relai_simulator/adapter_contract.go" "$temp_dir/adapter_contract.go"
  cp "$SIM_DIR/src/relai_simulator/adapter.go" "$temp_dir/adapter.go"
  cp "$SIM_DIR/src/relai_simulator/runner.go" "$temp_dir/runner.go"
  add_root_module_replace "$temp_dir"
  prepare_relai_go_global_evaluators "$temp_dir"
  (cd "$temp_dir" && go mod tidy)
}

prepare_relai_go_global_evaluators() {
  local temp_dir="$1"
  local evaluator_dir="$RELAI_DIR/evaluators"
  local module_path
  module_path="$(cd "$temp_dir" && go list -m -f '{{.Path}}')"
  local wrapper_path="$temp_dir/global_evaluators.go"
  local index=0
  local evaluator
  local evaluator_package

  {
    printf '%s\n' 'package main'
    printf '%s\n' ''
    printf '%s\n' 'import ('
    printf '%s\n' '  relai "github.com/relai-ai/relai-cli/sdk/go"'
    if [ -d "$evaluator_dir" ]; then
      for evaluator in "$evaluator_dir"/*.go; do
        [ -f "$evaluator" ] || continue
        case "$evaluator" in
          *_test.go) continue ;;
        esac
        evaluator_package="$temp_dir/global_evaluators/evaluator_$index"
        mkdir -p "$evaluator_package"
        cp "$evaluator" "$evaluator_package/$(basename "$evaluator")"
        printf '  evaluator_%s "%s"\n' "$index" "$module_path/global_evaluators/evaluator_$index"
        index=$((index + 1))
      done
    fi
    printf '%s\n' ')'
    printf '%s\n' ''
    printf '%s\n' 'func loadGlobalEvaluators() []relai.Evaluator {'
    if [ "$index" -eq 0 ]; then
      printf '%s\n' '  return nil'
    else
      printf '%s\n' '  return []relai.Evaluator{'
      local evaluator_index
      for ((evaluator_index = 0; evaluator_index < index; evaluator_index++)); do
        printf '    evaluator_%s.BuildEvaluator(),\n' "$evaluator_index"
      done
      printf '%s\n' '  }'
    fi
    printf '%s\n' '}'
  } > "$wrapper_path"
}

prepare_relai_go_metadata_workspace() {
  local temp_dir="$1"
  local learning_env="$2"
  mkdir -p "$temp_dir"
  cp "$RELAI_DIR/go.mod" "$temp_dir/go.mod"
  if [ -f "$RELAI_DIR/go.sum" ]; then
    cp "$RELAI_DIR/go.sum" "$temp_dir/go.sum"
  fi
  configure_relai_go_sdk_for_temporary_workspace "$temp_dir"
  cp "$learning_env" "$temp_dir/environment.go"
  add_root_module_replace "$temp_dir"
}

# END RELAI CLI SDK INSTALL

require_command go
export GOWORK=off

TMP_ROOT="$RELAI_DIR/tmp/go-metadata"
rm -rf "$TMP_ROOT"
case "$LEARNING_ENV" in
  /*) LEARNING_ENV_PATH="$LEARNING_ENV" ;;
  *) LEARNING_ENV_PATH="$ROOT_DIR/$LEARNING_ENV" ;;
esac
prepare_relai_go_metadata_workspace "$TMP_ROOT" "$LEARNING_ENV_PATH"

cat > "$TMP_ROOT/main.go" <<'GO'
package main

import (
	"encoding/json"
	"fmt"
	"os"

	relai "github.com/relai-ai/relai-cli/sdk/go"
)

func main() {
	environment := BuildEnvironment()
	var payload map[string]any
	if os.Getenv("RELAI_GO_UPLOAD_METADATA") == "1" {
		payload = relai.EnvironmentUploadMetadata(environment)
	} else {
		payload = relai.EnvironmentMetadata(environment)
	}
	if err := json.NewEncoder(os.Stdout).Encode(payload); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
GO

if [ "$UPLOAD" = "true" ]; then
  export RELAI_GO_UPLOAD_METADATA=1
fi

(
  cd "$TMP_ROOT"
  go run .
)
