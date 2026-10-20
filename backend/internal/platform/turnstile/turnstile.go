// Package turnstile verifies Cloudflare Turnstile tokens.
package turnstile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

// DefaultURL is the Turnstile siteverify endpoint.
const DefaultURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Result is the siteverify response.
type Result struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
}

// HasError reports whether code is among the error codes.
func (r Result) HasError(code string) bool { return slices.Contains(r.ErrorCodes, code) }

// Client calls siteverify.
type Client struct {
	url  string
	http *http.Client
}

// New returns a Client for the siteverify url with a request timeout.
func New(verifyURL string, timeout time.Duration) *Client {
	return &Client{url: verifyURL, http: &http.Client{Timeout: timeout}}
}

// Verify checks token with secret; ip is optional.
func (c *Client) Verify(ctx context.Context, secret, token, ip string) (Result, error) {
	form := url.Values{"secret": {secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, fmt.Errorf("build turnstile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.http.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("turnstile request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if err != nil {
		return Result{}, fmt.Errorf("read turnstile response: %w", err)
	}
	var out Result
	if err := json.Unmarshal(data, &out); err != nil {
		return Result{}, fmt.Errorf("decode turnstile response (status %d): %w", res.StatusCode, err)
	}
	return out, nil
}
