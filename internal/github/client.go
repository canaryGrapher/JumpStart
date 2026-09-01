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
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	restBase  = "https://api.github.com"
	userAgent = "JumpStart"
)

// graphQLEndpointURL is a var rather than a const so tests can point it
// at an httptest server.
var graphQLEndpointURL = "https://api.github.com/graphql"

// Client issues authenticated GraphQL requests. The zero value is not
// usable; build one with New or NewWithSource.
type Client struct {
	src  TokenSource
	http *http.Client
}

// New returns a Client bound to a fixed personal access or OAuth token.
// Prefer NewWithSource anywhere the token can expire.
func New(token string) *Client {
	return NewWithSource(StaticSource(token))
}

// NewWithSource returns a Client that asks src for a token before every
// request, so an expiring token is refreshed without the caller caring.
func NewWithSource(src TokenSource) *Client {
	return &Client{
		src:  src,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// Token reports the current access token, so callers can pass it to
// helpers that make their own request (git push, for one).
func (c *Client) Token(ctx context.Context) (string, error) {
	return c.token(ctx, false)
}

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
	body, err := json.Marshal(graphQLRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("encoding query: %w", err)
	}

	resp, err := c.do(ctx, func(ctx context.Context) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphQLEndpointURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{RetryAfter: retryAfter(resp)}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}

	var parsed graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	// GraphQL allows a response to carry both data and errors when only
	// part of a query failed to resolve — queryOwnerProjects, for
	// example, asks for both `user(login:)` and `organization(login:)`
	// for the same name, and exactly one of them always errors with
	// "Could not resolve to a User/Organization ..." while the other
	// resolves fine. Callers that read structured data (out != nil)
	// already null-check those fields, so as long as usable data came
	// back it takes priority over a partial-field error; a request that
	// doesn't want data back (out == nil, e.g. a mutation) has no data
	// to fall back on, so any error there is still treated as fatal.
	hasData := out != nil && len(parsed.Data) > 0 && string(parsed.Data) != "null"

	if len(parsed.Errors) > 0 && !hasData {
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

// restJSON issues an authenticated REST call against the GitHub API
// (as opposed to the GraphQL endpoint Query uses) and decodes the JSON
// response into out. It exists for the handful of operations GraphQL
// does not cover well, such as creating a repository under a specific
// owner. body is marshaled as the request payload when non-nil; out may
// be nil when the response body is not needed.
func (c *Client) restJSON(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		payload = b
	}

	resp, err := c.do(ctx, func(ctx context.Context) (*http.Request, error) {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, restBase+path, reader)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return &RateLimitError{RetryAfter: retryAfter(resp)}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 300 {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("GitHub: %s", apiErr.Message)
		}
		return fmt.Errorf("GitHub returned %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
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
