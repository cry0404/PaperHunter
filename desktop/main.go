package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.icns
var iconBytes []byte

func main() {

	app := NewApp()

	macOptions := &mac.Options{
		TitleBar:             mac.TitleBarHidden(),
		Appearance:           mac.NSAppearanceNameVibrantLight,
		WebviewIsTransparent: false,
		WindowIsTranslucent:  false,
		About: &mac.AboutInfo{
			Title:   "PaperHunter",
			Message: "多平台学术论文爬取与语义搜索工具",
			Icon:    iconBytes,
		},
	}

	winOptions := &windows.Options{
		WebviewIsTransparent:              false,
		WindowIsTranslucent:               false,
		DisableWindowIcon:                 false,
		DisableFramelessWindowDecorations: true,

		Theme: windows.SystemDefault,
	}

	var macOpts *mac.Options
	var winOpts *windows.Options

	switch runtime.GOOS {
	case "darwin":
		macOpts = macOptions
		err := wails.Run(&options.App{
			Title:  "PaperHunter",
			Width:  1024,
			Height: 768,
			AssetServer: &assetserver.Options{
				Assets: assets,
			},
			BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
			OnStartup:        app.startup,
			CSSDragProperty:  "--wails-draggable",
			CSSDragValue:     "drag",
			Mac:              macOpts,
			Windows:          winOpts,

			Bind: []interface{}{
				app,
			},
		})

		if err != nil {
			println("Error:", err.Error())
		}
	case "windows":
		winOpts = winOptions
		err := wails.Run(&options.App{
			Title:  "PaperHunter",
			Width:  1024,
			Height: 768,
			AssetServer: &assetserver.Options{
				Assets: assets,
			},
			BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
			OnStartup:        app.startup,
			CSSDragProperty:  "--wails-draggable",
			CSSDragValue:     "drag",
			Mac:              macOpts,
			Windows:          winOpts,
			Frameless:        true,
			Bind: []interface{}{
				app,
			},
		})

		if err != nil {
			println("Error:", err.Error())
		}
	}

}


