package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPollDeviceFlowPending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
	}))
	defer srv.Close()

	prev := tokenURL
	tokenURL = srv.URL
	t.Cleanup(func() { tokenURL = prev })

	_, err := PollDeviceFlow(context.Background(), "client", "device")
	if !errors.Is(err, ErrAuthPending) {
		t.Fatalf("PollDeviceFlow() err = %v, want ErrAuthPending", err)
	}
}

func TestPollDeviceFlowSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"glpat-test"}`))
	}))
	defer srv.Close()

	prev := tokenURL
	tokenURL = srv.URL
	t.Cleanup(func() { tokenURL = prev })

	token, err := PollDeviceFlow(context.Background(), "client", "device")
	if err != nil {
		t.Fatalf("PollDeviceFlow() err = %v", err)
	}
	if token != "glpat-test" {
		t.Fatalf("token = %q, want glpat-test", token)
	}
}

func TestWhoami(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"username":"dev",
			"name":"Dev User",
			"avatar_url":"https://gitlab.com/avatar.png",
			"bio":"builder",
			"location":"Earth",
			"website_url":"https://example.com",
			"web_url":"https://gitlab.com/dev"
		}`))
	}))
	defer srv.Close()

	prev := userURL
	userURL = srv.URL
	t.Cleanup(func() { userURL = prev })

	user, err := Whoami(context.Background(), "good")
	if err != nil {
		t.Fatalf("Whoami() err = %v", err)
	}
	if user.Login != "dev" || user.Name != "Dev User" || user.ProfileURL != "https://gitlab.com/dev" {
		t.Fatalf("unexpected user: %+v", user)
	}
}
