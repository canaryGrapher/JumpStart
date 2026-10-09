//go:build !darwin

package main

import "fmt"

func registerSystemHotkey(*App, string) error {
	return fmt.Errorf("a system-wide shortcut is only supported on macOS")
}

func unregisterSystemHotkey() {}
