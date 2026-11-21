#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "$repo_root/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$repo_root/.env"
  set +a
fi

: "${UMT_CLI_PATH:?UMT_CLI_PATH is not set. Add it to $repo_root/.env (UMT_CLI_PATH=/path/to/UndertaleModCli)}"
: "${HERO_SIEGE_DATA_WIN_PATH:?HERO_SIEGE_DATA_WIN_PATH is not set. Add it to $repo_root/.env (HERO_SIEGE_DATA_WIN_PATH=/path/to/data.win)}"
: "${HERO_SIEGE_BACKEND_URL:?HERO_SIEGE_BACKEND_URL is not set. Add it to $repo_root/.env (HERO_SIEGE_BACKEND_URL=http://localhost:8080/)}"

output="${HERO_SIEGE_DATA_WIN_PATH%.win}_patched.win"

"$UMT_CLI_PATH" load "$HERO_SIEGE_DATA_WIN_PATH" \
  -s "$repo_root/scripts/umt_patch_url.csx" \
  -o "$output"

echo "Patched data.win written to: $output"
