# Contributing to go-hexcore

Read the [README](../README.md), [live verification report](../docs/live-verification.md),
and [Code of Conduct](CODE_OF_CONDUCT.md) before contributing. Search existing
[issues](https://github.com/2chevskii/go-hexcore/issues) and
[discussions](https://github.com/2chevskii/go-hexcore/discussions) before opening one.
Report vulnerabilities through [SECURITY.md](SECURITY.md).

## Development

Use Go 1.22 or later. The library uses only the standard library. The default
tests use local HTTP servers and fixtures; they require no HexCore credentials.

```sh
go fmt ./...
go mod tidy
go build ./...
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

Race detection requires CGO and a supported C compiler. Note in your PR if your
local environment cannot run it; CI runs race checks on Linux.

CI checks formatting and module consistency with Go 1.22, tests Go 1.22 and the
two currently supported Go releases on Linux, and tests current stable Go on
Windows and macOS. It also validates workflow definitions and PR titles.

## Pull requests

Create a focused branch from `master`. Describe the problem, resulting behavior,
and validation in the pull request template. Use a
[Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) PR title,
such as `fix(client): preserve cancellation errors` or
`docs: clarify disk cleanup`. Maintainers use PR titles when squash merging.

Add regression coverage when behavior changes and update the relevant public
documentation. Keep unrelated formatting and generated artifacts out of the PR.

For API changes, cite the relevant [HexCore API documentation](https://dev.hexcore.ru/)
and explain any difference between the documented and observed behavior. Use
synthetic or sanitized fixtures. Preserve context cancellation, explicit HTTP
errors, and the client's lack of implicit retries and resource orchestration.

## Live API checks

Live tests are optional and are excluded from CI. Follow the README's instructions
only when you have a suitable account and permission to use its resources.
Provisioning tests spend account funds, and disk cleanup requires the control
panel. Check the live verification report for unresolved provider behavior first.

Never commit `.env`, tokens, passwords, account data, or unsanitized responses.
Remove resource identifiers and personal data from new fixtures and issue reports.
Describe live checks separately from offline results; mock tests do not establish
live API compatibility.

By contributing, you agree that your contributions are licensed under the
repository's [MIT License](../LICENSE).
