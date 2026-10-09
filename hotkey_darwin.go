//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Carbon -framework Cocoa
#include <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>
#include <pthread.h>

extern void jumpstartHotkeyPressed(void);

static EventHotKeyRef jsHotKey = NULL;
static EventHandlerRef jsHandler = NULL;

static OSStatus jsHotKeyHandler(EventHandlerCallRef next, EventRef event, void *data) {
	jumpstartHotkeyPressed();
	return noErr;
}

// Carbon hot keys must be (un)registered on the main thread.
static void jsOnMain(void (^block)(void)) {
	if (pthread_main_np()) block();
	else dispatch_sync(dispatch_get_main_queue(), block);
}

static int jsRegisterHotKey(int code, int mods) {
	__block OSStatus status = noErr;
	jsOnMain(^{
		if (jsHandler == NULL) {
			EventTypeSpec spec = { kEventClassKeyboard, kEventHotKeyPressed };
			InstallApplicationEventHandler(NewEventHandlerUPP(jsHotKeyHandler), 1, &spec, NULL, &jsHandler);
		}
		if (jsHotKey != NULL) {
			UnregisterEventHotKey(jsHotKey);
			jsHotKey = NULL;
		}
		EventHotKeyID hkid = { 'JSPK', 1 };
		status = RegisterEventHotKey(code, mods, hkid, GetApplicationEventTarget(), 0, &jsHotKey);
	});
	return (int)status;
}

static void jsUnregisterHotKey(void) {
	jsOnMain(^{
		if (jsHotKey != NULL) {
			UnregisterEventHotKey(jsHotKey);
			jsHotKey = NULL;
		}
	});
}

static void jsActivate(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp activateIgnoringOtherApps:YES];
	});
}
*/
import "C"

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"devdeck/internal/hotkey"
)

// hotkeyApp receives presses of the system-wide shortcut.
var hotkeyApp *App

//export jumpstartHotkeyPressed
func jumpstartHotkeyPressed() {
	a := hotkeyApp
	if a == nil || a.ctx == nil {
		return
	}
	go func() {
		C.jsActivate()
		runtime.WindowUnminimise(a.ctx)
		runtime.WindowShow(a.ctx)
		runtime.EventsEmit(a.ctx, "palette:open")
	}()
}

func registerSystemHotkey(a *App, key string) error {
	code, mods, err := hotkey.Parse(key)
	if err != nil {
		return err
	}
	hotkeyApp = a
	if status := C.jsRegisterHotKey(C.int(code), C.int(mods)); status != 0 {
		return fmt.Errorf("macOS refused the shortcut %s (error %d); it may be taken", key, int(status))
	}
	return nil
}

func unregisterSystemHotkey() {
	C.jsUnregisterHotKey()
}
