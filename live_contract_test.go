package hexcore_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"

	hexcore "github.com/2chevskii/go-hexcore"
)

// These fixtures preserve observed live response shapes while replacing account
// identifiers, email, dates, and monetary values with synthetic data.
func TestLiveResponseContracts(t *testing.T) {
	data, err := os.ReadFile("testdata/live-responses.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, path, message string
		call                func(*hexcore.Client) (any, error)
		want                any
	}{
		{"account_object", "/account", "", func(c *hexcore.Client) (any, error) { return c.Account.Get(context.Background()) },
			&hexcore.Account{ID: "123", Email: "account@example.invalid", Status: 1, Type: 1, Balance: "12.34", Currency: "RUB", Discount: "0", CreatedAt: "2000-01-01 00:00:00"}},
		{"empty_actions", "/actions_list", "No actions found.", func(c *hexcore.Client) (any, error) { return c.Actions.List(context.Background()) }, nil},
		{"empty_disks", "/disk_list", "Can not find disk for your account.", func(c *hexcore.Client) (any, error) { return c.Disks.List(context.Background()) }, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture, ok := fixtures[tt.name]
			if !ok {
				t.Fatal("missing live response fixture")
			}
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api"+tt.path {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(fixture.Status)
				_, _ = w.Write(fixture.Body)
			})
			got, err := tt.call(c)
			if tt.message != "" {
				var apiErr *hexcore.APIError
				if !errors.As(err, &apiErr) || apiErr.StatusCode != 403 || apiErr.Path != tt.path || apiErr.Message != tt.message {
					t.Fatalf("error = %v; want original 403 and nested message", err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("result = %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
}

func TestLiveEmptyCollectionRecognition(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{"observed empty", &hexcore.APIError{StatusCode: 403, Method: "GET", Path: "/actions_list", Message: "No actions found."}, true},
		{"no error", nil, false},
		{"unauthorized", &hexcore.APIError{StatusCode: 401, Method: "GET", Path: "/actions_list", Message: "No actions found."}, false},
		{"different message", &hexcore.APIError{StatusCode: 403, Method: "GET", Path: "/actions_list", Message: "Access denied."}, false},
		{"singular resource", &hexcore.APIError{StatusCode: 403, Method: "GET", Path: "/actions/123", Message: "No actions found."}, false},
		{"write operation", &hexcore.APIError{StatusCode: 403, Method: "POST", Path: "/actions_list", Message: "No actions found."}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := liveEmptyCollection(tt.err, "/actions_list", "No actions found."); got != tt.want {
				t.Fatalf("recognized = %v; want %v", got, tt.want)
			}
		})
	}
}
