// Package hexcore provides a client for the HexCore HTTP API.
//
// Methods make a single HTTP request and do not retry or wait for resource
// readiness. Actions.Wait explicitly polls asynchronous actions. All operations
// accept a context for cancellation and deadlines.
//
// The client implements the documented API at https://dev.hexcore.ru/.
// Resource IDs are strings, quantities are integers, and monetary values and
// dates retain the server's textual representation.
package hexcore
