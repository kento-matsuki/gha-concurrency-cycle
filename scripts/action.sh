#!/bin/sh
set -eu

action_path=${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}
root=${GCC_INPUT_ROOT:-.}
format=${GCC_INPUT_FORMAT:-text}

temp_root=${RUNNER_TEMP:-${TMPDIR:-/tmp}}
mkdir -p "$temp_root"
build_root=$(mktemp -d "$temp_root/gha-concurrency-cycle-source.XXXXXX")
trap 'rm -rf "$build_root"' 0 HUP INT TERM
binary=$build_root/gha-concurrency-cycle

(cd "$action_path" && go build -trimpath -o "$binary" ./cmd/gha-concurrency-cycle)
"$binary" check --format "$format" --root "$root"
