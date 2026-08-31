// Package analytics reports anonymous product usage to Google Analytics 4
// via the Measurement Protocol from the Go process (not the Wails webview).
//
// Three rules hold everywhere in this package:
//
//  1. Nothing is sent unless the user has analytics enabled. Track is a
//     no-op otherwise, and a nil *Client is safe to call.
//  2. No identifying content ever leaves the machine. Not filesystem paths,
//     project or process names, repo URLs, branch names, commit messages,
//     env vars, script bodies, chat text, prompts, model output, or tokens.
//     Callers pass derived shapes (counts, depths, buckets, bounded enums)
//     and Sanitize enforces that at the boundary.
//  3. Track never blocks and never returns an error. It is called from Wails
//     bindings on the UI's critical path, so it hands off to a buffered
//     channel and drops events rather than slowing the app down.
package analytics

import (
	"os"
	"path/filepath"
	"time"
)

// DefaultCollectURL is Google's GA4 Measurement Protocol collect endpoint.
const DefaultCollectURL = "https://www.google-analytics.com/mp/collect"

// Options configures a Client.
type Options struct {
	// MeasurementID is the GA4 Measurement ID (G-…). Empty disables the client.
	MeasurementID string
	// APISecret is the GA4 Measurement Protocol API secret. Empty disables the client.
	APISecret string
	// Version is the running build's version, e.g. "1.4.2" or "dev".
	Version string
	// Channel is the update channel: "stable" or "beta".
	Channel string
	// Dir is the app data directory (~/.jumpstart). Empty resolves it.
	Dir string
	// CollectURL overrides the GA4 MP endpoint for tests. Empty uses DefaultCollectURL.
	CollectURL string
}

// DataDir returns the app's data directory, creating it if needed. It
// mirrors internal/store so the install ID and offline queue sit alongside
// config.json rather than in a second location.
func DataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.TempDir()
	}
	dir := filepath.Join(home, ".jumpstart")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// Tuning constants. The batch is small and the interval short because
// desktop sessions are frequently killed without a clean shutdown.
const (
	bufferSize    = 512
	batchSize     = 20
	flushInterval = 15 * time.Second
	sendTimeout   = 10 * time.Second
	queueMax      = 5000
)
