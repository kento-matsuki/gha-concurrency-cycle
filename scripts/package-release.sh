#!/bin/sh
set -eu

version=${1:?usage: package-release.sh VERSION OUTPUT_DIR}
output_dir=${2:?usage: package-release.sh VERSION OUTPUT_DIR}
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

case "$version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "invalid release version: $version" >&2; exit 2 ;;
esac

case "$output_dir" in
  /*) ;;
  *) output_dir=$project_root/$output_dir ;;
esac
mkdir -p "$output_dir"
if find "$output_dir" -mindepth 1 -print -quit | grep -q .; then
  echo "output directory must be empty: $output_dir" >&2
  exit 2
fi
: > "$output_dir/SHA256SUMS"

source_date_epoch=${SOURCE_DATE_EPOCH:-0}
case "$source_date_epoch" in
  ''|*[!0-9]*) echo 'SOURCE_DATE_EPOCH must be a non-negative integer' >&2; exit 2 ;;
esac

targets=${GCC_TARGETS:-"linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"}
for target in $targets; do
  os=${target%/*}
  arch=${target#*/}
  name="gha-concurrency-cycle_${version}_${os}_${arch}"
  staging=$(mktemp -d)
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
    -trimpath -buildvcs=false -ldflags "-buildid= -s -w -X main.version=$version" \
    -o "$staging/gha-concurrency-cycle" "$project_root/cmd/gha-concurrency-cycle"
  cp "$project_root/LICENSE" "$staging/LICENSE"
  tar --sort=name --format=ustar --owner=0 --group=0 --numeric-owner \
    --mtime="@$source_date_epoch" -C "$staging" -cf - gha-concurrency-cycle LICENSE |
    gzip -n -9 > "$output_dir/$name.tar.gz"
  rm -rf "$staging"
done

for archive in "$output_dir"/gha-concurrency-cycle_*.tar.gz; do
  if command -v sha256sum >/dev/null 2>&1; then
    digest=$(sha256sum "$archive" | awk '{ print $1 }')
  else
    digest=$(shasum -a 256 "$archive" | awk '{ print $1 }')
  fi
  printf '%s  %s\n' "$digest" "$(basename "$archive")" >> "$output_dir/SHA256SUMS"
done
