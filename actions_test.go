package hexcore_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

func TestWaitAction(t *testing.T) {
	for _, tt := range []struct {
		name     string
		statuses []string
		wantErr  bool
		failed   bool
	}{
		{"already completed", []string{"completed"}, false, false},
		{"poll until completed", []string{"in-progress", "in-progress", "completed"}, false, false},
		{"failed", []string{"in-progress", "error"}, true, true},
		{"unknown status", []string{"unexpected"}, true, false},
		{"missing status", []string{""}, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				i := int(calls.Add(1)) - 1
				if i >= len(tt.statuses) {
					t.Error("unexpected additional poll")
					w.WriteHeader(500)
					return
				}
				if r.Method != "GET" || r.URL.Path != "/api/actions/9" {
					t.Errorf("unexpected polling request %s %s", r.Method, r.URL.Path)
				}
				_, _ = fmt.Fprintf(w, `{"actions":[{"id":"9","status":%q,"message":"operation detail"}]}`, tt.statuses[i])
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			action, err := c.Actions.Wait(ctx, "9", hexcore.WaitOptions{PollInterval: time.Microsecond})
			if (err != nil) != tt.wantErr || action == nil || action.ID != "9" {
				t.Fatalf("Wait = %#v, %v", action, err)
			}
			if calls.Load() != int32(len(tt.statuses)) {
				t.Errorf("polls = %d; want %d", calls.Load(), len(tt.statuses))
			}
			var failed *hexcore.ActionError
			if errors.As(err, &failed) != tt.failed {
				t.Errorf("ActionError = %v; expected=%v", err, tt.failed)
			}
			if failed != nil && (failed.Action != action || !strings.Contains(failed.Error(), "operation detail")) {
				t.Error("ActionError did not retain the failed action")
			}
		})
	}
}

func TestWaitCancellationDuringDelay(t *testing.T) {
	for _, interval := range []time.Duration{0, time.Hour} {
		var calls atomic.Int32
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			_, _ = io.WriteString(w, `{"actions":[{"id":"9","status":"in-progress"}]}`)
		})
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		_, err := c.Actions.Wait(ctx, "9", hexcore.WaitOptions{PollInterval: interval})
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
			t.Fatalf("Wait = %v; polls=%d", err, calls.Load())
		}
	}
}

func TestWaitHTTPFailureAndInvalidInterval(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"actions":[{"id":"9","status":"in-progress"}]}`)
			return
		}
		w.WriteHeader(503)
	})
	if _, err := c.Actions.Wait(context.Background(), "9", hexcore.WaitOptions{PollInterval: -time.Second}); err == nil || calls.Load() != 0 {
		t.Fatal("negative interval was not rejected before polling")
	}
	_, err := c.Actions.Wait(context.Background(), "9", hexcore.WaitOptions{PollInterval: time.Microsecond})
	var apiErr *hexcore.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 || calls.Load() != 2 {
		t.Fatalf("poll failure = %v; polls=%d", err, calls.Load())
	}
}
