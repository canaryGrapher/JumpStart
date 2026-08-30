// Package github talks to the GitHub GraphQL API for Projects v2:
// discovering boards, reading their items and field definitions, and
// writing values back. REST is only used for the OAuth device flow and
// for creating issues, which GraphQL handles less conveniently.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	graphQLEndpoint = "https://api.github.com/graphql"
	restBase        = "https://api.github.com"
	userAgent       = "JumpStart"
)

// Client issues authenticated GraphQL requests. The zero value is not
// usable; build one with New.
type Client struct {
	token string
	http  *http.Client
}

// New returns a Client bound to a personal access or OAuth token.
func New(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Token reports the token the client authenticates with, so callers can
// pass it to helpers that need their own request.
func (c *Client) Token() string { return c.token }

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

// RateLimitError signals a 403/429 with a reset time, so the scheduler
// can back off instead of hammering the API.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("github rate limit reached, retry in %s", e.RetryAfter.Round(time.Second))
}

// Query runs a GraphQL document and unmarshals data into out.
func (c *Client) Query(ctx context.Context, query string, vars map[string]any, out any) error {
	if c.token == "" {
		return fmt.Errorf("not connected to GitHub")
	}
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("encoding query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("calling GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{RetryAfter: retryAfter(resp)}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("GitHub rejected the token, reconnect in Settings")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}

	var parsed graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		msgs := make([]string, 0, len(parsed.Errors))
		for _, e := range parsed.Errors {
			msgs = append(msgs, e.Message)
		}
		return fmt.Errorf("GitHub: %s", strings.Join(msgs, "; "))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(parsed.Data, out); err != nil {
		return fmt.Errorf("decoding data: %w", err)
	}
	return nil
}

// retryAfter reads the Retry-After or rate-limit reset headers, falling
// back to a minute when GitHub sends neither.
func retryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		var secs int
		if _, err := fmt.Sscanf(v, "%d", &secs); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		var epoch int64
		if _, err := fmt.Sscanf(v, "%d", &epoch); err == nil {
			if d := time.Until(time.Unix(epoch, 0)); d > 0 {
				return d
			}
		}
	}
	return time.Minute
}
