#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
smoke_root=$(mktemp -d)
trap 'rm -rf "$smoke_root"' EXIT HUP INT TERM

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported smoke-test architecture: $(uname -m)" >&2; exit 2 ;;
esac

GCC_TARGETS="linux/$arch" "$project_root/scripts/package-release.sh" 0.1.0 "$smoke_root/assets"

common_env="GITHUB_ACTION_PATH=$project_root RUNNER_TEMP=$smoke_root/run GHA_CONCURRENCY_CYCLE_ASSET_DIR=$smoke_root/assets GCC_INPUT_VERSION=0.1.0"

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

installed=$smoke_root/run/gha-concurrency-cycle/0.1.0/linux_${arch}/gha-concurrency-cycle
[ "$($installed version)" = "0.1.0" ]

printf 'tampered\n' >> "$smoke_root/assets/gha-concurrency-cycle_0.1.0_linux_${arch}.tar.gz"
set +e
checksum_output=$(RUNNER_TEMP="$smoke_root/tampered" \
  GHA_CONCURRENCY_CYCLE_ASSET_DIR="$smoke_root/assets" \
  "$project_root/scripts/install.sh" 0.1.0 2>&1)
checksum_status=$?
set -e
[ "$checksum_status" -eq 2 ]
printf '%s' "$checksum_output" | grep -q '^checksum mismatch '
