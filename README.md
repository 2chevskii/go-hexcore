# go-hexcore

A Go 1.22+ client for the [HexCore HTTP API](https://dev.hexcore.ru/), intended
for use by a future Terraform provider. It uses only the Go standard library.

Phase 1 is implemented and covered by offline contract tests. Live compatibility
has **not** been verified. The opt-in phase 2 tests are described below.

```sh
go get github.com/2chevskii/go-hexcore
```

This workspace must be published at that module path before `go get` can fetch
this implementation.

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
documented success status and response envelope; malformed or unexpected
responses are errors. Unknown resource properties are ignored.

Resource IDs are strings. Quantities are `int64`, including values encoded as
strings by the server. Monetary values and timestamps preserve their textual
representation: no floating-point conversion or timezone assumption is made.
Optional fields use pointers where endpoint responses differ. In particular,
`VDS.Get` does not fill missing fields by making a second request to `VDS.List`.
Disk attachment status and VDS ID are exposed as `Disk.Status` and `Disk.VDSID`.

Standard creation uses `CreateVDSRequest` with size, image, and region slugs.
`CreateCustomVDSRequest.Size` is a `CustomSize`; the client supplies `size=0` and
`custom_size=1`. Custom memory is sent unchanged; its request unit remains
unverified. Disk sizes are in GB. `CreateDiskRequest.VDSID` optionally attaches
the disk during creation; this string must contain a positive decimal ID and
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
Non-JSON error bodies fall back to the HTTP status text. Request bodies and
authorization headers are not included in diagnostics. Cancellation and
transport errors remain discoverable through `errors.Is`/`errors.As`.

## Documentation gaps

The client is handwritten against the [Swagger 2.0 document](https://dev.hexcore.ru/hexcore.json?v=0.8.7).
The URL revision is `0.8.7`, while `info.version` is `0.6.2`. Saved success
examples in `testdata/responses.json` were retrieved on 2026-10-19. Tests retain
example inconsistencies, such as the unplug response reporting type `plug`.

- Resource routes use parameterized IDs from the prose, rather than literal
  example IDs in the Swagger path keys. This includes `/actions/{id}` and
  `/disk/{id}`. Both standard and custom creation use `/vds`, without a space.
- Read responses and action submissions use different array envelopes and
  timestamp names. The client unwraps and normalizes these documented variants.
- `Disks.Plug` sends only `{"type":"plug"}`. The documentation does not describe
  a target-VDS parameter. Its target-selection behavior needs live verification.
- Disk deletion is mentioned but has no documented operation. No disk deletion
  method is implemented. Use the control panel for cleanup.
- Disk resize requires an unplugged disk and only permits growth. Plugging may
  require a VDS shutdown and start before a disk becomes active. The client does
  not orchestrate these prerequisites.
- The custom VDS example sends memory `3` and returns `3072` MB. The client
  preserves the request value; phase 2 checks this apparent conversion.
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

`HEXCORE_BASE_URL` optionally overrides the endpoint. For writes, also set
`HEXCORE_LIVE_WRITES=1` and explicitly select the following:

| Variable | Meaning |
| --- | --- |
| `HEXCORE_REGION` | An available region slug |
| `HEXCORE_SIZE` | Standard VDS size slug |
| `HEXCORE_IMAGE` | OS image slug |
| `HEXCORE_CUSTOM_VCPU` | Custom VDS CPU count |
| `HEXCORE_CUSTOM_MEMORY` | Raw custom memory request quantity |
| `HEXCORE_CUSTOM_DISK` | Custom VDS disk size in GB |
| `HEXCORE_DISK_SIZE` | Initial additional disk size in GB; test grows it by 1 |

```sh
go test -run '^TestLive' -v -count=1 -timeout 45m
```

The write test creates two billable VDS and one additional disk with unique
names. It checks both creation modes, power actions, password reset, disk
attachment/detachment, resize, polling, and a known deleted-resource response.
Passwords are generated in memory. Readiness polling is test orchestration,
not implicit SDK behavior. The memory assertion tests the example's apparent
1024 multiplier; a mismatch requires contract investigation.

Cleanup attempts to unplug the test disk and destroy the created VDS even after
a failure. It reconciles unique resource names if creation's response was lost
or could not be decoded. **Remove the reported disk IDs through the control
panel after every run.** Inspect the unique names if cleanup discovery fails
or the process is interrupted. Only resources created by that run are eligible
for destructive checks.

Record verified behavior and update codecs, fixtures, and these notes after
phase 2. Passing mock tests alone does not establish live API compatibility.
