#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
release_root=$(mktemp -d)
trap 'rm -rf "$release_root"' EXIT HUP INT TERM

SOURCE_DATE_EPOCH=0 "$project_root/scripts/package-release.sh" v0.1.0 "$release_root/first"
SOURCE_DATE_EPOCH=0 "$project_root/scripts/package-release.sh" v0.1.0 "$release_root/second"

for target in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
  archive="gha-concurrency-cycle_v0.1.0_${target}.tar.gz"
  test -s "$release_root/first/$archive"
  cmp "$release_root/first/$archive" "$release_root/second/$archive"
done
cmp "$release_root/first/SHA256SUMS" "$release_root/second/SHA256SUMS"
[ "$(wc -l < "$release_root/first/SHA256SUMS")" -eq 4 ]

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$release_root/first" && sha256sum -c SHA256SUMS)
else
  (cd "$release_root/first" && shasum -a 256 -c SHA256SUMS)
fi
