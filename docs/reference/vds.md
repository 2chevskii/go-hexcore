# VDS

Use `client.VDS` to manage virtual servers. All methods take a context first.

## Read servers

| Method | Returns |
| --- | --- |
| `List(ctx)` | `([]VDS, error)` |
| `Get(ctx, id)` | `(*VDS, error)` |

List servers with the [quick start](../guide/getting-started.md). To read one:

<<< @/examples/recipes/reads.go#vds

IDs are strings. Some fields, including status and region, may be missing from
`Get` even when `List` supplies them. See [optional fields](../data-conventions.md#optional-fields).

## Create a server

Choose size, image, and region slugs from the [catalogs](./catalogs.md). This
creates a billable server and returns a `VDS` with its ID before it is ready:

<<< @/examples/recipes/vds.go#create

`Create(ctx, CreateVDSRequest)` returns `(*VDS, error)`. Use a unique name of at
most 45 characters so you can find the server if the response is lost. Keep its
ID for later management, including deletion. Set `Password` if you need to
specify the initial password; an empty value is omitted.

Creation returns no action ID. To wait, use the
[polling example](../guide/async.md#wait-for-a-new-server) with the returned resource ID.

### Choose CPU, memory, and disk sizes yourself

Use `CreateCustom` with a `CustomSize` instead of a plan slug:

<<< @/examples/recipes/vds.go#custom

`CreateCustom` returns `(*VDS, error)`. Request memory and disk space in GB;
returned memory is in MB (`1` GB becomes `1024` MB). Meet the image's minimum
disk size. The library adds the custom-size flags required by the API.

## Power and password actions

Each method returns `(*Action, error)`. Pass its `ID` to `Actions.Wait` before
starting an operation that depends on it; see the
[reboot example](../guide/async.md#wait-for-an-action).

| Method | Effect |
| --- | --- |
| `Reboot(ctx, id)` | Ask the guest OS to restart. |
| `HardReboot(ctx, id)` | Force a restart. |
| `Shutdown(ctx, id)` | Ask the guest OS to shut down. |
| `HardShutdown(ctx, id)` | Cut power. |
| `PowerOn(ctx, id)` | Start a halted server. |
| `ResetPassword(ctx, id, password)` | Change the root password and reboot. |

Graceful operations depend on the guest OS. Reboot and password-reset actions
can remain in progress until your deadline; inspect server state after a timeout.

## Delete a server

First halt the server with a shutdown action and confirm its status is `halted`.
`Destroy(ctx, id) error` accepts deletion without waiting for removal.
This helper checks the halted state, submits once, and waits until the ID
disappears from a successful listing:

<<< @/examples/recipes/vds.go#destroy

There is no general server update or VDS resize method. See
[VDS types](https://pkg.go.dev/github.com/2chevskii/go-hexcore#VDSService)
for all request and response fields.
