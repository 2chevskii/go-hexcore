# Contributing

Open normal pull requests against `develop`. Check existing
[issues](https://github.com/2chevskii/go-hexcore/issues) before starting work.
Follow the [Code of Conduct](CODE_OF_CONDUCT.md); use the
[security policy](SECURITY.md) for private vulnerability reports.

## Run the checks

Use Go 1.22+. The default tests use local HTTP servers and need no credentials.
Disable live tests if they were enabled in your shell:

Bash:

```sh
export HEXCORE_LIVE=0 HEXCORE_LIVE_WRITES=0
```

PowerShell:

```powershell
$env:HEXCORE_LIVE = '0'
$env:HEXCORE_LIVE_WRITES = '0'
```

```sh
go fmt ./...
go mod tidy
go build ./...
go vet ./...
go test -count=1 ./...
go test -race -count=1 ./...
```

Race checks need CGO and a C compiler. CI runs them on Linux; mention in your PR
if you could not run them locally.

For changes to workflows or release scripts, also run these with Node 24:

```sh
node --test .github/scripts/*.test.mjs
actionlint
```

## Write a pull request

Use a focused branch such as `fix/cancellation` or `docs/examples`.
Give the PR a Conventional Commit title, for example
`fix(client): preserve cancellation errors`. For breaking changes, add `!` and
explain the migration. Keep any `BREAKING CHANGE:` footer in the squash message.

Describe the change and how you checked it. Add tests for behavior changes and
update the affected docs. For API changes, link the
[HexCore documentation](https://dev.hexcore.ru/) or include sanitized responses.
Keep explicit errors, context cancellation, and caller-controlled waiting and
retries intact.

Normal PRs are squash-merged into `develop`. Release, hotfix, and synchronization
PRs follow [Releasing](RELEASING.md). Direct pushes to `develop` and `master`
are prohibited.

## Edit the docs

Use Node 24 and npm from the repository root:

```sh
npm ci
npm run docs:dev
```

Before opening a PR, build and preview:

```sh
npm run docs:build
npm run docs:preview
```

Open `http://localhost:4173/go-hexcore/`. The build checks internal links.
Check navigation, search, code copying, and light/dark appearance on desktop
and mobile.

Keep Go examples in `docs/examples/` and include the relevant regions in
Markdown. `go test ./...` compiles these sources without running the examples.
Do not commit dependencies, cache, or generated site output.

## Test against HexCore

Follow [Live checks](LIVE_CHECKS.md) only with permission to use the account.
Writes cost money, and disk cleanup requires the control panel. Keep credentials,
passwords, and account data out of commits, fixtures, and issue reports.

Contributions are licensed under the [MIT License](../LICENSE).
