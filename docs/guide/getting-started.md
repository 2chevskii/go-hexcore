# Quick start

You need Go 1.22+ and a HexCore API token.

## Install

In your Go module, run:

```sh
go get github.com/2chevskii/go-hexcore
```

For a new project, run `go mod init example.com/myapp` first.

## List your servers

Save this as `main.go`:

<<< @/examples/quickstart/main.go

Set `HEXCORE_TOKEN` to your API token, without `Bearer `, then run the program:

::: code-group

```sh [Bash]
export HEXCORE_TOKEN='your-api-token'
go run .
```

```powershell [PowerShell]
$env:HEXCORE_TOKEN = 'your-api-token'
go run .
```

:::

The program prints each server's ID and name. It stops after 30 seconds if the
request has not finished. The program reads the environment variable; the
library itself does not load `.env` files or find credentials.

## Next steps

- [Create a server](../reference/vds.md#create-a-server).
- [Wait for an action](./async.md#wait-for-an-action).
- [Change client settings](./configuration.md).
- [Handle errors](./errors.md).
