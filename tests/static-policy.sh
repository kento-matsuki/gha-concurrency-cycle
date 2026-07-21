#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

grep -q '^                                 Apache License$' LICENSE
grep -q '^                           Version 2.0, January 2004$' LICENSE
jq -e '
  .schemaVersion == 1 and
  .candidateId == "20260717T120911Z-5f11" and
  .owner == "kento-matsuki" and
  .author == "@kento-matsuki" and
  (.createdBy | test("Matsuki Kento") and test("@kento-matsuki") and test("automated AI agent"; "i")) and
  .automatedAgent == true and
  .project == "gha-concurrency-cycle"
' .kento-oss.json >/dev/null

actual_modules=$(go list -m -f '{{if not .Main}}{{.Path}} {{.Version}}{{end}}' all | sed '/^$/d' | sort)
expected_modules=$(printf '%s\n' \
  'gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405' \
  'gopkg.in/yaml.v3 v3.0.1' | sort)
[ "$actual_modules" = "$expected_modules" ]

go mod download gopkg.in/check.v1 gopkg.in/yaml.v3
check_dir=$(go list -m -f '{{.Dir}}' gopkg.in/check.v1)
yaml_dir=$(go list -m -f '{{.Dir}}' gopkg.in/yaml.v3)
grep -q 'Redistribution and use in source and binary forms' "$check_dir/LICENSE"
grep -q 'MIT and Apache' "$yaml_dir/LICENSE"

if git grep -I -n -E \
  '(BEGIN [A-Z ]*PRIVATE KEY|github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|AKIA[A-Z0-9]{16})' \
  -- . ':!tests/static-policy.sh'; then
  echo 'secret-like material found in tracked files' >&2
  exit 1
fi
