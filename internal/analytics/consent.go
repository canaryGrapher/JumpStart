package analytics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Consent lives in ~/.jumpstart/settings.json rather than config.json,
// because config.json holds a bare JSON array of projects and has no room
// for app-level settings.
const settingsFile = "settings.json"

// Settings is the on-disk shape of settings.json. It is deliberately a
// nested object so unrelated preferences can be added later without
// another file.
type Settings struct {
	Analytics ConsentState `json:"analytics"`
}

// ConsentState records the user's analytics choice. Enabled defaults to
// true for a fresh install; DecidedAt is set only when the user actively
// flips the toggle, which is how "never touched it" is told apart from
// "deliberately left it on".
type ConsentState struct {
	Enabled   bool  `json:"enabled"`
	DecidedAt int64 `json:"decidedAt,omitempty"`
}

var consentMu sync.Mutex

func settingsPath(dir string) string { return filepath.Join(dir, settingsFile) }

// LoadConsent reads the stored consent state. A missing or unreadable file
// means the user has not been asked yet, which counts as enabled.
func LoadConsent(dir string) ConsentState {
	consentMu.Lock()
	defer consentMu.Unlock()
	return loadConsentLocked(dir)
}

func loadConsentLocked(dir string) ConsentState {
	data, err := os.ReadFile(settingsPath(dir))
	if err != nil {
		return ConsentState{Enabled: true}
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return ConsentState{Enabled: true}
	}
	return s.Analytics
}

// SaveConsent persists the user's choice and stamps the decision time.
func SaveConsent(dir string, enabled bool) error {
	consentMu.Lock()
	defer consentMu.Unlock()

	// Read-modify-write so a future setting in this file is not clobbered.
	var s Settings
	if data, err := os.ReadFile(settingsPath(dir)); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	s.Analytics.Enabled = enabled
	s.Analytics.DecidedAt = time.Now().UnixMilli()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(dir), data, 0o644)
}
