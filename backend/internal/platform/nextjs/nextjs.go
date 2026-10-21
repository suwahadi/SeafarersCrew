// Package nextjs calls the on-demand revalidation endpoint of the public website.
package nextjs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SecretHeader carries the shared revalidation secret.
const SecretHeader = "X-Revalidate-Secret"

// Client posts cache tags to {baseURL}/api/revalidate.
type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

// NewClient builds a client; an empty baseURL disables revalidation.
func NewClient(baseURL, secret string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), secret: secret, http: &http.Client{Timeout: timeout}}
}

// Enabled reports whether a website URL is configured.
func (c *Client) Enabled() bool { return c.baseURL != "" }

// Revalidate asks the website to drop cached data for tags.
func (c *Client) Revalidate(ctx context.Context, tags []string) error {
	if !c.Enabled() || len(tags) == 0 {
		return nil
	}
	body, err := json.Marshal(map[string][]string{"tags": tags})
	if err != nil {
		return fmt.Errorf("encode revalidate body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/revalidate", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build revalidate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SecretHeader, c.secret)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("revalidate website: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("revalidate website: status %d", resp.StatusCode)
	}
	return nil
}
