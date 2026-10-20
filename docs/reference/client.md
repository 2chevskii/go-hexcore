# Client

Create a client with your API token:

<<< @/examples/recipes/configuration.go#client

`NewClient` checks the settings without making a request. Reuse the returned
client for subsequent calls.

## Settings

| `Config` field | Default | Purpose |
| --- | --- | --- |
| `Token` | Required | Raw token, without `Bearer ` or whitespace. |
| `BaseURL` | `https://api.hexcore.ru/` | API endpoint; an optional path prefix is supported. |
| `HTTPClient` | A client with a 30-second timeout | Your own `*http.Client`, used as supplied. |
| `UserAgent` | `go-hexcore` | Replaces the request's User-Agent. Newlines are rejected. |

See [configuration examples](../guide/configuration.md) for custom settings.

## Services

| Field | Use it to |
| --- | --- |
| [`Account`](./account.md) | Read your balance and account details. |
| [`Regions`, `Sizes`, `Images`, `Applications`](./catalogs.md) | Choose where and what to deploy. |
| [`VDS`](./vds.md) | Create, inspect, restart, and delete servers. |
| [`Disks`](./disks.md) | Create and manage additional storage. |
| [`Actions`](./actions.md) | Read action history and wait for work to finish. |

Every method takes a context first and returns an error. Reads and creations
return a resource pointer; lists return slices. See
[pkg.go.dev](https://pkg.go.dev/github.com/2chevskii/go-hexcore) for every type and field.
