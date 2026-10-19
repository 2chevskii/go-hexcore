package hexcore_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	hexcore "github.com/2chevskii/go-hexcore"
)

func ptr[T any](v T) *T { return &v }

func testClient(t *testing.T, handler http.HandlerFunc) *hexcore.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c, err := hexcore.NewClient(hexcore.Config{Token: "test-token", BaseURL: server.URL + "/api/", UserAgent: "test-provider"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func equalJSON(t *testing.T, got, want string) {
	t.Helper()
	decode := func(text string) any {
		dec := json.NewDecoder(strings.NewReader(text))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("invalid test JSON: %v", err)
		}
		return v
	}
	if !reflect.DeepEqual(decode(got), decode(want)) {
		t.Errorf("JSON = %s; want %s", got, want)
	}
}
