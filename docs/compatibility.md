# API compatibility

These API behaviors matter when automating resource management.

## HTTP 403 can mean several things

The API can return 403 for access failures, insufficient funds, missing resources,
and these empty lists:

| Request | Message |
| --- | --- |
| `Actions.List` | `No actions found.` |
| `Disks.List` | `Can not find disk for your account.` |

The library returns `*APIError` with the original status and message. Check the
message for the cause; do not assume a resource was deleted because a read
returned 403. See [error handling](./guide/errors.md).

## Some operations need extra checks

- Creation returns a resource ID without an action ID. [Poll resource state](./guide/async.md#wait-for-a-new-server).
- Deletion returns when accepted. [Confirm removal](./reference/vds.md#delete-a-server) through a successful listing.
- Graceful reboot and password reset can stay in progress beyond your deadline.
  Check server state before trying another operation.
- Disk attachment fields can be unknown, and disk actions can return
  `Can not get Disk status.`. See [disk limits](./reference/disks.md#attachment-limits).

## Unsupported operations

The API does not document disk deletion, VDS resizing, general VDS updates,
monitoring, or pagination. The library has no methods for these operations.
