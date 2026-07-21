#!/bin/sh
set -eu

version=${1:?usage: package-release.sh VERSION OUTPUT_DIR}
output_dir=${2:?usage: package-release.sh VERSION OUTPUT_DIR}
version=${version#v}
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

case "$output_dir" in
  /*) ;;
  *) output_dir=$project_root/$output_dir ;;
esac
mkdir -p "$output_dir"
: > "$output_dir/checksums.txt"

targets=${GCC_TARGETS:-"linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"}
for target in $targets; do
  os=${target%/*}
  arch=${target#*/}
  name="gha-concurrency-cycle_${version}_${os}_${arch}"
  staging=$(mktemp -d)
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
    -trimpath -ldflags "-s -w -X main.version=$version" \
    -o "$staging/gha-concurrency-cycle" "$project_root/cmd/gha-concurrency-cycle"
  tar -C "$staging" -czf "$output_dir/$name.tar.gz" gha-concurrency-cycle
  rm -rf "$staging"
done

for archive in "$output_dir"/gha-concurrency-cycle_*.tar.gz; do
  if command -v sha256sum >/dev/null 2>&1; then
    digest=$(sha256sum "$archive" | awk '{ print $1 }')
  else
    digest=$(shasum -a 256 "$archive" | awk '{ print $1 }')
  fi
  printf '%s  %s\n' "$digest" "$(basename "$archive")" >> "$output_dir/checksums.txt"
done
