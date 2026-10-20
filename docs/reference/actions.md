# Actions

Power, password, and disk operations return an action. Use `client.Actions` to
check its progress or wait for it.

| Method | Returns | Use |
| --- | --- | --- |
| `Get(ctx, id)` | `(*Action, error)` | Read one action. |
| `List(ctx)` | `([]Action, error)` | Read your account's action history. |
| `Wait(ctx, id, opts)` | `(*Action, error)` | Poll until completion, failure, or deadline. |

## Read action history

<<< @/examples/recipes/actions.go#history

`Get` takes an action ID, not a VDS or disk ID. An empty history can return
HTTP 403; see [API compatibility](../compatibility.md).

## Wait for completion

Use a context deadline and the ID returned by the operation:

<<< @/examples/recipes/actions.go#wait

`WaitOptions{}` polls immediately and then every two seconds. Set a positive
`PollInterval` to change that interval; negative values are rejected. See the
[submission-and-wait example](../guide/async.md#wait-for-an-action).

## Wait results

| Status | Constant | Result |
| --- | --- | --- |
| `in-progress` | `ActionInProgress` | Keep polling. |
| `completed` | `ActionCompleted` | Return the action and nil error. |
| `error` | `ActionFailed` | Return the action and `*ActionError`. |
| Anything else | None | Return the action and an error. |

Request errors and cancellation stop polling. No operation is resubmitted.
See [handling action failures](../guide/errors.md#read-an-action-failure).

`StartedAt` is a string; `CompletedAt` and `Message` are optional. See
[Action fields](https://pkg.go.dev/github.com/2chevskii/go-hexcore#Action).
