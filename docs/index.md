# go-hexcore

A Go client for the [HexCore API](https://dev.hexcore.ru/). Use it to read account
and catalog data, create and manage virtual servers, work with additional disks,
and check the progress of asynchronous operations.

Requires **Go 1.22+** and a **HexCore API token**. The library uses only the Go
standard library.

## Install

```sh
go get github.com/2chevskii/go-hexcore
```

## Usage

Create a client with your raw token, without `Bearer `, then call a service.
For example, this function lists your servers with a 30-second deadline:

<<< @/examples/recipes/overview.go#list-servers

The [quick start](./guide/getting-started.md) has a runnable program with imports
and environment setup.

## Find what you need

| Task | Documentation |
| --- | --- |
| Change the endpoint, HTTP client, or User-Agent | [Client configuration](./guide/configuration.md) |
| Check your balance | [Account](./reference/account.md) |
| Choose a region, server plan, or OS image | [Catalogs](./reference/catalogs.md) |
| Create, restart, stop, or delete a server | [VDS](./reference/vds.md) |
| Create or grow an additional disk | [Disks](./reference/disks.md) |
| Wait for an operation or check resource readiness | [Actions and readiness](./guide/async.md) |
| Inspect an API failure or decide whether to retry | [Errors and retries](./guide/errors.md) |

Requests are not retried. Creation and deletion return before the resource is
ready or gone; your code must check completion. Some disk operations also have
[API limitations](./compatibility.md).

See [pkg.go.dev](https://pkg.go.dev/github.com/2chevskii/go-hexcore) for all
exported types and methods.
