# go-hexcore

[![CI](https://github.com/2chevskii/go-hexcore/actions/workflows/ci.yml/badge.svg)](https://github.com/2chevskii/go-hexcore/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go 1.22+ client for the [HexCore API](https://dev.hexcore.ru/), built with the
Go standard library.

[Documentation](https://2chevskii.github.io/go-hexcore/) ·
[API reference](https://pkg.go.dev/github.com/2chevskii/go-hexcore) ·
[Quick start](docs/guide/getting-started.md)

## Install

```sh
go get github.com/2chevskii/go-hexcore
```

## List your servers

Save this as `main.go`, set `HEXCORE_TOKEN` to your API token without `Bearer `,
and run `go run .`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	client, err := hexcore.NewClient(hexcore.Config{Token: os.Getenv("HEXCORE_TOKEN")})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	servers, err := client.VDS.List(ctx)
	if err != nil {
		return err
	}
	for _, server := range servers {
		fmt.Printf("%s: %s\n", server.ID, server.Name)
	}
	return nil
}
```

## What you can do

| Service | Use |
| --- | --- |
| `Account` | Read account details and balance. |
| `Regions`, `Sizes`, `Images`, `Applications` | Choose deployment options. |
| `VDS` | Create, read, restart, stop, and delete servers; reset passwords. |
| `Disks` | Create, read, attach, detach, and grow additional storage. |
| `Actions` | Read action history and wait for completion. |

## Before automating writes

Requests are not retried. Creation returns before a resource is ready, and
deletion returns before removal is confirmed. Use bounded contexts and
[wait explicitly](docs/guide/async.md).

Some operations have API limitations, including disk attachment and deletion.
Check [compatibility](docs/compatibility.md) before relying on them.

## Help and contributions

Use [Discussions](https://github.com/2chevskii/go-hexcore/discussions) for questions
and [issues](https://github.com/2chevskii/go-hexcore/issues/new/choose) for bugs
and feature requests. Report vulnerabilities through the
[security policy](.github/SECURITY.md).

For development, see [Contributing](.github/CONTRIBUTING.md).
For releases, see the [release guide](.github/RELEASING.md) and [changelog](CHANGELOG.md).

## License

[MIT](LICENSE). The bundled [Go development skill](.agents/skills/go-expert/NOTICE)
retains its [Apache 2.0 license](.agents/skills/go-expert/LICENSE).
