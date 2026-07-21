#!/bin/sh
set -eu

action_path=${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}
root=${GCC_INPUT_ROOT:-.}
format=${GCC_INPUT_FORMAT:-text}
version=${GCC_INPUT_VERSION:-0.1.0}

binary=$($action_path/scripts/install.sh "$version")
"$binary" check --format "$format" --root "$root"
