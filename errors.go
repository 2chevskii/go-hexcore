package hexcore

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		var message struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &message) == nil && message.Message != "" {
			e.Message = message.Message
		} else {
			// Live errors can be wrapped under the resource key, for example
			// {"actions":{"message":"No actions found."}}. Restrict lookup
			// to the requested resource rather than arbitrary nested objects.
			key := strings.Split(strings.TrimPrefix(path, "/"), "/")[0]
			key = strings.TrimSuffix(key, "_list")
			var envelope map[string]json.RawMessage
			if json.Unmarshal(data, &envelope) == nil && json.Unmarshal(envelope[key], &message) == nil && message.Message != "" {
				e.Message = message.Message
			}
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
