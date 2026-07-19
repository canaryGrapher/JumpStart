// Package banner fetches a remote overlay-banner (announcement/ad)
// configuration so the app can display promotions without shipping a new
// build. The config is a small JSON file the maintainer hosts remotely,
// by default in the app's GitHub repository via raw.githubusercontent.com.
package banner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultURL is where the banner config lives unless the user overrides
// it in Preferences.
const DefaultURL = "https://raw.githubusercontent.com/canaryGrapher/JumpStart/main/social/banner.json"

// Banner is the remote overlay banner definition. An empty or disabled
// banner means nothing is shown.
type Banner struct {
	// ID lets the frontend remember dismissals per banner.
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
	Title   string `json:"title"`
	Message string `json:"message"`
	// LinkURL/LinkText render an optional call-to-action button.
	LinkURL  string `json:"linkUrl"`
	LinkText string `json:"linkText"`
	// ImageURL renders an optional image above the text.
	ImageURL string `json:"imageUrl"`
	// Style is "info", "promo", or "warning"; the frontend maps it to a look.
	Style string `json:"style"`
	// StartsAt/EndsAt (RFC3339) bound when the banner is shown. Empty means
	// unbounded.
	StartsAt string `json:"startsAt"`
	EndsAt   string `json:"endsAt"`
}

const maxBody = 64 << 10 // 64 KiB is plenty for a banner config.

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Fetch downloads and validates the banner config at url. It returns a
// zero-value Banner (Enabled=false) when no banner should be shown.
func Fetch(url string) (Banner, error) {
	if url == "" {
		url = DefaultURL
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return Banner{}, fmt.Errorf("banner fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return Banner{}, nil // no config published: nothing to show
	}
	if resp.StatusCode != http.StatusOK {
		return Banner{}, fmt.Errorf("banner fetch failed: server returned %s", resp.Status)
	}
	var b Banner
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&b); err != nil {
		return Banner{}, fmt.Errorf("banner config is not valid JSON: %w", err)
	}
	if !b.Enabled || !activeNow(b) {
		return Banner{}, nil
	}
	return b, nil
}

func activeNow(b Banner) bool {
	now := time.Now()
	if t, err := time.Parse(time.RFC3339, b.StartsAt); b.StartsAt != "" && err == nil && now.Before(t) {
		return false
	}
	if t, err := time.Parse(time.RFC3339, b.EndsAt); b.EndsAt != "" && err == nil && now.After(t) {
		return false
	}
	return true
}
