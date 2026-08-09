package main

import (
	"time"

	"devdeck/internal/analytics"
)

// PostHogAPIKey is the PostHog project API key. Release builds inject it the
// same way Version is injected:
//
//	-ldflags "-X main.PostHogAPIKey=phc_xxx"
//
// Local and dev builds leave it empty, which makes the whole analytics layer
// an unconditional no-op. A PostHog project key is write-only and safe to
// ship in a binary; it cannot read data back out.
//
// PostHogHost selects the ingestion region. Empty means PostHog Cloud US.
var (
	PostHogAPIKey = ""
	PostHogHost   = ""
)

// initAnalytics builds the client for this session. It runs during Startup,
// before any event can be emitted.
func (a *App) initAnalytics() {
	a.analytics = analytics.New(analytics.Options{
		APIKey:  PostHogAPIKey,
		Host:    PostHogHost,
		Version: Version,
		Dir:     analytics.DataDir(),
	})
}

// track is the single call-site API for the Go side. It tolerates a nil
// client so bindings can be instrumented without worrying about ordering
// against Startup.
func (a *App) track(name string, props map[string]any) {
	a.analytics.Track(name, props)
}

// trackOnce emits at most one event per key per session. Used for reach
// metrics like panel_opened, where per-click volume would swamp the free
// tier without answering a different question.
func (a *App) trackOnce(key, name string, props map[string]any) {
	a.analytics.TrackOnce(key, name, props)
}

// ref turns a local project or process ID into a per-install opaque token,
// so "how many distinct projects does this user run?" is answerable without
// learning what any of them are.
func (a *App) ref(id string) string {
	return a.analytics.Ref(id)
}

// outcome builds the succeeded / failure_reason / duration_ms trio that
// every fallible operation reports. Keeping it in one place is what stops a
// raw error string reaching PostHog from some forgotten call site.
func outcome(start time.Time, err error, extra map[string]any) map[string]any {
	props := map[string]any{
		"succeeded":      err == nil,
		"failure_reason": analytics.FailureReason(err),
		"duration_ms":    time.Since(start).Milliseconds(),
	}
	for k, v := range extra {
		props[k] = v
	}
	return props
}

// --- Wails bindings ---

// AnalyticsSettings is what Preferences renders.
type AnalyticsSettings struct {
	Enabled     bool            `json:"enabled"`
	Configured  bool            `json:"configured"`
	DetailLevel string          `json:"detailLevel"`
	Categories  map[string]bool `json:"categories"`
}

// GetAnalyticsSettings reports the analytics state for Preferences.
func (a *App) GetAnalyticsSettings() AnalyticsSettings {
	prefs := a.analytics.Prefs()
	return AnalyticsSettings{
		Enabled:     prefs.Enabled,
		Configured:  a.analytics.Configured(),
		DetailLevel: prefs.DetailLevel,
		Categories:  prefs.Categories,
	}
}

// SetAnalyticsEnabled records the user's choice and applies it immediately.
// Turning it off also discards anything still buffered on disk.
func (a *App) SetAnalyticsEnabled(enabled bool) error {
	if !enabled {
		a.track("consent_decided", map[string]any{"granted": false})
	}
	if err := a.analytics.SetEnabled(enabled); err != nil {
		return err
	}
	if enabled {
		a.track("consent_decided", map[string]any{"granted": true})
	}
	return nil
}

// SetAnalyticsDetailLevel applies a Full / Balanced / Minimal preset.
func (a *App) SetAnalyticsDetailLevel(level string) error {
	return a.analytics.SetDetailLevel(level)
}

// SetAnalyticsCategories stores per-category toggles (marks Custom when needed).
func (a *App) SetAnalyticsCategories(categories map[string]bool) error {
	return a.analytics.SetCategories(categories)
}

// TrackEvent lets the frontend emit events that only the UI can see: which
// panel was opened, whether an AI suggestion was accepted, a theme change.
// Everything routes through the same consent gate and the same redaction as
// the Go side, so there is exactly one way out of this app.
func (a *App) TrackEvent(name string, props map[string]any) {
	a.track(name, props)
}

// TrackEventOnce is TrackEvent with per-session deduplication. The frontend
// uses it for panel_opened so a user flipping between tabs does not emit
// hundreds of identical events.
func (a *App) TrackEventOnce(key, name string, props map[string]any) {
	a.trackOnce(key, name, props)
}

// TrackModelSelected reports which local model the user picked.
//
// The raw model tag is passed in and split here rather than in the
// frontend, because a tag like "my-client-finetune:latest" can name a
// private project. Only the family and the parameter size leave this
// function; the tag itself never becomes a property.
func (a *App) TrackModelSelected(model string) {
	a.trackOnce("ai_model_selected:"+model, "ai_model_selected", map[string]any{
		"model_family": analytics.AIModelFamily(model),
		"param_size":   analytics.AIParamSize(model),
	})
}

// SetUpdateChannel tells the analytics layer which channel the user is on.
// The channel preference lives in the frontend, so the Go side cannot know
// it at Startup; the frontend calls this once at boot. The beta cohort is
// the early-warning system for regressions, so it needs to be a breakdown
// on every event rather than a property on update events alone.
func (a *App) SetUpdateChannel(beta bool) {
	channel := "stable"
	if beta {
		channel = "beta"
	}
	a.analytics.SetGlobal("update_channel", channel)
}
