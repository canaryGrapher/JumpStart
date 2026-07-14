//go:build !darwin

package main

// setNativeAppearance is a no-op off macOS; Wails' runtime theme calls
// in SetNativeTheme cover Windows.
func setNativeAppearance(mode string) {}
