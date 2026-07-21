#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
smoke_root=$(mktemp -d)
trap 'rm -rf "$smoke_root"' EXIT HUP INT TERM

common_env="GITHUB_ACTION_PATH=$project_root RUNNER_TEMP=$smoke_root/run GHA_CONCURRENCY_CYCLE_ASSET_DIR=$smoke_root/nonexistent-assets"

safe_output=$(env $common_env \
  GCC_INPUT_ROOT="$project_root/testdata/safe-caller-only" \
  GCC_INPUT_FORMAT=json \
  "$project_root/scripts/action.sh")
printf '%s' "$safe_output" | grep -q '"diagnostics": \[\]'

set +e
conflict_output=$(env $common_env \
  GCC_INPUT_ROOT="$project_root/testdata/conflict-basic" \
  GCC_INPUT_FORMAT=text \
  "$project_root/scripts/action.sh" 2>&1)
conflict_status=$?
set -e
[ "$conflict_status" -eq 1 ]
printf '%s' "$conflict_output" | grep -q '^GCC001 '

! grep -q 'scripts/install.sh' "$project_root/scripts/action.sh"
! grep -q 'GCC_INPUT_VERSION' "$project_root/scripts/action.sh"
! find "$smoke_root/run" -maxdepth 1 -name 'gha-concurrency-cycle-source.*' -print -quit | grep -q .
