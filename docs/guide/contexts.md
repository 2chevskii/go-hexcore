# Contexts and deadlines

Every API method takes a context first. Use a deadline so a slow request or
unfinished action cannot keep your program waiting forever.

## Limit a request

<<< @/examples/recipes/reads.go#deadline

Call `cancel` when finished to release the timer. In a server, pass the incoming
request's context as `parent` so disconnects can cancel the API request too.

## Limit a whole operation

Pass the same context to the action call and `Actions.Wait`. Its deadline covers
both submission and polling; see the [reboot example](./async.md#wait-for-an-action).

The HTTP client's timeout applies to each request. The context deadline applies
to everything using that context. Whichever expires first stops the request.

## After cancellation

Use `errors.Is(err, context.Canceled)` or
`errors.Is(err, context.DeadlineExceeded)` to recognize cancellation.

Canceling a request does not undo an operation the API has accepted. After a
timed-out write, check the action or resource before trying again. See
[retry decisions](./errors.md#when-to-retry).
