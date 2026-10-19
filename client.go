package hexcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the public HexCore API endpoint.
const DefaultBaseURL = "https://api.hexcore.ru/"

// Config configures a Client. Token is the raw token, without a Bearer prefix.
// A supplied HTTPClient is used as-is, including its timeout and transport.
// BaseURL and UserAgent default to DefaultBaseURL and "go-hexcore".
type Config struct {
	Token      string
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// Client exposes the HexCore API services. It supports concurrent use provided
// callers do not replace service fields or mutate the supplied HTTP client.
type Client struct {
	Account      *AccountService
	Actions      *ActionsService
	VDS          *VDSService
	Disks        *DisksService
	Regions      *RegionsService
	Sizes        *SizesService
	Images       *ImagesService
	Applications *ApplicationsService

	token     string
	baseURL   string
	http      *http.Client
	userAgent string
}

// NewClient validates configuration without making a network request.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Token == "" || strings.ContainsAny(cfg.Token, " \t\r\n") {
		return nil, errors.New("hexcore: a nonempty bearer token without whitespace is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, errors.New("hexcore: base URL must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "go-hexcore"
	}
	if strings.ContainsAny(cfg.UserAgent, "\r\n") {
		return nil, errors.New("hexcore: user agent must not contain newlines")
	}
	c := &Client{token: cfg.Token, baseURL: strings.TrimRight(u.String(), "/"), http: cfg.HTTPClient, userAgent: cfg.UserAgent}
	c.Account = &AccountService{c}
	c.Actions = &ActionsService{c}
	c.VDS = &VDSService{c}
	c.Disks = &DisksService{c}
	c.Regions = &RegionsService{c}
	c.Sizes = &SizesService{c}
	c.Images = &ImagesService{c}
	c.Applications = &ApplicationsService{c}
	return c, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, status int, out any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("hexcore: encode %s %s: %w", method, path, err)
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return fmt.Errorf("hexcore: construct %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hexcore: %s %s: %w", method, path, err)
	}
	defer res.Body.Close()
	if res.StatusCode != status {
		return readAPIError(res, method, path)
	}
	if out == nil {
		_, err = io.Copy(io.Discard, res.Body)
		if err != nil {
			return fmt.Errorf("hexcore: read %s %s: %w", method, path, err)
		}
		return nil
	}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("hexcore: decode %s %s: %w", method, path, err)
	}
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return fmt.Errorf("hexcore: decode %s %s: %w", method, path, err)
	}
	return nil
}

func requestList[T any](ctx context.Context, c *Client, method, path string, body any, status int, key string) ([]T, error) {
	var envelope map[string]json.RawMessage
	if err := c.do(ctx, method, path, body, status, &envelope); err != nil {
		return nil, err
	}
	raw := bytes.TrimSpace(envelope[key])
	if len(raw) == 0 || raw[0] != '[' {
		return nil, fmt.Errorf("hexcore: decode %s %s: expected %q array", method, path, key)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("hexcore: decode %s %s: %w", method, path, err)
	}
	result := make([]T, len(items))
	for i, item := range items {
		if bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
			return nil, fmt.Errorf("hexcore: decode %s %s: null %q item at index %d", method, path, key, i)
		}
		if err := json.Unmarshal(item, &result[i]); err != nil {
			return nil, fmt.Errorf("hexcore: decode %s %s item %d: %w", method, path, i, err)
		}
	}
	return result, nil
}

func requestOne[T any](ctx context.Context, c *Client, method, path string, body any, status int, key string) (*T, error) {
	items, err := requestList[T](ctx, c, method, path, body, status, key)
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("hexcore: decode %s %s: expected one %q item, got %d", method, path, key, len(items))
	}
	return &items[0], nil
}

func resourcePath(resource, id, suffix string) (string, error) {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, " /\\?#\t\r\n") {
		return "", errors.New("hexcore: resource ID must be a nonempty URL path segment")
	}
	return "/" + resource + "/" + url.PathEscape(id) + suffix, nil
}
