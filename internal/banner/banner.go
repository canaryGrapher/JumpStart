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

// DefaultURL is where the banner config lives. It is fixed; there is no
// user override.
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

// Fetch downloads and validates the banner config from DefaultURL. It
// returns the list of enabled, currently-active banners, in the order
// published. An empty slice means nothing should be shown. The config may
// be a JSON array of banners or a single banner object.
func Fetch() ([]Banner, error) {
	resp, err := httpClient.Get(DefaultURL)
	if err != nil {
		return nil, fmt.Errorf("banner fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // no config published: nothing to show
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("banner fetch failed: server returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("banner fetch failed: %w", err)
	}
	all, err := parse(body)
	if err != nil {
		return nil, err
	}
	active := make([]Banner, 0, len(all))
	for _, b := range all {
		if b.Enabled && b.ID != "" && activeNow(b) {
			active = append(active, b)
		}
	}
	return active, nil
}

// parse accepts either a JSON array of banners or a single banner object.
func parse(body []byte) ([]Banner, error) {
	var list []Banner
	if err := json.Unmarshal(body, &list); err == nil {
		return list, nil
	}
	var one Banner
	if err := json.Unmarshal(body, &one); err != nil {
		return nil, fmt.Errorf("banner config is not valid JSON: %w", err)
	}
	return []Banner{one}, nil
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
