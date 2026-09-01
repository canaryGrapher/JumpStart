package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// TokenSource hands the client a usable access token. It is called
// before every request, so an implementation that refreshes expiring
// tokens keeps every call site current without any of them knowing.
//
// force asks for a token that is definitely not the one that just came
// back 401: implementations should refresh unconditionally rather than
// returning what they have cached. A source that cannot do better than
// its current token returns ErrReauthRequired for force, which stops the
// client from retrying a request that would only fail again.
type TokenSource func(ctx context.Context, force bool) (string, error)

// StaticSource wraps a fixed token, such as a pasted personal access
// token or a test fixture. It has nothing to refresh to.
func StaticSource(token string) TokenSource {
	token = strings.TrimSpace(token)
	return func(_ context.Context, force bool) (string, error) {
		if token == "" {
			return "", fmt.Errorf("not connected to GitHub")
		}
		if force {
			return "", ErrReauthRequired
		}
		return token, nil
	}
}

// buildRequest makes one attempt's request. The client calls it again
// for a retry rather than reusing the first request, because a request
// body is consumed once.
type buildRequest func(ctx context.Context) (*http.Request, error)

// do sends a request, and on a 401 refreshes the token once and sends it
// again. The retry is what turns an expired GitHub App user token into a
// pause of a few hundred milliseconds instead of the "reconnect in
// Settings" message the user would otherwise see every eight hours.
func (c *Client) do(ctx context.Context, build buildRequest) (*http.Response, error) {
	resp, err := c.attempt(ctx, build, false)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	resp.Body.Close()

	retried, rerr := c.attempt(ctx, build, true)
	if rerr != nil {
		// The refresh itself failed. Report the 401 the caller is
		// actually facing, keeping the refresh failure as context.
		return nil, fmt.Errorf("%w (%v)", ErrUnauthorized, rerr)
	}
	return retried, nil
}

// attempt builds, authorizes and sends a single request.
func (c *Client) attempt(ctx context.Context, build buildRequest, force bool) (*http.Response, error) {
	token, err := c.token(ctx, force)
	if err != nil {
		return nil, err
	}
	req, err := build(ctx)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling GitHub: %w", err)
	}
	return resp, nil
}

// token resolves the current access token from the source.
func (c *Client) token(ctx context.Context, force bool) (string, error) {
	if c.src == nil {
		return "", fmt.Errorf("not connected to GitHub")
	}
	return c.src(ctx, force)
}
