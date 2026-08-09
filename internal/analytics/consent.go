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

// ConsentState records the user's analytics choice and which categories
// may emit. Enabled defaults to true for a fresh install; DecidedAt is set
// only when the user actively flips the master toggle.
type ConsentState struct {
	Enabled     bool            `json:"enabled"`
	DecidedAt   int64           `json:"decidedAt,omitempty"`
	DetailLevel string          `json:"detailLevel,omitempty"`
	Categories  map[string]bool `json:"categories,omitempty"`
}

// Prefs is the runtime view of analytics preferences returned to the UI.
type Prefs struct {
	Enabled     bool            `json:"enabled"`
	DetailLevel string          `json:"detailLevel"`
	Categories  map[string]bool `json:"categories"`
}

var consentMu sync.Mutex

func settingsPath(dir string) string { return filepath.Join(dir, settingsFile) }

// LoadConsent reads the stored consent state. A missing or unreadable file
// means the user has not been asked yet, which counts as enabled + Full.
func LoadConsent(dir string) ConsentState {
	consentMu.Lock()
	defer consentMu.Unlock()
	return loadConsentLocked(dir)
}

func loadConsentLocked(dir string) ConsentState {
	data, err := os.ReadFile(settingsPath(dir))
	if err != nil {
		return defaultConsent()
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return defaultConsent()
	}
	return normalizeConsent(s.Analytics)
}

func defaultConsent() ConsentState {
	return ConsentState{
		Enabled:     true,
		DetailLevel: LevelFull,
		Categories:  DefaultCategories(),
	}
}

func normalizeConsent(c ConsentState) ConsentState {
	// Older files only had enabled/decidedAt — fill prefs with Full defaults.
	if c.DetailLevel == "" && len(c.Categories) == 0 {
		c.DetailLevel = LevelFull
		c.Categories = DefaultCategories()
		return c
	}
	c.Categories = MergeCategories(c.Categories)
	if c.DetailLevel == "" {
		c.DetailLevel = InferDetailLevel(c.Categories)
	} else {
		c.DetailLevel = NormalizeDetailLevel(c.DetailLevel)
		if c.DetailLevel != LevelCustom {
			// Keep stored categories authoritative if they diverge from the
			// labelled preset (e.g. partial writes); re-infer when mismatched.
			if InferDetailLevel(c.Categories) != c.DetailLevel {
				c.DetailLevel = InferDetailLevel(c.Categories)
			}
		}
	}
	return c
}

// SaveConsent persists the master switch and stamps the decision time,
// leaving detail/category prefs untouched.
func SaveConsent(dir string, enabled bool) error {
	consentMu.Lock()
	defer consentMu.Unlock()

	s := loadSettingsLocked(dir)
	s.Analytics = normalizeConsent(s.Analytics)
	s.Analytics.Enabled = enabled
	s.Analytics.DecidedAt = time.Now().UnixMilli()
	return writeSettingsLocked(dir, s)
}

// SaveAnalyticsPrefs persists the full analytics preference blob.
func SaveAnalyticsPrefs(dir string, prefs Prefs) error {
	consentMu.Lock()
	defer consentMu.Unlock()

	s := loadSettingsLocked(dir)
	s.Analytics.Enabled = prefs.Enabled
	s.Analytics.Categories = MergeCategories(prefs.Categories)
	s.Analytics.DetailLevel = NormalizeDetailLevel(prefs.DetailLevel)
	if s.Analytics.DetailLevel != LevelCustom {
		s.Analytics.Categories = CategoriesForLevel(s.Analytics.DetailLevel)
	} else {
		s.Analytics.DetailLevel = InferDetailLevel(s.Analytics.Categories)
	}
	s.Analytics.DecidedAt = time.Now().UnixMilli()
	return writeSettingsLocked(dir, s)
}

func loadSettingsLocked(dir string) Settings {
	var s Settings
	if data, err := os.ReadFile(settingsPath(dir)); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	s.Analytics = normalizeConsent(s.Analytics)
	return s
}

func writeSettingsLocked(dir string, s Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(dir), data, 0o644)
}
