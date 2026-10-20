// Package recipes contains the examples used by the documentation.
package recipes

import (
	"net/http"
	"os"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// #region client
func NewClient() (*hexcore.Client, error) {
	return hexcore.NewClient(hexcore.Config{
		Token: os.Getenv("HEXCORE_TOKEN"),
	})
}

// #endregion client

// #region configure
func Configure(token string) (*hexcore.Client, error) {
	return hexcore.NewClient(hexcore.Config{
		Token:      token,
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
		UserAgent:  "my-app/1.0",
	})
}

// #endregion configure

// #region endpoint
func UseEndpoint(token, endpoint string) (*hexcore.Client, error) {
	return hexcore.NewClient(hexcore.Config{
		Token:   token,
		BaseURL: endpoint, // For example, https://gateway.example.com/hexcore/.
	})
}

// #endregion endpoint
