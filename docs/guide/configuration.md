# Authentication and configuration

Pass your raw API token in `Config.Token`. The client adds the `Bearer ` prefix.
An empty token or one containing whitespace is rejected before any request.

## Change the HTTP timeout or User-Agent

<<< @/examples/recipes/configuration.go#configure

Without these settings, the client uses a 30-second HTTP timeout and the
User-Agent `go-hexcore`. A supplied `http.Client` keeps its timeout and transport;
if its timeout is zero, requests have no HTTP timeout. Use
[context deadlines](./contexts.md) to bound the whole operation.

## Use a different endpoint

Set `Config.BaseURL` for a proxy or test server:

<<< @/examples/recipes/configuration.go#endpoint

The default is `https://api.hexcore.ru/`. A path prefix is allowed; URL
credentials, query parameters, and fragments are not.

Create the client once and reuse it, including across goroutines. Do not replace
its services or change its HTTP client while requests are running.
