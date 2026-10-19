package hexcore

import (
	"context"
	"net/http"
)

// AccountService retrieves the authenticated account.
type AccountService struct{ client *Client }

// Get retrieves information about the current account.
func (s *AccountService) Get(ctx context.Context) (*Account, error) {
	return requestOne[Account](ctx, s.client, http.MethodGet, "/account", nil, http.StatusOK, "account")
}
