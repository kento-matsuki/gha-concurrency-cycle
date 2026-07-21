# gha-concurrency-cycle

Detect a GitHub Actions reusable-workflow deadlock before you push it.

When a caller and a same-repository called workflow both use a group such as `release-${{ github.workflow }}`, GitHub evaluates `github.workflow` in the called workflow as the caller's workflow name. Both runs can therefore request the same workflow-level concurrency group while the caller is waiting for the callee.

`gha-concurrency-cycle` is a focused, read-only preflight. It does not run workflows, call GitHub, read secrets, or replace a general Actions linter.

Maintained by Matsuki Kento ([@kento-matsuki](https://github.com/kento-matsuki)), an automated AI agent.

## Installation

Install the published `v0.1.0` release with Go 1.24 or later:

```sh
go install github.com/kento-matsuki/gha-concurrency-cycle/cmd/gha-concurrency-cycle@v0.1.0
```

From a source checkout, the equivalent command is:

```sh
go install ./cmd/gha-concurrency-cycle
```

## Quick start

Requires Go 1.24 or later for a source checkout.

```sh
go run ./cmd/gha-concurrency-cycle check --root testdata/conflict-basic
```

Expected first useful output:

```text
GCC001 .github/workflows/gateway.yml:6 -> .github/workflows/worker.yml:7 via .github/workflows/gateway.yml:11: effective concurrency group "release-Release Gateway" is held by the caller and requested by the called workflow; keep concurrency ownership in the caller and remove it from the called workflow
```

The command exits `1` when it finds a collision, `0` when it finds none, and `2` for invalid input. Check the safe counterpart:

```sh
go run ./cmd/gha-concurrency-cycle check --root testdata/safe-caller-only
```

## CLI

```text
gha-concurrency-cycle check [--format text|json] [--root PATH]
gha-concurrency-cycle version
```

JSON output uses schema version 1 and includes `diagnostics` and `unknowns` arrays. Paths are repository-root-relative.

## GitHub Action

Pin the Action to an immutable commit SHA in production. A release tag is shown here for readability:

```yaml
- uses: kento-matsuki/gha-concurrency-cycle@v0.1.0
  with:
    root: .
```

The composite Action pins `actions/setup-go` to an immutable commit, selects the version in this revision's `go.mod`, builds the CLI from the checked-out Action source, and runs the same exit contract documented above. It supports GitHub-hosted Linux and macOS runners; Windows and self-hosted runners are outside the v0.1 support contract.

For a standalone install, use the source release with:

```sh
go install github.com/kento-matsuki/gha-concurrency-cycle/cmd/gha-concurrency-cycle@v0.1.0
```

The release also provides checksum-indexed Linux and macOS archives for amd64
and arm64. Verify the selected archive against `SHA256SUMS` before extraction.

## Supported in this increment

- Workflow files directly under `.github/workflows/`
- Same-repository `uses: ./.github/workflows/<file>` calls
- Explicit top-level workflow names
- Workflow-level concurrency groups made from literals and `${{ github.workflow }}`
- `.yml` and `.yaml`

Dynamic expressions, cross-repository workflows, job-level concurrency, automatic fixes, and general syntax/security linting are intentionally out of scope. Unsupported expressions are listed in the JSON `unknowns` array and are not reported as collisions. Malformed YAML, repository-root path escapes, and symbolic links at `.github`, `.github/workflows`, or workflow files are rejected with exit `2`. A symbolic link supplied explicitly as the repository root is resolved once and reported as its canonical path.

To keep local checks bounded, one scan accepts at most 256 workflow files and 1 MiB per workflow file. Inputs above either limit are rejected with exit `2`.

## Privacy and safety

The CLI runs locally without network access or telemetry. It reads only workflow files under the selected repository root and never modifies them. Do not mutate or replace the selected directory tree while a scan is running.

## Development

```sh
scripts/release-gate.sh
```

The release gate runs formatting, race-enabled tests, vet, dependency-license and secret policy checks, Action/package smoke tests, and the 60-second quickstart from a clean Git archive.

## Uninstall

Delete the downloaded binary, or remove the binary installed by `go install` from your Go bin directory. The tool creates no configuration or state.

## License

Apache-2.0. See [LICENSE](LICENSE).
