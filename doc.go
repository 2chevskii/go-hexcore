// Package hexcore provides a client for the HexCore HTTP API.
//
// Create a Client with NewClient, then use its Account, VDS, Disks, Actions,
// and catalog services. All methods accept a context for deadlines and
// cancellation. Requests are not retried. Use Actions.Wait to wait for an
// action; creation and deletion need separate resource checks.
//
// IDs are strings, quantities are integers, and prices and dates remain strings.
// Guides and examples: https://2chevskii.github.io/go-hexcore/.
package hexcore
