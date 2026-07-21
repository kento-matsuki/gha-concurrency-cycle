# Contributing

Thank you for helping keep reusable-workflow concurrency failures predictable.

Before submitting a change, add or update a synthetic fixture and run:

```sh
gofmt -w .
go test ./...
go vet ./...
```

Keep the scope narrow: same-repository reusable workflows and statically resolvable workflow-level concurrency groups. Do not include private workflow files, logs, secrets, or copied production configurations in reports or fixtures.

By contributing, you agree that your contribution is licensed under Apache-2.0.
