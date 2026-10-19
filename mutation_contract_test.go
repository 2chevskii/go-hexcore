package hexcore_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	hexcore "github.com/2chevskii/go-hexcore"
)

func TestActionAcceptedResponses(t *testing.T) {
	for _, tt := range []struct {
		name string
		call func(*hexcore.Client) (*hexcore.Action, error)
	}{
		{"VDS action", func(c *hexcore.Client) (*hexcore.Action, error) { return c.VDS.Reboot(context.Background(), "123") }},
		{"disk action", func(c *hexcore.Client) (*hexcore.Action, error) { return c.Disks.Plug(context.Background(), "123") }},
		{"destroy acknowledgement", func(c *hexcore.Client) (*hexcore.Action, error) {
			return nil, c.VDS.Destroy(context.Background(), "123")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" {
					t.Error("action did not use POST")
				}
				w.WriteHeader(http.StatusAccepted)
				_, _ = io.WriteString(w, `{"action":{"id":"5","resource_id":123,"resource_type":"vds","status":"in-progress","date_started":"2000-01-01T00:00:00Z"}}`)
			})
			action, err := tt.call(c)
			if err != nil {
				t.Fatal(err)
			}
			if tt.name != "destroy acknowledgement" && (action == nil || action.ID != "5" || action.ResourceID != "123" || action.Status != hexcore.ActionInProgress) {
				t.Fatalf("action = %#v", action)
			}
			if calls.Load() != 1 {
				t.Error("accepted action was replayed")
			}
		})
	}
	// Allowing 202 for asynchronous actions must not change read status checks.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, `{"account":{"id":"123"}}`)
	})
	_, err := c.Account.Get(context.Background())
	var apiErr *hexcore.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusAccepted {
		t.Fatalf("read error = %v; want unexpected HTTP 202", err)
	}
}

func TestLiveDestroyVDSCleanup(t *testing.T) {
	for _, tt := range []struct {
		name              string
		initialState      int32 // 0: running; 1: halted; 2: absent
		malformedShutdown bool
		wantHardShutdown  int32
		wantDestroy       int32
		wantErr           bool
	}{
		{"running VDS", 0, false, 1, 1, false},
		{"halted VDS", 1, false, 0, 1, false},
		{"already removed", 2, false, 0, 0, false},
		{"accepted hard shutdown with malformed response", 0, true, 1, 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var state, hardShutdowns, destroys atomic.Int32
			state.Store(tt.initialState)
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" && r.URL.Path == "/api/vds_list" {
					extra := ""
					if state.Load() != 2 {
						status := "Running"
						if state.Load() == 1 {
							status = "Halted"
						}
						extra = fmt.Sprintf(`,{"id":"123","status":%q}`, status)
					}
					_, _ = fmt.Fprintf(w, `{"vds":[{"id":"preexisting","status":"Running"}%s]}`, extra)
					return
				}
				if r.Method != "POST" || r.URL.Path != "/api/vds/123/action" {
					t.Errorf("cleanup touched an unexpected resource: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(404)
					return
				}
				var body struct {
					Type string `json:"type"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				switch body.Type {
				case "hard_shutdown":
					hardShutdowns.Add(1)
					state.Store(1)
					w.WriteHeader(202)
					if tt.malformedShutdown {
						_, _ = io.WriteString(w, `{"action":{}}`)
					} else {
						_, _ = io.WriteString(w, `{"action":{"id":"5","status":"in-progress"}}`)
					}
				case "destroy":
					if state.Load() != 1 {
						t.Error("destroy requested before the VDS halted")
						w.WriteHeader(403)
						return
					}
					destroys.Add(1)
					state.Store(2)
					w.WriteHeader(202)
				default:
					t.Errorf("unexpected cleanup action %q", body.Type)
					w.WriteHeader(400)
				}
			})
			err := liveDestroyVDS(context.Background(), c, "123")
			if (err != nil) != tt.wantErr || state.Load() != 2 || hardShutdowns.Load() != tt.wantHardShutdown || destroys.Load() != tt.wantDestroy {
				t.Fatalf("cleanup: error=%v state=%d hardShutdowns=%d destroys=%d", err, state.Load(), hardShutdowns.Load(), destroys.Load())
			}
		})
	}
}

func TestLiveCleanupPreservesAccessErrors(t *testing.T) {
	var shutdowns atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/vds_list" {
			_, _ = io.WriteString(w, `{"vds":[{"id":"123","status":"Running"}]}`)
			return
		}
		var body struct {
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/api/vds/123/action" || body.Type != "hard_shutdown" {
			t.Errorf("cleanup continued after access refusal: %s %s", r.URL.Path, body.Type)
		}
		shutdowns.Add(1)
		w.WriteHeader(403)
		_, _ = io.WriteString(w, `{"vds":{"message":"Access denied."}}`)
	})
	err := liveDestroyVDS(context.Background(), c, "123")
	var apiErr *hexcore.APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "Access denied." || shutdowns.Load() != 1 {
		t.Fatalf("cleanup = %v; attempts=%d", err, shutdowns.Load())
	}
}

func TestLiveActionReadiness(t *testing.T) {
	for _, tt := range []struct {
		name          string
		initialStatus int
		initialBody   string
		wantCalls     int32
		wantErr       bool
	}{
		{"explicit guest refusal", 403, `{"vds":{"message":"Action not allowed by VDS."}}`, 2, false},
		{"access error", 403, `{"vds":{"message":"Access denied."}}`, 1, true},
		{"accepted but malformed", 202, `{"action":{}}`, 1, true},
		{"accepted", 202, `{"action":{"id":"5","status":"in-progress"}}`, 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.WriteHeader(tt.initialStatus)
					_, _ = io.WriteString(w, tt.initialBody)
					return
				}
				w.WriteHeader(202)
				_, _ = io.WriteString(w, `{"action":{"id":"5","status":"in-progress"}}`)
			})
			action, err := liveVDSAction(context.Background(), "123", c.VDS.Reboot)
			if (err != nil) != tt.wantErr || calls.Load() != tt.wantCalls || (!tt.wantErr && (action == nil || action.ID != "5")) {
				t.Fatalf("action=%#v error=%v attempts=%d", action, err, calls.Load())
			}
		})
	}
}
