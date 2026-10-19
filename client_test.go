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
