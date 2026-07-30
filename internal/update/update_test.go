package update

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareOrdersPrereleasesBelowStable(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.0", "1.4.0-beta.1", 1},
		{"1.4.0-beta.1", "1.4.0", -1},
		{"1.4.0-beta.2", "1.4.0-beta.1", 1},
		{"1.4.0-beta", "1.4.0-beta.1", -1},
		{"v1.4.0", "1.4.0", 0},
		{"1.5.0-beta.1", "1.4.0", 1},
		{"1.4.0+build.7", "1.4.0", 0},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsBeta(t *testing.T) {
	for tag, want := range map[string]bool{
		"v1.4.0-beta":   true,
		"v1.4.0-beta.3": true,
		"v1.4.0":        false,
		"v1.4.0-rc.1":   false,
	} {
		if got := IsBeta(tag); got != want {
			t.Errorf("IsBeta(%q) = %v, want %v", tag, got, want)
		}
	}
}

const releasesJSON = `[
  {"tag_name":"v1.5.0-beta.2","prerelease":true,"name":"1.5.0 beta 2","body":"Beta notes","html_url":"u/beta"},
  {"tag_name":"v1.4.0","prerelease":false,"name":"1.4.0","body":"Stable notes","html_url":"u/stable"},
  {"tag_name":"v1.6.0","draft":true,"prerelease":false,"name":"draft","html_url":"u/draft"}
]`

func serveReleases(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(releasesJSON))
	}))
	prev := apiBase
	apiBase = srv.URL
	t.Cleanup(func() {
		apiBase = prev
		srv.Close()
	})
}

func TestCheckStableChannelSkipsBeta(t *testing.T) {
	serveReleases(t)
	info, err := Check("o", "r", "1.3.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestVersion != "1.4.0" || !info.Available || info.Prerelease {
		t.Fatalf("got %+v, want stable 1.4.0", info)
	}
	if info.ReleaseNotes != "Stable notes" || info.Channel != "stable" {
		t.Fatalf("got %+v, want stable notes and channel", info)
	}
}

func TestCheckBetaChannelOffersPrerelease(t *testing.T) {
	serveReleases(t)
	info, err := Check("o", "r", "1.4.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if info.LatestVersion != "1.5.0-beta.2" || !info.Available || !info.Prerelease {
		t.Fatalf("got %+v, want beta 1.5.0-beta.2", info)
	}
	if info.Channel != "beta" || info.ReleaseNotes != "Beta notes" {
		t.Fatalf("got %+v, want beta channel and notes", info)
	}
}

func TestCheckBetaChannelStaysOnNewerStable(t *testing.T) {
	serveReleases(t)
	info, err := Check("o", "r", "1.5.0-beta.2", true)
	if err != nil {
		t.Fatal(err)
	}
	if info.Available {
		t.Fatalf("got %+v, want no update when already on latest beta", info)
	}
}
