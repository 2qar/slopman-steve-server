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

output="$repo_root/.dump"
mkdir -p "$output"

"$UMT_CLI_PATH" dump "$HERO_SIEGE_DATA_WIN_PATH" -c UMT_DUMP_ALL -o "$output"
