#!/bin/sh
set -eu

workflow=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)/.github/workflows/release.yml
grep -Fq 'types: [published]' "$workflow"
grep -Fq 'workflow_dispatch:' "$workflow"
grep -Fq 'tagName:' "$workflow"
grep -Fq 'required: true' "$workflow"
grep -Fq 'contents: write' "$workflow"
[ "$(grep -Ec 'uses: [^ ]+@[0-9a-f]{40}([[:space:]]|$)' "$workflow")" -eq 2 ]
! grep -Eq 'uses: [^ ]+@(main|master|v[0-9]+)([[:space:]]|$)' "$workflow"
grep -Fq 'gh release upload "$TAG_NAME"' "$workflow"
grep -Fq 'dist/SHA256SUMS' "$workflow"
grep -Fq -- '--clobber' "$workflow"
