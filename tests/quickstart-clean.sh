#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
clean_root=$(mktemp -d)
cleanup() {
  chmod -R u+w "$clean_root" 2>/dev/null || true
  rm -rf "$clean_root"
}
trap cleanup EXIT HUP INT TERM

git -C "$project_root" archive --format=tar HEAD | tar -xf - -C "$clean_root"
mkdir -p "$clean_root/cache" "$clean_root/modcache" "$clean_root/gopath"

started=$(date +%s)
set +e
quickstart_output=$(cd "$clean_root" && \
  GOCACHE="$clean_root/cache" \
  GOMODCACHE="$clean_root/modcache" \
  GOPATH="$clean_root/gopath" \
  GOTOOLCHAIN=local \
  timeout 300 go run ./cmd/gha-concurrency-cycle check --root testdata/conflict-basic)
quickstart_status=$?
set -e
elapsed=$(( $(date +%s) - started ))

[ "$quickstart_status" -eq 1 ]
[ "$elapsed" -le 60 ]
printf '%s' "$quickstart_output" | grep -q '^GCC001 '
printf 'clean quickstart: %ss\n' "$elapsed"
