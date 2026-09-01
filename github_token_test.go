package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"devdeck/internal/github"
	"devdeck/internal/secrets"
)

// useMockKeychain swaps in go-keyring's in-memory backend so these tests
// never touch the real login keychain.
func useMockKeychain(t *testing.T) {
	t.Helper()
	keyring.MockInit()
	t.Cleanup(func() {
		_ = secrets.DeleteToken(secrets.Service, secrets.KeyGitHubToken)
		_ = secrets.DeleteToken(secrets.Service, secrets.KeyGitHubTokenSet)
	})
}

// stubRefresh replaces the network call for the duration of a test.
func stubRefresh(t *testing.T, fn func(ctx context.Context, clientID, refreshToken string) (*github.TokenSet, error)) {
	t.Helper()
	prev := refreshAccessToken
	refreshAccessToken = fn
	t.Cleanup(func() { refreshAccessToken = prev })
}

func TestGhAccessTokenReturnsFreshTokenUntouched(t *testing.T) {
	useMockKeychain(t)
	stubRefresh(t, func(context.Context, string, string) (*github.TokenSet, error) {
		t.Fatal("a token with hours left should not be refreshed")
		return nil, nil
	})

	if err := saveGitHubTokenSet(&github.TokenSet{
		AccessToken:  "gho_live",
		RefreshToken: "ghr_1",
		ExpiresAt:    time.Now().Add(4 * time.Hour),
	}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	app := &App{}
	got, err := app.ghAccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("ghAccessToken: %v", err)
	}
	if got != "gho_live" {
		t.Fatalf("token = %q, want the stored one", got)
	}
}

func TestGhAccessTokenRefreshesBeforeExpiry(t *testing.T) {
	useMockKeychain(t)
	var refreshedWith string
	stubRefresh(t, func(_ context.Context, _, refresh string) (*github.TokenSet, error) {
		refreshedWith = refresh
		return &github.TokenSet{
			AccessToken:  "gho_fresh",
			RefreshToken: "ghr_rotated",
			ExpiresAt:    time.Now().Add(8 * time.Hour),
		}, nil
	})

	// Inside the skew window: still technically valid, already stale.
	if err := saveGitHubTokenSet(&github.TokenSet{
		AccessToken:  "gho_stale",
		RefreshToken: "ghr_old",
		ExpiresAt:    time.Now().Add(time.Minute),
	}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	app := &App{}
	got, err := app.ghAccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("ghAccessToken: %v", err)
	}
	if got != "gho_fresh" {
		t.Fatalf("token = %q, want the refreshed one", got)
	}
	if refreshedWith != "ghr_old" {
		t.Fatalf("refreshed with %q", refreshedWith)
	}

	// The rotated set must be persisted, or the next launch of the app
	// would try the old refresh token and be told to sign in again.
	stored, err := loadGitHubTokenSet()
	if err != nil {
		t.Fatalf("loadGitHubTokenSet: %v", err)
	}
	if stored.AccessToken != "gho_fresh" || stored.RefreshToken != "ghr_rotated" {
		t.Fatalf("stored set not updated: %+v", stored)
	}
	// The bare key is what git push and release publishing read.
	bare, _ := secrets.GetToken(secrets.Service, secrets.KeyGitHubToken)
	if bare != "gho_fresh" {
		t.Fatalf("bare token key = %q, want it kept in step", bare)
	}
}

