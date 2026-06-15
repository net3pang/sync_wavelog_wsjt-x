package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "WSJT-X \u2194 Wavelog \u65e5\u5fd7\u540c\u6b65\u5de5\u5177",
		Width:     820,
		Height:    680,
		MinWidth:  700,
		MinHeight: 550,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Frameless:        false,
		StartHidden:      false,
		HideWindowOnClose: false,
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "WSJT-X \u2194 Wavelog Gate",
				Message: "v1.0.0",
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
