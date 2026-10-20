# Actions and resource readiness

An operation can return before the server has finished it. What you do next
depends on the return value:

| Operation | Next step |
| --- | --- |
| Power or password action | Pass its action ID to `Actions.Wait`. |
| VDS or disk creation | Poll the resource using its returned ID. Creation has no action ID. |
| VDS destruction | Poll `VDS.List` until the ID is gone. |

## Wait for an action

This forces an existing server to restart and waits up to ten minutes:

<<< @/examples/recipes/actions.go#restart

`Wait` checks immediately, then every two seconds. Set
`WaitOptions{PollInterval: 3 * time.Second}` to change the interval. A deadline
limits how long you wait; it does not guarantee completion.

`Wait` returns on completion, action failure, an unknown status, a request error,
or cancellation. See [action results](../reference/actions.md#wait-results).

## Wait for a new server

Keep the ID returned by `Create` or `CreateCustom` and poll for the state you
need. This helper checks for `running` and an IPv4 address:

<<< @/examples/recipes/vds.go#ready

Use `VDS.List` for this check because `VDS.Get` may omit status. A running server
with an IP address may still need time before SSH or your application is ready.

For disks, missing attachment fields mean the state is unknown. See
[disk limitations](../reference/disks.md#attachment-limits).

## Confirm deletion

Halt the server, confirm its status, then call `Destroy`. A nil error means the
API accepted deletion. Confirm it by checking that the ID is absent from a
successful `VDS.List` response. A failed read, including HTTP 403, proves
nothing about whether the resource still exists.

The [deletion example](../reference/vds.md#delete-a-server) handles both checks.
