package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "JumpStart",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		// Fully transparent so AppKit's vibrancy shows through wherever the
		// web content does not paint an opaque background (i.e. the sidebar).
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Mac: &mac.Options{
			// Hides the title bar and title text, keeps the traffic lights,
			// inset from the window edge (the Mail / Notes / Xcode look).
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "JumpStart",
				Message: "Local app and project manager",
			},
		},
		Windows: &windows.Options{
			// Mica gives a translucent, macOS-vibrancy-like backdrop on
			// Windows 11; older systems fall back to a solid window.
			BackdropType:         windows.Mica,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
