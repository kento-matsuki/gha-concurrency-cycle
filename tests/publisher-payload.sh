#!/usr/bin/env bash
set -euo pipefail

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

mapfile -d '' tracked_files < <(git ls-files -z)
file_count=${#tracked_files[@]}
if ((file_count < 9 || file_count > 200)); then
  echo "tracked file count $file_count is outside 9..200" >&2
  exit 1
fi

has_test=0
total_bytes=0
for path in "${tracked_files[@]}"; do
  case "$path" in
    /*|*\\*|../*|*/../*|*/..)
      echo "unsafe tracked path: $path" >&2
      exit 1
      ;;
  esac
  case "$path" in
    test/*|tests/*|spec/*|*/test/*|*/tests/*|*/spec/*|*/__tests__/*|*.test.*|*.spec.*)
      has_test=1
      ;;
  esac
  if [[ ! -f "$path" || -L "$path" ]]; then
    echo "non-regular tracked path: $path" >&2
    exit 1
  fi
  size=$(stat -c %s -- "$path")
  if ((size > 256 * 1024)); then
    echo "file exceeds 256 KiB: $path" >&2
    exit 1
  fi
  total_bytes=$((total_bytes + size))
done

if ((has_test == 0)); then
  echo 'at least one tracked test file is required' >&2
  exit 1
fi
if ((total_bytes > 3 * 1024 * 1024)); then
  echo "repository payload exceeds 3 MiB: $total_bytes" >&2
  exit 1
fi

if printf '%s\n' "${tracked_files[@]}" | grep -Eiq '(^|/)(\.env($|\.)|id_(rsa|dsa|ecdsa|ed25519)|[^/]+\.(pem|key|p12|pfx)|credentials?\.json|secrets?\.)'; then
  echo 'tracked file path resembles a credential' >&2
  exit 1
fi

printf 'publisher payload preflight: %d files, %d bytes\n' "$file_count" "$total_bytes"
