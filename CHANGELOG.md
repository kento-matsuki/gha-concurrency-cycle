# Changelog

## Unreleased

- Mirror the publisher's tracked-file, payload-size, credential-path, and credential-content limits in the local release gate.

All notable changes to this project will be documented here.

## Unreleased

- Add the initial `check` CLI.
- Detect `GCC001` for a same-repository caller/callee effective concurrency-group collision.
- Add conflict and caller-only safe fixtures with automated tests.
- Cover distinct literal groups, dynamic unknowns, malformed YAML, path escape, symbolic links, and graph cycles.
- Add deterministic fan-out text/JSON golden tests and bounded workflow count/file size checks.
- Add checksum-verified release archives and a composite GitHub Action with a local Linux smoke test.
- Add one CI release gate for race tests, dependency licenses, secret patterns, and a timed clean-checkout quickstart.
- Canonicalize an explicitly selected root and reject internal workflow-directory symbolic links before reading content.
- Add a broker-v2 publication contract, immutable CI Action pins, and a checksum-verified self-contained publisher gate.
