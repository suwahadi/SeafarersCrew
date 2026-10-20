// Package brevo sends transactional email through the Brevo API.
package brevo

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the Brevo API root.
const DefaultBaseURL = "https://api.brevo.com/v3"

// Address is a named email address.
type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment is a file sent with a message.
type Attachment struct {
	Name    string
	Content []byte
}

// Message is one transactional email.
type Message struct {
	APIKey      string
	Sender      Address
	To          []Address
	CC          []Address
	ReplyTo     *Address
	Subject     string
	HTML        string
	Attachments []Attachment
}

// Error is a failed API call; Retryable is false for client errors other than rate limiting.
type Error struct {
	Status    int
	Message   string
	Retryable bool
}

func (e *Error) Error() string {
	if e.Status == 0 {
		return "brevo: " + e.Message
	}
	return fmt.Sprintf("brevo: status %d: %s", e.Status, e.Message)
}

// IsRetryable reports whether err may succeed when retried.
func IsRetryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Retryable
	}
	return true
}

// Client calls the Brevo API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client for baseURL with a request timeout.
func New(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

type attachmentBody struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type sendBody struct {
	Sender      Address          `json:"sender"`
	To          []Address        `json:"to"`
	CC          []Address        `json:"cc,omitempty"`
	ReplyTo     *Address         `json:"replyTo,omitempty"`
	Subject     string           `json:"subject"`
	HTMLContent string           `json:"htmlContent"`
	Attachment  []attachmentBody `json:"attachment,omitempty"`
}

// Send delivers m and returns the Brevo message ID.
func (c *Client) Send(ctx context.Context, m Message) (string, error) {
	if m.APIKey == "" {
		return "", &Error{Message: "API key is not configured"}
	}
	body := sendBody{Sender: m.Sender, To: m.To, CC: m.CC, ReplyTo: m.ReplyTo, Subject: m.Subject, HTMLContent: m.HTML}
	for _, a := range m.Attachments {
		body.Attachment = append(body.Attachment, attachmentBody{Name: a.Name, Content: base64.StdEncoding.EncodeToString(a.Content)})
	}
	var out struct {
		MessageID string `json:"messageId"`
	}
	if err := c.do(ctx, http.MethodPost, "/smtp/email", m.APIKey, body, &out); err != nil {
		return "", err
	}
	return out.MessageID, nil
}

// Account returns the email of the account owning apiKey, verifying the key works.
func (c *Client) Account(ctx context.Context, apiKey string) (string, error) {
	if apiKey == "" {
		return "", &Error{Message: "API key is not configured"}
	}
	var out struct {
		Email string `json:"email"`
	}
	if err := c.do(ctx, http.MethodGet, "/account", apiKey, nil, &out); err != nil {
		return "", err
	}
	return out.Email, nil
}

func (c *Client) do(ctx context.Context, method, path, apiKey string, in, out any) error {
	var reader io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return &Error{Message: "encode request: " + err.Error()}
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return &Error{Message: "build request: " + err.Error()}
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return &Error{Message: "request failed: " + err.Error(), Retryable: true}
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return &Error{Status: res.StatusCode, Message: "read response: " + err.Error(), Retryable: true}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if out == nil || len(data) == 0 {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return &Error{Status: res.StatusCode, Message: "decode response: " + err.Error()}
		}
		return nil
	}
	return &Error{
		Status:    res.StatusCode,
		Message:   errorMessage(data),
		Retryable: res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500,
	}
}

func errorMessage(data []byte) string {
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(data, &body) == nil && body.Message != "" {
		if body.Code != "" {
			return body.Code + ": " + body.Message
		}
		return body.Message
	}
	msg := strings.TrimSpace(string(data))
	if len(msg) > 200 {
		msg = msg[:200]
	}
	if msg == "" {
		return "empty response"
	}
	return msg
}
