package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// withGraphQLEndpoint points the GraphQL endpoint at a test server.
func withGraphQLEndpoint(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	prev := graphQLEndpointURL
	graphQLEndpointURL = srv.URL
	t.Cleanup(func() {
		graphQLEndpointURL = prev
		srv.Close()
	})
	return srv
}

func TestQueryRetriesOnceAfterRefresh(t *testing.T) {
	var calls int32
	var seen []string
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("retry sent an empty body; the request was not rebuilt")
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"yash"}}}`))
	})

	var forced int32
	client := NewWithSource(func(_ context.Context, force bool) (string, error) {
		if force {
			atomic.AddInt32(&forced, 1)
			return "gho_fresh", nil
		}
		return "gho_stale", nil
	})

	viewer, err := client.Whoami(context.Background())
	if err != nil {
		t.Fatalf("Whoami after refresh: %v", err)
	}
	if viewer.Login != "yash" {
		t.Fatalf("login = %q", viewer.Login)
	}
	if calls != 2 {
		t.Fatalf("request count = %d, want 2 (original plus retry)", calls)
	}
	if forced != 1 {
		t.Fatalf("forced refreshes = %d, want exactly 1", forced)
	}
	if seen[0] != "Bearer gho_stale" || seen[1] != "Bearer gho_fresh" {
		t.Fatalf("retry did not use the refreshed token: %v", seen)
	}
}

func TestQueryDoesNotRetryTwice(t *testing.T) {
	var calls int32
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
	})

	client := NewWithSource(func(_ context.Context, force bool) (string, error) {
		return "gho_any", nil
	})

	err := client.Query(context.Background(), queryViewer, nil, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if calls != 2 {
		t.Fatalf("request count = %d, want 2; a token GitHub keeps rejecting must not loop", calls)
	}
}

func TestStaticSourceCannotRefresh(t *testing.T) {
	var calls int32
	withGraphQLEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
	})

	// A pasted PAT has nothing to refresh to, so the 401 surfaces at once
	// rather than costing a second round trip.
	err := New("ghp_pat").Query(context.Background(), queryViewer, nil, nil)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
	if calls != 1 {
		t.Fatalf("request count = %d, want 1", calls)
	}
}

func TestEmptyTokenIsNotConnected(t *testing.T) {
	err := New("").Query(context.Background(), queryViewer, nil, nil)
	if err == nil {
		t.Fatal("expected an error with no token stored")
	}
}
