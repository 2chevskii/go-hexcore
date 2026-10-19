package hexcore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ActionsService retrieves action history and polls asynchronous operations.
type ActionsService struct{ client *Client }

// Get retrieves one action using the /actions/{id} route in the documentation.
func (s *ActionsService) Get(ctx context.Context, id string) (*Action, error) {
	path, err := resourcePath("actions", id, "")
	if err != nil {
		return nil, err
	}
	return requestOne[Action](ctx, s.client, http.MethodGet, path, nil, http.StatusOK, "actions")
}

// List retrieves the account's action history.
func (s *ActionsService) List(ctx context.Context) ([]Action, error) {
	return requestList[Action](ctx, s.client, http.MethodGet, "/actions_list", nil, http.StatusOK, "actions")
}

// WaitOptions configures explicit action polling. A zero PollInterval defaults
// to two seconds. The context, rather than this struct, controls the deadline.
type WaitOptions struct {
	PollInterval time.Duration
}

// Wait polls immediately and then at PollInterval until the action completes.
// Action failure returns the last action and an *ActionError. An unknown status
// returns the last action and an error. HTTP failures are returned without retry.
func (s *ActionsService) Wait(ctx context.Context, id string, opts WaitOptions) (*Action, error) {
	if opts.PollInterval < 0 {
		return nil, errors.New("hexcore: poll interval must not be negative")
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 2 * time.Second
	}
	for {
		action, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		switch action.Status {
		case ActionCompleted:
			return action, nil
		case ActionFailed:
			return action, &ActionError{Action: action}
		case ActionInProgress:
		default:
			return action, fmt.Errorf("hexcore: action %s has unknown status %q", id, action.Status)
		}
		timer := time.NewTimer(opts.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return action, fmt.Errorf("hexcore: wait for action %s: %w", id, ctx.Err())
		case <-timer.C:
		}
	}
}
