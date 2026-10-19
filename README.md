# go-hexcore

[![CI](https://github.com/2chevskii/go-hexcore/actions/workflows/ci.yml/badge.svg)](https://github.com/2chevskii/go-hexcore/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go 1.22+ client for the [HexCore HTTP API](https://dev.hexcore.ru/), intended
for use by a future Terraform provider. It uses only the Go standard library.

Phase 1 is implemented and covered by offline contract tests. Phase 2 has
verified reads, standard/custom VDS provisioning and deletion, custom memory
units, and power actions. Graceful reboot, password reset, and disk attachment
still have unresolved completion/status behavior; full write compatibility is
not established. See the
[live verification report](docs/live-verification.md) for evidence and blockers.

```sh
go get github.com/2chevskii/go-hexcore
```

Normal contributions target `develop`; `master` contains released code. Releases
start at v1.0.0 and follow Semantic Versioning. See the
[release process](.github/RELEASING.md) and [changelog](CHANGELOG.md).

## Usage

```go
import (
    "context"
    "os"
    "time"

    hexcore "github.com/2chevskii/go-hexcore"
)

client, err := hexcore.NewClient(hexcore.Config{
    Token: os.Getenv("HEXCORE_TOKEN"), // raw token, without "Bearer "
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

servers, err := client.VDS.List(ctx)
if err != nil {
    return err
}
_ = servers
```

`Config.BaseURL` defaults to `https://api.hexcore.ru/`; a custom URL can include
a path prefix. `Config.HTTPClient` injects a caller-owned client and preserves
its timeout and transport. Otherwise, requests have a 30-second timeout.
`Config.UserAgent` defaults to `go-hexcore` and can identify the consuming provider.
Configuration is validated locally; construction makes no API call. A client
supports concurrent use if its services and HTTP client are not mutated.

## Services

| Service | Methods |
| --- | --- |
| `Account` | `Get` |
| `Actions` | `Get`, `List`, `Wait` |
| `VDS` | `Get`, `List`, `Create`, `CreateCustom`, `Reboot`, `HardReboot`, `Shutdown`, `HardShutdown`, `PowerOn`, `Destroy`, `ResetPassword` |
| `Disks` | `Get`, `List`, `Create`, `Plug`, `Unplug`, `Resize` |
| `Regions`, `Sizes`, `Images`, `Applications` | `List` on each service |

All methods accept `context.Context` first. Get/create methods return `(*T,
error)`, list methods return `([]T, error)`, and `VDS.Destroy` returns `error`.
Lists return non-nil empty slices for empty arrays. The client checks the
documented success status and response envelope. Singular responses accept a
resource object or a one-item array; lists require arrays. Malformed or
unexpected responses are errors. Unknown resource properties are ignored.

Resource IDs are strings. Quantities are `int64`, including values encoded as
strings by the server. Monetary values and timestamps preserve their textual
representation: no floating-point conversion or timezone assumption is made.
Optional fields use pointers where endpoint responses differ. In particular,
`VDS.Get` does not fill missing fields by making a second request to `VDS.List`.
Disk attachment status and VDS ID are exposed as `Disk.Status` and `Disk.VDSID`.

Standard creation uses `CreateVDSRequest` with size, image, and region slugs.
`CreateCustomVDSRequest.Size` is a `CustomSize`; the client supplies `size=0` and
`custom_size=1`. Custom memory and disk sizes are in GB; memory is reported in
MB. `CreateDiskRequest.VDSID` optionally requests attachment
during creation; this string must contain a positive decimal ID and
is encoded as a JSON number without loss of precision.

## Asynchronous actions and errors

Creation and action calls return immediately. The API's creation responses
contain a resource ID but no action ID. Resource-readiness checks belong to
the caller. For an operation that returns an action:

```go
action, err := client.VDS.Reboot(ctx, id)
if err != nil {
    return err
}
completed, err := client.Actions.Wait(ctx, action.ID, hexcore.WaitOptions{})
```

Use a context with an appropriate deadline for the entire operation. `Wait`
polls immediately, then every two seconds by default. Set `PollInterval` to
override the interval. It returns on completion, action failure, unknown
status, HTTP failure, or cancellation. An action failure returns the failed
action together with `*hexcore.ActionError`.

Live VDS action calls return `202 Accepted`, while the Swagger examples use
`200 OK`; both are supported for action submissions. Destruction accepts `204`
or `202` and returns on acceptance. The VDS must already be halted. A nil error
does not imply that an asynchronous action or resource deletion has completed.

There are no automatic retries, including for writes, throttling, or transport
errors. Inspect HTTP failures with `errors.As`:

```go
var apiErr *hexcore.APIError
if errors.As(err, &apiErr) {
    status := apiErr.StatusCode
    retryAfter := apiErr.Headers.Get("Retry-After")
    _, _ = status, retryAfter
}
```

`APIError` includes status, message, method, relative path, and response headers.
Messages can appear at the top level or inside the requested resource envelope.
Non-JSON error bodies fall back to the HTTP status text. Request bodies and
authorization headers are not included in diagnostics. Cancellation and
transport errors remain discoverable through `errors.Is`/`errors.As`.

## Documentation gaps

The client is handwritten against the [Swagger 2.0 document](https://dev.hexcore.ru/hexcore.json?v=0.8.7).
The URL revision is `0.8.7`, while `info.version` is `0.6.2`. Saved success
examples in `testdata/responses.json` were retrieved on 2026-10-19. Tests retain
example inconsistencies, such as the unplug response reporting type `plug`.
Sanitized live response fixtures are in `testdata/live-responses.json`; account
identifiers, email, dates, and monetary values are replaced with synthetic data.

- Resource routes use parameterized IDs from the prose, rather than literal
  example IDs in the Swagger path keys. This includes `/actions/{id}` and
  `/disk/{id}`. Both standard and custom creation use `/vds`, without a space.
- Read responses and action submissions use different array envelopes and
  timestamp names. The client unwraps and normalizes these variants. Live
  `/account` responses use an object where the Swagger example shows an array.
- Empty action history returns `403` with `actions.message` equal to
  `No actions found.`. The empty disk list returns `403` with `disk.message`
  equal to `Can not find disk for your account.`. The SDK preserves these HTTP
  errors. Only the integration harness recognizes these exact list responses;
  it does not generalize them to singular resources or other access failures.
- `Disks.Plug` sends only `{"type":"plug"}`. The documentation does not describe
  a target-VDS parameter. Its target-selection behavior needs live verification.
- Disk deletion is mentioned but has no documented operation. No disk deletion
  method is implemented. Use the control panel for cleanup.
- Live VDS names are truncated to 45 characters by the server. The SDK preserves
  returned names; the live harness uses shorter names for reliable ownership
  reconciliation and cleanup.
- In live checks, disk creation with `vds_id` produced a disk whose attachment
  status and VDS ID were both null. Plug and resize returned `403` with
  `Can not get Disk status.`. The documented attachment workflow remains
  unverified; null attachment fields are preserved rather than interpreted as
  a confirmed state. The control panel also had an empty UUID for this disk,
  which prevented its attachment form from submitting.
- The disk cURL example quotes `size`, but live creation rejects that form with
  `403: Size is not int.`. The SDK sends a JSON integer. Numeric and quoted
  `vds_id` forms both produced null attachment metadata in separate live checks.
- Disk resize requires an unplugged disk and only permits growth. Plugging may
  require a VDS shutdown and start before a disk becomes active. The client does
  not orchestrate these prerequisites.
- Custom memory is requested in GB and reported in MB: a live request of `1`
  returned `1024`. The client preserves the request value without conversion.
- A `403` may mean either an inaccessible or missing resource. Do not translate
  all `403` errors into Terraform resource deletion. No absence helper does so.
- Monitoring, pagination, VDS resizing, and general VDS updates have no documented
  operations and are not implemented.

## Testing

The default suite uses local HTTP servers and fixtures. It needs no credentials
or access to HexCore:

```sh
go test ./...
go vet ./...
go test -race ./...
```

Race tests require a supported C compiler and CGO. The module declares Go 1.22;
the suite should also pass with a Go 1.22 toolchain.

### Phase 2: live contract checks

Run these only after the offline suite passes. Configure a token locally;
do not commit credentials. Set `HEXCORE_LIVE=1` and `HEXCORE_TOKEN` to run reads:

```sh
go test -run '^TestLiveReads$' -v -count=1
```

Tests read process environment variables; `.env` is not loaded automatically.
`HEXCORE_BASE_URL` optionally overrides the endpoint. For writes, also set
`HEXCORE_LIVE_WRITES=1`. Start with one standard VDS create/read/destroy cycle,
setting `HEXCORE_REGION`, `HEXCORE_SIZE`, and `HEXCORE_IMAGE` from the catalogs:

```sh
go test -run '^TestLiveVDSProvisioning$' -v -count=1 -timeout 25m
```

This check makes one creation attempt and registers cleanup before sending it.
A rejected creation fails the check; it is not retried. Cleanup reconciles the
unique name even when the creation response cannot be decoded.

VDS lifecycle checks can also run independently of additional disk provisioning,
using the same standard VDS settings and creating one disposable server:

```sh
go test -run '^TestLiveVDSActions$' -v -count=1 -timeout 45m
```

To isolate graceful reboot on a fresh VDS when another action is blocked, use
`-run 'TestLiveVDSActions/^reboot$'` instead. This still creates and cleans up
one disposable server.

After provisioning succeeds, select these settings for the broader write suite:

| Variable | Meaning |
| --- | --- |
| `HEXCORE_REGION` | An available region slug |
| `HEXCORE_SIZE` | Standard VDS size slug |
| `HEXCORE_IMAGE` | OS image slug |
| `HEXCORE_CUSTOM_VCPU` | Custom VDS CPU count |
| `HEXCORE_CUSTOM_MEMORY` | Custom VDS memory in GB |
| `HEXCORE_CUSTOM_DISK` | Custom VDS disk size in GB |
| `HEXCORE_DISK_SIZE` | Initial additional disk size in GB; test grows it by 1 |

```sh
go test -run '^TestLiveWrites$' -v -count=1 -timeout 45m
```

The write test creates two billable VDS sequentially and one additional disk
with unique names, keeping only one VDS allocated at a time. It checks both
creation modes, power actions, password reset, disk
attachment/detachment, resize, polling, and a known deleted-resource response.
Passwords are generated in memory. Readiness polling is test orchestration,
not implicit SDK behavior. The memory assertion verifies the observed 1024
multiplier. Actions have up to ten minutes to complete, and resource-readiness
checks have up to five minutes, within the suite's overall deadline.
State-dependent checks stop after a failed step.
The harness only repeats commands explicitly rejected with the observed
`Action not allowed by VDS.` response while waiting for guest readiness; it
never repeats accepted or ambiguously failed submissions for that check.

Cleanup attempts to unplug the test disk, hard shutdown each disposable VDS,
and destroy it even after a failure. It polls VDS state independently of guest
commands and action history. Disk cleanup has a separate deadline from VDS
cleanup. Cleanup reconciles unique resource names if creation's response was lost
or could not be decoded. **Remove the reported disk IDs through the control
panel after every run.** Inspect the unique names if cleanup discovery fails
or the process is interrupted. Only resources created by that run are eligible
for destructive checks.

Record verified behavior and update codecs, fixtures, and these notes after
phase 2. Passing mock tests alone does not establish live API compatibility.

Live verification on 2026-10-19 passed account retrieval, region/size/image/
application catalogs, VDS and action listing/retrieval, disk creation/listing/
retrieval, standard/custom VDS creation, and VDS deletion after halting.
Insufficient-funds rejection was verified before the account was funded.
After funding, hard shutdown, power-on, hard reboot, and graceful shutdown
completed in sequential checks. Graceful reboot and password reset remained
in progress after separate ten-minute checks on fresh fixtures. Disk attachment
and subsequent disk actions are blocked by `Can not get Disk status.`; one
inspected disk also had a missing control-panel UUID. These findings do not
establish that all acknowledged writes complete successfully.

## Contributing and support

Read the [contribution guide](.github/CONTRIBUTING.md) before opening a pull
request. Use [Discussions](https://github.com/2chevskii/go-hexcore/discussions)
for questions and the [issue templates](https://github.com/2chevskii/go-hexcore/issues/new/choose)
for bugs, feature requests, and documentation improvements. See the
[security policy](.github/SECURITY.md) for private vulnerability reporting.

## License

This library is available under the [MIT License](LICENSE).
The bundled [Go development skill](.agents/skills/go-expert/NOTICE) retains its
upstream [Apache License 2.0](.agents/skills/go-expert/LICENSE).
