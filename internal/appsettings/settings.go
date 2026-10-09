// Package appsettings stores app-wide preferences that more than the UI
// needs to see (the MCP server, the AI backend, Raycast): autosave, AI
// thinking effort, hotkeys, OCR engine and date order. It lives in
// <data dir>/settings.json. Purely cosmetic per-window state stays in the
// browser's localStorage.
package appsettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const fileName = "settings.json"

// Think levels. ThinkAuto sends nothing and lets the model use its default.
const (
	ThinkAuto   = "auto"
	ThinkOff    = "off"
	ThinkLow    = "low"
	ThinkMedium = "medium"
	ThinkHigh   = "high"
)

// Hotkey modes.
const (
	HotkeyOff    = "off"
	HotkeyApp    = "app"    // ⌘K inside JumpStart only
	HotkeySystem = "system" // global shortcut that raises JumpStart
)

// OCR engines.
const (
	OCRVision    = "vision"
	OCROllama    = "ollama"
	OCRTesseract = "tesseract"
	OCROff       = "off"
)

// Settings is the persisted preference blob.
type Settings struct {
	// Autosave saves task edits when a field loses focus instead of on Save.
	Autosave bool `json:"autosave"`
	// ThinkLevel is the default AI reasoning effort: auto|off|low|medium|high.
	ThinkLevel string `json:"thinkLevel"`
	// HotkeyMode is off|app|system; HotkeyKey is the system shortcut, e.g. "cmd+shift+j".
	HotkeyMode string `json:"hotkeyMode"`
	HotkeyKey  string `json:"hotkeyKey"`
	// OCREngine picks how text is read from images: vision|ollama|tesseract|off.
	OCREngine string `json:"ocrEngine"`
	// OCRModel is the Ollama vision model used when OCREngine is "ollama".
	OCRModel string `json:"ocrModel,omitempty"`
	// DateOrder resolves ambiguous dates like 10/9: "mdy" (US) or "dmy".
	DateOrder string `json:"dateOrder"`
}

// Defaults keeps current behavior: manual save, model-default thinking,
// in-app palette only, Vision OCR, US date order.
func Defaults() Settings {
	return Settings{
		ThinkLevel: ThinkAuto,
		HotkeyMode: HotkeyApp,
		HotkeyKey:  "cmd+shift+j",
		OCREngine:  OCRVision,
		DateOrder:  "mdy",
	}
}

var mu sync.Mutex

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// Normalize fills blanks with defaults and rejects unknown values.
func Normalize(s Settings) (Settings, error) {
	d := Defaults()
	if s.ThinkLevel == "" {
		s.ThinkLevel = d.ThinkLevel
	}
	if s.HotkeyMode == "" {
		s.HotkeyMode = d.HotkeyMode
	}
	if s.HotkeyKey == "" {
		s.HotkeyKey = d.HotkeyKey
	}
	if s.OCREngine == "" {
		s.OCREngine = d.OCREngine
	}
	if s.DateOrder == "" {
		s.DateOrder = d.DateOrder
	}
	switch {
	case !oneOf(s.ThinkLevel, ThinkAuto, ThinkOff, ThinkLow, ThinkMedium, ThinkHigh):
		return s, fmt.Errorf("thinkLevel must be auto, off, low, medium or high")
	case !oneOf(s.HotkeyMode, HotkeyOff, HotkeyApp, HotkeySystem):
		return s, fmt.Errorf("hotkeyMode must be off, app or system")
	case !oneOf(s.OCREngine, OCRVision, OCROllama, OCRTesseract, OCROff):
		return s, fmt.Errorf("ocrEngine must be vision, ollama, tesseract or off")
	case !oneOf(s.DateOrder, "mdy", "dmy"):
		return s, fmt.Errorf("dateOrder must be mdy or dmy")
	}
	return s, nil
}

// Load reads settings; a missing or corrupt file yields the defaults.
func Load(dir string) Settings {
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return Defaults()
	}
	s := Defaults()
	if json.Unmarshal(data, &s) != nil {
		return Defaults()
	}
	if n, err := Normalize(s); err == nil {
		return n
	}
	return Defaults()
}

// Save validates and writes settings atomically.
func Save(dir string, s Settings) (Settings, error) {
	s, err := Normalize(s)
	if err != nil {
		return s, err
	}
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return s, err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return s, err
	}
	tmp := filepath.Join(dir, fileName+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return s, err
	}
	return s, os.Rename(tmp, filepath.Join(dir, fileName))
}
