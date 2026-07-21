#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

test -z "$(gofmt -l .)"
go test -race ./...
go vet ./...
tests/static-policy.sh
tests/publisher-contract.sh
tests/publisher-payload.sh
tests/action-smoke.sh
tests/release-package.sh
tests/quickstart-clean.sh
