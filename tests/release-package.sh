#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
release_root=$(mktemp -d)
trap 'rm -rf "$release_root"' EXIT HUP INT TERM

"$project_root/scripts/package-release.sh" 0.1.0 "$release_root"

for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
  test -s "$release_root/gha-concurrency-cycle_0.1.0_${target}.tar.gz"
done
[ "$(wc -l < "$release_root/checksums.txt")" -eq 4 ]

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$release_root" && sha256sum -c checksums.txt)
else
  (cd "$release_root" && shasum -a 256 -c checksums.txt)
fi
