//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>

// Sets the app-wide NSAppearance. Passing "dark"/"light" pins the
// appearance (window chrome, sidebar vibrancy, controls); anything else
// resets to nil so AppKit follows the system setting again.
static void setAppAppearance(const char *name) {
	NSString *s = [NSString stringWithUTF8String:name];
	dispatch_async(dispatch_get_main_queue(), ^{
		if ([s isEqualToString:@"dark"]) {
			[NSApp setAppearance:[NSAppearance appearanceNamed:NSAppearanceNameDarkAqua]];
		} else if ([s isEqualToString:@"light"]) {
			[NSApp setAppearance:[NSAppearance appearanceNamed:NSAppearanceNameAqua]];
		} else {
			[NSApp setAppearance:nil];
		}
	});
}
*/
import "C"

import "unsafe"

// setNativeAppearance pins (or releases) the native appearance on macOS.
// Wails' WindowSetLightTheme/WindowSetDarkTheme are Windows-only no-ops,
// so without this the sidebar vibrancy tracks the OS appearance even when
// a theme is explicitly selected in-app.
func setNativeAppearance(mode string) {
	c := C.CString(mode)
	defer C.free(unsafe.Pointer(c))
	C.setAppAppearance(c)
}
