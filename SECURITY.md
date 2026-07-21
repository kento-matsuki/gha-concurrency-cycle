# Security policy

## Supported versions

The project is not published yet. Security fixes target the current development branch.

## Reporting

Do not place private workflow content, secrets, tokens, repository names, or deployment details in a public report. A private reporting broker is not available yet; until one is documented, provide only a minimal synthetic reproducer in public and do not disclose a secret.

## Security boundary

`gha-concurrency-cycle` is designed to run locally without network access or telemetry. It must not follow a workflow reference outside the selected repository root or modify workflow files. Treat unexpected file reads, path traversal, secret disclosure, or workflow modification as security bugs.

The CLI canonicalizes the explicitly selected root, rejects symbolic links at the internal `.github`, `.github/workflows`, and workflow-file boundaries, and rechecks containment before opening each file. Concurrent mutation or replacement of the scanned directory tree by another process is outside the supported threat model; run the check against a stable checkout.
