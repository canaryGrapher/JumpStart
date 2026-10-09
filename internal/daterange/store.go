package daterange

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"devdeck/internal/model"
)

const globalFile = "calendar.json"

var globalMu sync.Mutex

type globalConfig struct {
	Quarters []model.QuarterRange `json:"quarters"`
}

// LoadGlobal reads the app-wide quarter dates from dir/calendar.json. A
// missing, unreadable, or invalid file yields nil, which callers treat as
// "use the calendar default".
func LoadGlobal(dir string) []model.QuarterRange {
	globalMu.Lock()
	defer globalMu.Unlock()
	data, err := os.ReadFile(filepath.Join(dir, globalFile))
	if err != nil {
		return nil
	}
	var c globalConfig
	if json.Unmarshal(data, &c) != nil || ValidateQuarters(c.Quarters) != nil {
		return nil
	}
	return c.Quarters
}

// SaveGlobal validates and writes the app-wide quarter dates. Passing nil
// removes the override so the calendar default applies again.
func SaveGlobal(dir string, qs []model.QuarterRange) error {
	globalMu.Lock()
	defer globalMu.Unlock()
	path := filepath.Join(dir, globalFile)
	if len(qs) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := ValidateQuarters(qs); err != nil {
		return err
	}
	data, err := json.MarshalIndent(globalConfig{Quarters: qs}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
