package main

import (
	"sync"

	"devdeck/internal/analytics"
	"devdeck/internal/appsettings"
)

type hotkeyState struct {
	mu         sync.Mutex
	registered string
	err        string
}

// HotkeyStatus tells Settings whether the system-wide shortcut is active.
type HotkeyStatus struct {
	Mode       string `json:"mode"`
	Key        string `json:"key"`
	Registered bool   `json:"registered"`
	Error      string `json:"error,omitempty"`
}

// applyHotkeySettings registers the system-wide palette shortcut when the
// mode is "system" and removes it otherwise.
func (a *App) applyHotkeySettings(s appsettings.Settings) {
	a.hotkey.mu.Lock()
	defer a.hotkey.mu.Unlock()
	if s.HotkeyMode != appsettings.HotkeySystem {
		if a.hotkey.registered != "" {
			unregisterSystemHotkey()
		}
		a.hotkey.registered, a.hotkey.err = "", ""
		return
	}
	if a.hotkey.registered == s.HotkeyKey {
		return
	}
	if err := registerSystemHotkey(a, s.HotkeyKey); err != nil {
		a.hotkey.registered, a.hotkey.err = "", err.Error()
		return
	}
	a.hotkey.registered, a.hotkey.err = s.HotkeyKey, ""
}

// GetHotkeyStatus reports the configured and active system shortcut.
func (a *App) GetHotkeyStatus() HotkeyStatus {
	s := appsettings.Load(analytics.DataDir())
	a.hotkey.mu.Lock()
	defer a.hotkey.mu.Unlock()
	return HotkeyStatus{Mode: s.HotkeyMode, Key: s.HotkeyKey, Registered: a.hotkey.registered != "", Error: a.hotkey.err}
}
