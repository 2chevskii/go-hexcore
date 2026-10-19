package hexcore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError describes an unexpected HTTP status. Headers includes Retry-After
// when supplied by the server. A 403 is not evidence that a resource is absent.
type APIError struct {
	StatusCode int
	Message    string
	Method     string
	Path       string
	Headers    http.Header
	cause      error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hexcore: %s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, e.Message)
}

// Unwrap exposes an error encountered while reading the error response body.
func (e *APIError) Unwrap() error { return e.cause }

func readAPIError(res *http.Response, method, path string) *APIError {
	e := &APIError{StatusCode: res.StatusCode, Method: method, Path: path, Headers: res.Header.Clone(), Message: http.StatusText(res.StatusCode)}
	if e.Message == "" {
		e.Message = "unexpected HTTP status"
	}
	// Error responses are not required to be JSON. Do not include arbitrary HTML
	// or request bodies in diagnostics; retain a documented JSON message if any.
	data, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	e.cause = err
	if err == nil {
		var body struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &body) == nil && body.Message != "" {
			e.Message = body.Message
		}
	}
	return e
}

// ActionError reports an action that reached the terminal "error" status.
// Action retains the server's failure message and associated resource.
type ActionError struct {
	Action *Action
}

func (e *ActionError) Error() string {
	if e.Action.Message != nil && *e.Action.Message != "" {
		return fmt.Sprintf("hexcore: action %s failed: %s", e.Action.ID, *e.Action.Message)
	}
	return fmt.Sprintf("hexcore: action %s failed", e.Action.ID)
}
