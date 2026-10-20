# Errors and retries

Check the error before using a result. The library does not retry requests.

## Read an HTTP error

Use `errors.As` to get the API's status, message, and response headers:

<<< @/examples/recipes/errors.go#http-error

`APIError` also includes the request method and relative path. Request bodies
and authorization headers are excluded. Transport and cancellation errors stay
available through `errors.Is` and `errors.As`.

## Read an action failure

When `Actions.Wait` sees an action with status `error`, it returns an
`*ActionError` and the failed action:

<<< @/examples/recipes/errors.go#action-error

## When to retry

| Result | What to do |
| --- | --- |
| Temporary read failure | Retry with a deadline and backoff. Respect `Retry-After` when present. |
| Accepted write | Wait for the action or resource; do not submit it again. |
| Write timeout, connection failure, or unreadable response | Check whether the operation took effect. Use its ID or the unique name you supplied. |
| Insufficient funds or invalid input | Fix the cause before retrying. |
| HTTP 403 | Read the message. It can mean missing, inaccessible, or another API rejection. |

Do not treat HTTP 403 as proof of deletion. Even an empty disk or action list
can return 403; see [API compatibility](../compatibility.md).
