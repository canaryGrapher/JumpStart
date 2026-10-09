// Package hotkey parses shortcut strings such as "cmd+shift+j" into macOS
// virtual key codes and Carbon modifier flags for RegisterEventHotKey.
package hotkey

import (
	"fmt"
	"strings"
)

// Carbon modifier flags (Events.h).
const (
	Cmd     = 1 << 8
	Shift   = 1 << 9
	Option  = 1 << 11
	Control = 1 << 12
)

// ANSI virtual key codes (Events.h kVK_*).
var keyCodes = map[string]int{
	"a": 0, "s": 1, "d": 2, "f": 3, "h": 4, "g": 5, "z": 6, "x": 7, "c": 8, "v": 9,
	"b": 11, "q": 12, "w": 13, "e": 14, "r": 15, "y": 16, "t": 17,
	"1": 18, "2": 19, "3": 20, "4": 21, "6": 22, "5": 23, "=": 24, "9": 25, "7": 26,
	"-": 27, "8": 28, "0": 29, "]": 30, "o": 31, "u": 32, "[": 33, "i": 34, "p": 35,
	"l": 37, "j": 38, "'": 39, "k": 40, ";": 41, "\\": 42, ",": 43, "/": 44, "n": 45,
	"m": 46, ".": 47, "space": 49, "`": 50,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118, "f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
}

var modNames = map[string]int{
	"cmd": Cmd, "command": Cmd, "⌘": Cmd, "shift": Shift, "⇧": Shift,
	"opt": Option, "option": Option, "alt": Option, "⌥": Option,
	"ctrl": Control, "control": Control, "⌃": Control,
}

// Parse turns "cmd+shift+j" into a key code and modifier mask. A shortcut
// needs Command, Control or Option so it cannot swallow ordinary typing.
func Parse(s string) (code, mods int, err error) {
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "")), "+")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("shortcut %q needs a modifier and a key, like cmd+shift+j", s)
	}
	key := parts[len(parts)-1]
	for _, p := range parts[:len(parts)-1] {
		m, ok := modNames[p]
		if !ok {
			return 0, 0, fmt.Errorf("unknown modifier %q in %q", p, s)
		}
		mods |= m
	}
	code, ok := keyCodes[key]
	if !ok {
		return 0, 0, fmt.Errorf("unsupported key %q in %q", key, s)
	}
	if mods&(Cmd|Control|Option) == 0 {
		return 0, 0, fmt.Errorf("shortcut %q needs ⌘, ⌃ or ⌥", s)
	}
	return code, mods, nil
}
