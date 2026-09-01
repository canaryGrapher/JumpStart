package main

import (
	"context"
	"errors"
	"time"

	"devdeck/internal/github"
	"devdeck/internal/secrets"
)

// refreshTimeout bounds a token refresh. It is short on purpose: the
// refresh sits in front of an API call the user is waiting on.
const refreshTimeout = 30 * time.Second

// errNotConnected is what every GitHub entry point returns when no token
// has ever been stored.
var errNotConnected = errors.New("not connected to GitHub, connect in Settings")

// refreshAccessToken is a var so tests can stand in for the network.
var refreshAccessToken = github.RefreshAccessToken

// ghTokenSource is the single place a GitHub access token comes from.
//
// GitHub Apps issue user tokens that expire after a few hours and hand
// back a refresh token alongside them. Callers should never read the
// keychain directly: they take a token from here, which refreshes just
// before expiry (and again if a request still comes back 401) and writes
// the new set back to the keychain.
func (a *App) ghTokenSource() github.TokenSource {
	return func(ctx context.Context, force bool) (string, error) {
		return a.ghAccessToken(ctx, force)
	}
}

// ghAccessToken returns a token that should be good for the next call.
// force skips the freshness check and refreshes unconditionally, which
// is what the client does after a 401.
//
// The whole body is serialized: a board sync fires several requests at
// once, and without the lock each would start its own refresh, and all
// but one of the resulting tokens would be thrown away.
func (a *App) ghAccessToken(ctx context.Context, force bool) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	st := a.gh()
	st.tokenMu.Lock()
	defer st.tokenMu.Unlock()

	set, err := loadGitHubTokenSet()
	if err != nil {
		return "", err
	}
	if set == nil {
		return "", errNotConnected
	}
	if !force && !set.Expired(time.Now()) {
		return set.AccessToken, nil
	}
	if !set.CanRefresh(time.Now()) {
		// A personal access token or a pre-refresh-support connection.
		// Nothing to refresh to, so hand back what we have and let the
		// API be the judge, unless the caller already knows it failed.
		if force {
			return "", github.ErrReauthRequired
		}
		return set.AccessToken, nil
	}

	refreshCtx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()

	refreshed, err := refreshAccessToken(refreshCtx, GitHubClientID, set.RefreshToken)
	if err != nil {
		// A refusal is terminal and the stored set is now dead weight,
		// but it is not deleted: Settings reads it to tell "expired,
		// sign in again" apart from "never connected".
		if !errors.Is(err, github.ErrReauthRequired) && !force && set.Usable(time.Now()) {
			// Network hiccup while merely topping up early. The current
			// token has not actually expired yet, so use it and try the
			// refresh again on the next call.
			return set.AccessToken, nil
		}
		return "", err
	}

	// GitHub rotates the refresh token on every use, but be tolerant of
	// a response that omits it rather than losing the one we have.
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = set.RefreshToken
		refreshed.RefreshExpiresAt = set.RefreshExpiresAt
	}
	if err := saveGitHubTokenSet(refreshed); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

// ghHasToken reports whether any GitHub token is stored, without making
// a network call. Used by panes that only need to show connected state.
func ghHasToken() bool {
	set, err := loadGitHubTokenSet()
	return err == nil && set != nil
}

// loadGitHubTokenSet reads the stored token set, falling back to the
// bare-token key for connections made before refresh support existed
// and for pasted personal access tokens.
func loadGitHubTokenSet() (*github.TokenSet, error) {
	raw, err := secrets.GetToken(secrets.Service, secrets.KeyGitHubTokenSet)
	if err != nil {
		return nil, err
	}
	if set, perr := github.UnmarshalTokenSet(raw); perr == nil && set != nil {
		return set, nil
	}
	token, err := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, nil
	}
	return github.StaticTokenSet(token), nil
}

// saveGitHubTokenSet writes both keychain entries. The bare-token key is
// kept in step so any code path that still reads it (git push, release
// publishing) gets the current token rather than a stale one.
func saveGitHubTokenSet(set *github.TokenSet) error {
	if set == nil || set.AccessToken == "" {
		return errors.New("no token to save")
	}
	encoded, err := github.MarshalTokenSet(set)
	if err != nil {
		return err
	}
	if err := secrets.SaveToken(secrets.Service, secrets.KeyGitHubToken, set.AccessToken); err != nil {
		return err
	}
	return secrets.SaveToken(secrets.Service, secrets.KeyGitHubTokenSet, encoded)
}

// deleteGitHubTokenSet clears both keychain entries on disconnect.
func deleteGitHubTokenSet() error {
	if err := secrets.DeleteToken(secrets.Service, secrets.KeyGitHubTokenSet); err != nil {
		return err
	}
	return secrets.DeleteToken(secrets.Service, secrets.KeyGitHubToken)
}
