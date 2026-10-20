# Disks

Use `client.Disks` for additional storage. Disk sizes are in GB.

::: warning Attachment limits
Disk creation can return unknown attachment state, and disk actions can fail
with `403: Can not get Disk status.`. `Plug` has no target-VDS argument because
the API does not specify one. See [attachment limits](#attachment-limits) before
building an automated workflow.
:::

## Read disks

| Method | Returns |
| --- | --- |
| `List(ctx)` | `([]Disk, error)` |
| `Get(ctx, id)` | `(*Disk, error)` |

<<< @/examples/recipes/reads.go#disks

An empty disk list can return HTTP 403 rather than an empty slice. The library
returns that error; see [API compatibility](../compatibility.md).

## Create a disk

`Create(ctx, CreateDiskRequest)` returns `(*Disk, error)`. It creates billable
storage without waiting for readiness:

<<< @/examples/recipes/disks.go#create-disk

Pass nil as `vdsID` to omit attachment, or a pointer to a positive decimal VDS ID
to request it. The library sends that ID as a JSON number. The returned disk
has no action ID. Keep its ID and unique name for management and cleanup.

## Attach, detach, and grow

These methods return `(*Action, error)`. Wait using the returned action ID.

| Method | Requirement |
| --- | --- |
| `Plug(ctx, id)` | The API chooses the target; shutdown/start of the VDS may be needed to activate it. |
| `Unplug(ctx, id)` | Requests detachment; confirm disk state before resizing. |
| `Resize(ctx, id, size)` | Disk must be unplugged. `size` is the new total in GB and must increase. |

This helper checks the disk before requesting growth:

<<< @/examples/recipes/disks.go#grow-disk

## Attachment limits

Check `Disk.Status` and `Disk.VDSID` for nil. Missing values mean unknown, not
confirmed attached or detached. A request's acceptance alone does not confirm
that attachment or resizing succeeded.

There is no disk deletion method in the documented API. Remove disks through
the control panel; contact HexCore if an incomplete disk record prevents it.

See [disk types](https://pkg.go.dev/github.com/2chevskii/go-hexcore#DisksService)
for all fields.
