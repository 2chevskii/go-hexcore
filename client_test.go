package hexcore_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewClient(t *testing.T) {
	for _, cfg := range []hexcore.Config{
		{}, {Token: "bad\ntoken"}, {Token: "Bearer token"},
		{Token: "test", BaseURL: "://broken"}, {Token: "test", BaseURL: "/relative"},
		{Token: "test", BaseURL: "ftp://example.com"}, {Token: "test", BaseURL: "https://user:pass@example.com"},
		{Token: "test", BaseURL: "https://example.com?q=1"}, {Token: "test", BaseURL: "https://example.com?"},
		{Token: "test", BaseURL: "https://example.com#fragment"}, {Token: "test", UserAgent: "bad\r\nagent"},
	} {
		if _, err := hexcore.NewClient(cfg); err == nil {
			t.Error("accepted invalid configuration")
		}
	}
	httpClient := &http.Client{Timeout: 7 * time.Second, Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != hexcore.DefaultBaseURL+"regions" || r.Header.Get("User-Agent") != "go-hexcore" {
			t.Errorf("incorrect defaults: URL=%s user agent=%s", r.URL, r.Header.Get("User-Agent"))
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"regions":[]}`))}, nil
	})}
	c, err := hexcore.NewClient(hexcore.Config{Token: "test", HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	if httpClient.Timeout != 7*time.Second {
		t.Fatal("modified caller's HTTP timeout")
	}
	regions, err := c.Regions.List(context.Background())
	if err != nil || regions == nil || len(regions) != 0 {
		t.Fatalf("empty list = %#v, %v", regions, err)
	}
}

func TestAPIErrorsAndNoRetries(t *testing.T) {
	for _, tt := range []struct {
		name, body, message string
		status              int
	}{
		{"unauthorized", `{"message":"Invalid token"}`, "Invalid token", 401},
		{"forbidden", `{"message":"Missing or inaccessible"}`, "Missing or inaccessible", 403},
		{"not found", ``, "Not Found", 404},
		{"throttled", `{"message":"Try later"}`, "Try later", 429},
		{"HTML", `<h1>Unavailable</h1>`, "Service Unavailable", 503},
		{"malformed", `{"message":`, "Internal Server Error", 500},
		{"unexpected success", ``, "Created", 201},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, write := range []bool{false, true} {
				var calls atomic.Int32
				c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Retry-After", "12")
					w.WriteHeader(tt.status)
					_, _ = io.WriteString(w, tt.body)
				})
				var err error
				method, path := "GET", "/vds/123"
				if write {
					_, err = c.VDS.Reboot(context.Background(), "123")
					method, path = "POST", "/vds/123/action"
				} else {
					_, err = c.VDS.Get(context.Background(), "123")
				}
				var apiErr *hexcore.APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("error = %v; want APIError", err)
				}
				if apiErr.StatusCode != tt.status || apiErr.Message != tt.message || apiErr.Method != method || apiErr.Path != path || apiErr.Headers.Get("Retry-After") != "12" {
					t.Errorf("APIError = %#v", apiErr)
				}
				if calls.Load() != 1 {
					t.Errorf("attempts = %d; want 1", calls.Load())
				}
				if strings.Contains(err.Error(), "test-token") {
					t.Error("error contains bearer token")
				}
			}
		})
	}
}

func TestResponseValidation(t *testing.T) {
	for _, body := range []string{
		``, `not-json`, `null`, `[]`, `{}`, `{"vds":null}`, `{"vds":{}}`,
		`{"vds":[]}`, `{"vds":[null]}`, `{"vds":[{}]}`, `{"vds":[{"id":"1"},{"id":"2"}]}`,
		`{"vds":[{"id":"1","memory":"bad"}]}`, `{"vds":[{"id":"1","memory":1.2}]}`,
		`{"vds":[{"id":"1"}]} {}`, `{"vds":[{"id":"1"}]} garbage`,
	} {
		t.Run(body, func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, body) })
			if v, err := c.VDS.Get(context.Background(), "1"); err == nil || v != nil {
				t.Fatalf("result = %#v, %v; want error", v, err)
			}
		})
	}
}

func TestTransportAndCancellation(t *testing.T) {
	sentinel := errors.New("transport offline")
	var attempts atomic.Int32
	c, err := hexcore.NewClient(hexcore.Config{Token: "test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts.Add(1)
		return nil, sentinel
	})}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Account.Get(context.Background()); !errors.Is(err, sentinel) || attempts.Load() != 1 {
		t.Fatalf("transport error = %v; attempts=%d", err, attempts.Load())
	}
	started := make(chan struct{})
	c = testClient(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := c.Account.Get(ctx); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestResponseBodyCleanup(t *testing.T) {
	sentinel := errors.New("body read failed")
	for _, tt := range []struct {
		status    int
		reader    io.Reader
		write     bool
		wantErr   bool
		wantCause bool
	}{
		{200, strings.NewReader(`{"account":[{"id":"2"}]}`), false, false, false},
		{200, strings.NewReader(`bad-json`), false, true, false},
		{403, strings.NewReader(`{"message":"denied"}`), false, true, false},
		{204, strings.NewReader(``), true, false, false},
		{200, failingReader{sentinel}, false, true, true},
		{403, failingReader{sentinel}, false, true, true},
		{204, failingReader{sentinel}, true, true, true},
	} {
		body := &trackedBody{Reader: tt.reader}
		c, err := hexcore.NewClient(hexcore.Config{Token: "test", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tt.status, Header: make(http.Header), Body: body}, nil
		})}})
		if err != nil {
			t.Fatal(err)
		}
		if tt.write {
			err = c.VDS.Destroy(context.Background(), "1")
		} else {
			_, err = c.Account.Get(context.Background())
		}
		if (err != nil) != tt.wantErr || !body.closed || (tt.wantCause && !errors.Is(err, sentinel)) {
			t.Errorf("status=%d: error=%v closed=%v", tt.status, err, body.closed)
		}
	}
}

func TestConcurrentClient(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, `{"regions":[]}`) })
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Regions.List(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestPasswordResetIncludesRequiredPassword(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		equalJSON(t, string(data), `{"type":"reset_password","password":""}`)
		_, _ = io.WriteString(w, `{"action":[{"id":"9","status":"in-progress"}]}`)
	})
	if _, err := c.VDS.ResetPassword(context.Background(), "1", ""); err != nil {
		t.Fatal(err)
	}
}
