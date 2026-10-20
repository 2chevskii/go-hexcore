# Account

`client.Account.Get(ctx)` returns `(*Account, error)` with your balance, currency,
email, account status, and billing details.

## Check your balance

<<< @/examples/recipes/reads.go#balance

Balances and prices are strings so their decimal values stay intact.
`ConfigData` contains company billing details when present; check for nil before
reading it. See [Account fields](https://pkg.go.dev/github.com/2chevskii/go-hexcore#Account).
