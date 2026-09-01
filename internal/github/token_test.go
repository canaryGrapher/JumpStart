package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// withTokenEndpoint points the OAuth endpoints at a test server for the
// duration of a test.
func withTokenEndpoint(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	prev := accessTokenURL
	accessTokenURL = srv.URL
	t.Cleanup(func() {
		accessTokenURL = prev
		srv.Close()
	})
}

func TestTokenSetExpired(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		set  *TokenSet
		want bool
	}{
		{"nil set", nil, false},
		{"no expiry means a PAT, never expires", &TokenSet{AccessToken: "t"}, false},
		{"an hour left", &TokenSet{ExpiresAt: now.Add(time.Hour)}, false},
		{"inside the skew window", &TokenSet{ExpiresAt: now.Add(RefreshSkew - time.Minute)}, true},
		{"already past", &TokenSet{ExpiresAt: now.Add(-time.Minute)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.set.Expired(now); got != tc.want {
				t.Fatalf("Expired() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTokenSetCanRefresh(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		set  *TokenSet
		want bool
	}{
		{"nil set", nil, false},
		{"no refresh token", &TokenSet{AccessToken: "t"}, false},
		{"refresh token with no stated expiry", &TokenSet{RefreshToken: "r"}, true},
		{"refresh token still valid", &TokenSet{RefreshToken: "r", RefreshExpiresAt: now.Add(time.Hour)}, true},
		{"refresh token lapsed", &TokenSet{RefreshToken: "r", RefreshExpiresAt: now.Add(-time.Hour)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.set.CanRefresh(now); got != tc.want {
				t.Fatalf("CanRefresh() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestToTokenSetConvertsLifetimes(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	set := tokenResponse{
		AccessToken:           "gho_new",
		RefreshToken:          "ghr_new",
		ExpiresIn:             28800,    // 8 hours, GitHub's user-token default
		RefreshTokenExpiresIn: 15811200, // 6 months
	}.toTokenSet(now)

	if set.AccessToken != "gho_new" || set.RefreshToken != "ghr_new" {
		t.Fatalf("tokens not carried over: %+v", set)
	}
	if want := now.Add(8 * time.Hour); !set.ExpiresAt.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", set.ExpiresAt, want)
	}
	if !set.RefreshExpiresAt.After(now.Add(180 * 24 * time.Hour).Add(-time.Hour)) {
		t.Fatalf("RefreshExpiresAt = %v, want roughly six months out", set.RefreshExpiresAt)
	}
}

func TestToTokenSetLeavesNonExpiringTokenZero(t *testing.T) {
	set := tokenResponse{AccessToken: "ghp_pat"}.toTokenSet(time.Now())
	if !set.ExpiresAt.IsZero() || !set.RefreshExpiresAt.IsZero() {
		t.Fatalf("expiries should stay zero for a non-expiring token: %+v", set)
	}
}

func TestRefreshAccessToken(t *testing.T) {
	var got url.Values
	withTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"gho_fresh","refresh_token":"ghr_rotated","expires_in":28800,"refresh_token_expires_in":15811200}`))
	})

	set, err := RefreshAccessToken(context.Background(), "Ov23test", "ghr_old")
	if err != nil {
		t.Fatalf("RefreshAccessToken: %v", err)
	}
	if set.AccessToken != "gho_fresh" {
		t.Fatalf("access token = %q", set.AccessToken)
	}
	if set.RefreshToken != "ghr_rotated" {
		t.Fatalf("rotated refresh token not kept: %q", set.RefreshToken)
	}
	if set.ExpiresAt.IsZero() {
		t.Fatal("expiry not recorded, the next call would not know to refresh")
	}
	if got.Get("grant_type") != "refresh_token" {
		t.Fatalf("grant_type = %q", got.Get("grant_type"))
	}
	if got.Get("refresh_token") != "ghr_old" || got.Get("client_id") != "Ov23test" {
		t.Fatalf("unexpected form: %v", got)
	}
}

func TestRefreshAccessTokenRefusalIsTerminal(t *testing.T) {
	withTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"bad_refresh_token","error_description":"The refresh token passed is incorrect or expired."}`))
	})

	_, err := RefreshAccessToken(context.Background(), "Ov23test", "ghr_dead")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
}

func TestRefreshAccessTokenWithoutRefreshTokenNeedsReauth(t *testing.T) {
	if _, err := RefreshAccessToken(context.Background(), "Ov23test", ""); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
	if _, err := RefreshAccessToken(context.Background(), "", "ghr"); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want ErrReauthRequired", err)
	}
}

func TestPollDeviceFlowKeepsRefreshToken(t *testing.T) {
	withTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"gho_1","refresh_token":"ghr_1","expires_in":28800,"refresh_token_expires_in":15811200}`))
	})

	set, err := PollDeviceFlow(context.Background(), "Ov23test", "device")
	if err != nil {
		t.Fatalf("PollDeviceFlow: %v", err)
	}
	// The bug this whole change fixes: the refresh token used to be
	// dropped here, leaving nothing to renew the 8-hour access token.
	if set.RefreshToken != "ghr_1" {
		t.Fatalf("refresh token = %q, want it kept", set.RefreshToken)
	}
	if set.ExpiresAt.IsZero() {
		t.Fatal("expiry not recorded")
	}
}

func TestPollDeviceFlowPending(t *testing.T) {
	withTokenEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	})

	if _, err := PollDeviceFlow(context.Background(), "Ov23test", "device"); !errors.Is(err, ErrAuthPending) {
		t.Fatalf("err = %v, want ErrAuthPending", err)
	}
}

func TestTokenSetRoundTrip(t *testing.T) {
	in := &TokenSet{
		AccessToken:      "gho_1",
		RefreshToken:     "ghr_1",
		ExpiresAt:        time.Now().Add(8 * time.Hour).Round(time.Second).UTC(),
		RefreshExpiresAt: time.Now().Add(180 * 24 * time.Hour).Round(time.Second).UTC(),
	}
	raw, err := MarshalTokenSet(in)
	if err != nil {
		t.Fatalf("MarshalTokenSet: %v", err)
	}
	out, err := UnmarshalTokenSet(raw)
	if err != nil {
		t.Fatalf("UnmarshalTokenSet: %v", err)
	}
	if out.AccessToken != in.AccessToken || out.RefreshToken != in.RefreshToken {
		t.Fatalf("tokens changed: %+v", out)
	}
	if !out.ExpiresAt.Equal(in.ExpiresAt) || !out.RefreshExpiresAt.Equal(in.RefreshExpiresAt) {
		t.Fatalf("expiries changed: %+v", out)
	}
}

func TestUnmarshalTokenSetEmptyAndJunk(t *testing.T) {
	set, err := UnmarshalTokenSet("")
	if err != nil || set != nil {
		t.Fatalf("empty = (%v, %v), want (nil, nil)", set, err)
	}
	if _, err := UnmarshalTokenSet("not json"); err == nil {
		t.Fatal("expected an error for malformed stored data")
	}
	// A set with no access token is as good as nothing stored.
	set, err = UnmarshalTokenSet(`{"refreshToken":"ghr"}`)
	if err != nil || set != nil {
		t.Fatalf("tokenless set = (%v, %v), want (nil, nil)", set, err)
	}
}
