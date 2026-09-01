package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GitHub Apps issue user access tokens that expire (8 hours at the time
// of writing) together with a longer-lived refresh token (6 months).
// These endpoints are vars rather than consts so tests can point them at
// an httptest server.
var (
	deviceCodeURL  = "https://github.com/login/device/code"
	accessTokenURL = "https://github.com/login/oauth/access_token"
)

// RefreshSkew is how long before actual expiry a token is treated as
// stale. It absorbs clock drift and the round trip of the request the
// token is about to be used for.
const RefreshSkew = 5 * time.Minute

// ErrReauthRequired means no automatic recovery is possible any more:
// there is no refresh token, or GitHub refused the one we have. The user
// has to run the device flow again.
var ErrReauthRequired = errors.New("GitHub sign-in expired, reconnect in Settings")

// ErrUnauthorized is returned for a 401 from the API. It is a sentinel so
// callers can tell "this token is dead" apart from other API failures and
// try a refresh before giving up.
var ErrUnauthorized = errors.New("GitHub rejected the token, reconnect in Settings")

// TokenSet is everything the OAuth flows hand back. A zero ExpiresAt
// means the token does not expire, which is the case for personal access
// tokens and for OAuth apps that have not opted into expiring tokens.
type TokenSet struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken,omitempty"`
	ExpiresAt        time.Time `json:"expiresAt,omitempty"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt,omitempty"`
}

// StaticTokenSet wraps a non-expiring token, such as a pasted PAT.
func StaticTokenSet(token string) *TokenSet {
	return &TokenSet{AccessToken: strings.TrimSpace(token)}
}

// Expired reports whether the access token is at or past its expiry,
// with RefreshSkew of headroom. Non-expiring tokens are never expired.
func (t *TokenSet) Expired(now time.Time) bool {
	if t == nil || t.ExpiresAt.IsZero() {
		return false
	}
	return !now.Add(RefreshSkew).Before(t.ExpiresAt)
}

// Usable reports whether the access token has not actually expired yet,
// ignoring the refresh skew. A token inside the skew window is stale
// (Expired) but still usable, which is what makes an early refresh
// failure survivable.
func (t *TokenSet) Usable(now time.Time) bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	return t.ExpiresAt.IsZero() || now.Before(t.ExpiresAt)
}

// CanRefresh reports whether a refresh attempt is worth making: there is
// a refresh token and it has not itself lapsed.
func (t *TokenSet) CanRefresh(now time.Time) bool {
	if t == nil || t.RefreshToken == "" {
		return false
	}
	return t.RefreshExpiresAt.IsZero() || now.Before(t.RefreshExpiresAt)
}

// RefreshAccessToken exchanges a refresh token for a fresh token set.
// A refusal from GitHub (the refresh token was revoked, used, or has
// lapsed) comes back wrapped in ErrReauthRequired, because retrying it
// will never succeed.
func RefreshAccessToken(ctx context.Context, clientID, refreshToken string) (*TokenSet, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, ErrReauthRequired
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, ErrReauthRequired
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	var out tokenResponse
	if err := postForm(ctx, accessTokenURL, form, &out); err != nil {
		// A transport failure is not proof the grant is dead, so it stays
		// a plain error and the caller can try again on the next tick.
		return nil, err
	}
	switch out.Error {
	case "":
	case "bad_refresh_token", "invalid_grant", "unauthorized_client":
		return nil, fmt.Errorf("%w: %s", ErrReauthRequired, firstNonEmpty(out.ErrorDesc, out.Error))
	default:
		return nil, fmt.Errorf("GitHub: %s", firstNonEmpty(out.ErrorDesc, out.Error))
	}
	set := out.toTokenSet(time.Now())
	if set.AccessToken == "" {
		return nil, fmt.Errorf("%w: GitHub returned no access token", ErrReauthRequired)
	}
	return set, nil
}

// tokenResponse is the shared shape of the device-flow and refresh
// responses from https://github.com/login/oauth/access_token.
type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	Error                 string `json:"error"`
	ErrorDesc             string `json:"error_description"`
}

// toTokenSet converts the wire response's relative lifetimes into
// absolute instants, so a set stays meaningful after being written to
// the keychain and read back in a later run of the app.
func (r tokenResponse) toTokenSet(now time.Time) *TokenSet {
	set := &TokenSet{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
	}
	if r.ExpiresIn > 0 {
		set.ExpiresAt = now.Add(time.Duration(r.ExpiresIn) * time.Second)
	}
	if r.RefreshTokenExpiresIn > 0 {
		set.RefreshExpiresAt = now.Add(time.Duration(r.RefreshTokenExpiresIn) * time.Second)
	}
	return set
}

// MarshalTokenSet encodes a set for storage in the OS keychain.
func MarshalTokenSet(t *TokenSet) (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("encoding token: %w", err)
	}
	return string(b), nil
}

// UnmarshalTokenSet decodes a stored set. An empty string yields a nil
// set and no error, meaning "nothing stored".
func UnmarshalTokenSet(raw string) (*TokenSet, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var t TokenSet
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return nil, fmt.Errorf("decoding stored token: %w", err)
	}
	if t.AccessToken == "" {
		return nil, nil
	}
	return &t, nil
}
