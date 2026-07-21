#!/bin/sh
set -eu

version=${1:?usage: install.sh VERSION}
version=${version#v}

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 2 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 2 ;;
esac

asset="gha-concurrency-cycle_${version}_${os}_${arch}.tar.gz"
install_root=${RUNNER_TEMP:-${TMPDIR:-/tmp}}/gha-concurrency-cycle/${version}/${os}_${arch}
archive=$install_root/$asset
checksums=$install_root/checksums.txt
mkdir -p "$install_root"

if [ -n "${GHA_CONCURRENCY_CYCLE_ASSET_DIR:-}" ]; then
  cp "$GHA_CONCURRENCY_CYCLE_ASSET_DIR/$asset" "$archive"
  cp "$GHA_CONCURRENCY_CYCLE_ASSET_DIR/checksums.txt" "$checksums"
else
  base_url="https://github.com/kento-matsuki/gha-concurrency-cycle/releases/download/v${version}"
  curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
    "$base_url/$asset" --output "$archive"
  curl --fail --silent --show-error --location --proto '=https' --tlsv1.2 \
    "$base_url/checksums.txt" --output "$checksums"
fi

expected=$(awk -v name="$asset" '$2 == name { print $1 }' "$checksums")
if [ -z "$expected" ]; then
  echo "checksum entry missing for $asset" >&2
  exit 2
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{ print $1 }')
else
  actual=$(shasum -a 256 "$archive" | awk '{ print $1 }')
fi
if [ "$actual" != "$expected" ]; then
  echo "checksum mismatch for $asset" >&2
  exit 2
fi

tar -xzf "$archive" -C "$install_root"
chmod +x "$install_root/gha-concurrency-cycle"
printf '%s\n' "$install_root/gha-concurrency-cycle"
