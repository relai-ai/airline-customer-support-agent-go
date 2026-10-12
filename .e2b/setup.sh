#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Install the playground tools only. Never authenticate or call models in a prebuild.
mkdir -p "$HOME/.local/bin" "$HOME/.local/share/relai-demo"
export PATH="$HOME/.local/bin:$PATH"
npm install --global --prefix "$HOME/.local" @openai/codex@0.156.0
python3 -m venv "$HOME/.local/share/relai-demo/uv"
"$HOME/.local/share/relai-demo/uv/bin/pip" install uv==0.8.22
ln -sf "$HOME/.local/share/relai-demo/uv/bin/uv" "$HOME/.local/bin/uv"

installer=$(mktemp)
trap 'rm -f "$installer"' EXIT
curl --fail --silent --show-error --location https://api.relai.ai/install.sh -o "$installer"
printf '%s  %s\n' fd95daad8ef085b8bc2d8ffbd085da9f4ff5b244aa632204bb66df59f82f8069 "$installer" | sha256sum --check --status
RELAI_INSTALL_COMPLETIONS=0 bash "$installer"
relai plugins install --agent codex

# Make tools available to new interactive shells as well as this hook.
path_line='export PATH="$HOME/.local/bin:$PATH"'
for profile in "$HOME/.bashrc" "$HOME/.profile"; do
  touch "$profile"
  grep -Fqx "$path_line" "$profile" || printf '\n%s\n' "$path_line" >> "$profile"
done

go mod download
go build -o /tmp/relai-airline-demo ./cmd/airline-support
{
  go version
  node --version
  codex --version
  relai --version
  uv --version
} > "$HOME/.local/share/relai-demo/tool-versions.txt"
printf '\nEnvironment installed. Read .e2b/README.md for the playground runtime.\n'