func TestGhAccessTokenKeepsRotatedTokenWhenResponseOmitsIt(t *testing.T) {
	useMockKeychain(t)
	stubRefresh(t, func(context.Context, string, string) (*github.TokenSet, error) {
		return &github.TokenSet{AccessToken: "gho_fresh", ExpiresAt: time.Now().Add(8 * time.Hour)}, nil
	})
	if err := saveGitHubTokenSet(&github.TokenSet{
		AccessToken:  "gho_stale",
		RefreshToken: "ghr_keep",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	if _, err := (&App{}).ghAccessToken(context.Background(), false); err != nil {
		t.Fatalf("ghAccessToken: %v", err)
	}
	stored, _ := loadGitHubTokenSet()
	if stored.RefreshToken != "ghr_keep" {
		t.Fatalf("refresh token = %q, want the existing one retained", stored.RefreshToken)
	}
}

func TestGhAccessTokenSurvivesNetworkBlipWhileToppingUp(t *testing.T) {
	useMockKeychain(t)
	stubRefresh(t, func(context.Context, string, string) (*github.TokenSet, error) {
		return nil, errors.New("dial tcp: no route to host")
	})
	if err := saveGitHubTokenSet(&github.TokenSet{
		AccessToken:  "gho_still_good",
		RefreshToken: "ghr_1",
		ExpiresAt:    time.Now().Add(2 * time.Minute), // stale, not yet dead
	}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	got, err := (&App{}).ghAccessToken(context.Background(), false)
	if err != nil {
		t.Fatalf("a transient refresh failure must not break a usable token: %v", err)
	}
	if got != "gho_still_good" {
		t.Fatalf("token = %q", got)
	}
}

func TestGhAccessTokenReportsReauthWhenRefreshRefused(t *testing.T) {
	useMockKeychain(t)
	stubRefresh(t, func(context.Context, string, string) (*github.TokenSet, error) {
		return nil, github.ErrReauthRequired
	})
	if err := saveGitHubTokenSet(&github.TokenSet{
		AccessToken:  "gho_dead",
		RefreshToken: "ghr_dead",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	if _, err := (&App{}).ghAccessToken(context.Background(), false); !errors.Is(err, github.ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
}

func TestGhAccessTokenPersonalAccessToken(t *testing.T) {
	useMockKeychain(t)
	stubRefresh(t, func(context.Context, string, string) (*github.TokenSet, error) {
		t.Fatal("a PAT has nothing to refresh")
		return nil, nil
	})
	if err := saveGitHubTokenSet(github.StaticTokenSet("ghp_pat")); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}

	app := &App{}
	got, err := app.ghAccessToken(context.Background(), false)
	if err != nil || got != "ghp_pat" {
		t.Fatalf("ghAccessToken = (%q, %v)", got, err)
	}
	// Forcing a refresh on a PAT cannot produce anything new.
	if _, err := app.ghAccessToken(context.Background(), true); !errors.Is(err, github.ErrReauthRequired) {
		t.Fatalf("forced err = %v, want ErrReauthRequired", err)
	}
}

func TestLoadGitHubTokenSetFallsBackToLegacyKey(t *testing.T) {
	useMockKeychain(t)
	// A connection made by an older build: bare key only, no set.
	if err := secrets.SaveToken(secrets.Service, secrets.KeyGitHubToken, "gho_legacy"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	set, err := loadGitHubTokenSet()
	if err != nil {
		t.Fatalf("loadGitHubTokenSet: %v", err)
	}
	if set == nil || set.AccessToken != "gho_legacy" {
		t.Fatalf("set = %+v, want the legacy token adopted", set)
	}
	if !ghHasToken() {
		t.Fatal("ghHasToken() = false for a legacy connection")
	}
}

func TestLoadGitHubTokenSetEmpty(t *testing.T) {
	useMockKeychain(t)
	set, err := loadGitHubTokenSet()
	if err != nil || set != nil {
		t.Fatalf("loadGitHubTokenSet = (%v, %v), want (nil, nil)", set, err)
	}
	if ghHasToken() {
		t.Fatal("ghHasToken() = true with nothing stored")
	}
	if _, err := (&App{}).ghAccessToken(context.Background(), false); !errors.Is(err, errNotConnected) {
		t.Fatalf("err = %v, want errNotConnected", err)
	}
}

func TestDeleteGitHubTokenSetClearsBothKeys(t *testing.T) {
	useMockKeychain(t)
	if err := saveGitHubTokenSet(&github.TokenSet{AccessToken: "gho_1", RefreshToken: "ghr_1"}); err != nil {
		t.Fatalf("saveGitHubTokenSet: %v", err)
	}
	if err := deleteGitHubTokenSet(); err != nil {
		t.Fatalf("deleteGitHubTokenSet: %v", err)
	}
	if ghHasToken() {
		t.Fatal("a token survived disconnect")
	}
	raw, _ := secrets.GetToken(secrets.Service, secrets.KeyGitHubTokenSet)
	if raw != "" {
		t.Fatalf("token set survived disconnect: %q", raw)
	}
}
