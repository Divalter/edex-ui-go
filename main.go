// eDEX-UI-GO — a Go + Wails port of eDEX-UI by Gabriel "Squared" SAILLARD.
//
// This is the desktop entry point. Everything runs locally without any
// network listener: the UI talks to the backend through the Wails IPC
// (Host.Call and events) and reads local files through the asset handler.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"

	"edex-ui-go/internal/app"
	"edex-ui-go/internal/bridge"
	"edex-ui-go/internal/buildinfo"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// Host is bound to the UI as window.go.main.Host.
type Host struct {
	ctx      atomic.Pointer[context.Context]
	backend  *app.App
	startErr error
}

// Call runs a backend RPC method (see internal/app/handlers.go).
func (h *Host) Call(name string, args []json.RawMessage) (any, error) {
	if h.backend == nil {
		return nil, h.startErr
	}
	return h.backend.Call(name, args)
}

func (h *Host) context() context.Context {
	if c := h.ctx.Load(); c != nil {
		return *c
	}
	return nil
}

func (h *Host) emit(channel string, args ...any) {
	if ctx := h.context(); ctx != nil {
		runtime.EventsEmit(ctx, "ipc", channel, args)
	}
}

func (h *Host) emitTTY(port int, b64 string) {
	if ctx := h.context(); ctx != nil {
		runtime.EventsEmit(ctx, "tty", port, b64)
	}
}

func (h *Host) quit() {
	if ctx := h.context(); ctx != nil {
		runtime.Quit(ctx)
	} else {
		os.Exit(0)
	}
}

func (h *Host) startup(ctx context.Context) {
	h.ctx.Store(&ctx)
	if h.startErr != nil {
		// Same behavior as the original crash dialog.
		_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   app.Name + " crashed",
			Message: h.startErr.Error(),
		})
		runtime.Quit(ctx)
	}
}

func (h *Host) shutdown(context.Context) {
	if h.backend != nil {
		h.backend.Shutdown()
	}
}

// serveAsset handles the requests the embedded assets cannot answer: local
// files requested by the file browser, media players and custom fonts.
func (h *Host) serveAsset(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == bridge.FilePath {
		bridge.ServeFile(w, r)
		return
	}
	http.NotFound(w, r)
}

func hasFlag(name string) bool {
	return slices.ContainsFunc(os.Args[1:], func(a string) bool { return strings.EqualFold(a, name) })
}

func main() {
	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	bundled, err := fs.Sub(dist, "assets")
	if err != nil {
		log.Fatal(err)
	}

	host := &Host{}
	host.backend, host.startErr = app.New(app.Options{
		Version:  buildinfo.Version,
		Runtime:  buildinfo.Runtime(true),
		Assets:   bundled,
		NoIntro:  hasFlag("--nointro"),
		NoCursor: hasFlag("--nocursor"),
		Quit:     host.quit,
		Emit:     host.emit,
		EmitTTY:  host.emitTTY,
	})
	if host.startErr != nil {
		log.Printf("Startup failed: %v", host.startErr)
	}

	allowWindowed, fullscreen := false, true
	if host.backend != nil {
		s := host.backend.Settings()
		allowWindowed = s.Bool("allowWindowed", false)
		fullscreen = s.Bool("forceFullscreen", true)
		if use, ok := host.backend.LastWindowState()["useFullscreen"].(bool); allowWindowed && ok && !use {
			fullscreen = false
		}
	}
	startState := options.Normal
	if fullscreen {
		startState = options.Fullscreen
	}

	err = wails.Run(&options.App{
		Title:            app.Name,
		Width:            1280,
		Height:           720,
		Frameless:        !allowWindowed,
		DisableResize:    !allowWindowed,
		WindowStartState: startState,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 255},
		AssetServer: &assetserver.Options{
			Assets:  dist,
			Handler: http.HandlerFunc(host.serveAsset),
		},
		OnStartup:  host.startup,
		OnShutdown: host.shutdown,
		Bind:       []interface{}{host},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "c1f3a7e2-edex-ui-go",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				log.Printf("Another instance of %s is already running.", app.Name)
				if ctx := host.context(); ctx != nil {
					runtime.WindowShow(ctx)
				}
			},
		},
		Linux: &linux.Options{
			// Like the original's ignore-gpu-blocklist switch: the globe and
			// the terminal renderer rely on WebGL.
			WebviewGpuPolicy: linux.WebviewGpuPolicyAlways,
			ProgramName:      "edex-ui-go",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
