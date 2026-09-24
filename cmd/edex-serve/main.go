// Command edex-serve runs the eDEX-UI-GO backend without a desktop window
// and serves the UI to a regular browser. It is meant for development and
// automated testing: it does not need the webview libraries Wails requires.
//
//	go run ./cmd/edex-serve -dist frontend/dist
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"edex-ui-go/internal/app"
	"edex-ui-go/internal/buildinfo"
)

func main() {
	dist := flag.String("dist", "frontend/dist", "built frontend directory")
	addr := flag.String("addr", "127.0.0.1:0", "listen address")
	configDir := flag.String("config", "", "user data directory (default: the platform config dir)")
	nointro := flag.Bool("nointro", false, "skip the boot animation")
	nocursor := flag.Bool("nocursor", false, "hide the mouse cursor")
	ipc := flag.Bool("ipc", false, "stream terminals as events, like the Wails IPC transport (for testing it in a browser)")
	flag.Parse()

	distFS := os.DirFS(*dist)
	quit := make(chan os.Signal, 1)
	var emitTTY func(port int, b64 string)
	var backend *app.App
	if *ipc {
		emitTTY = func(port int, b64 string) { backend.Bridge().Emit("__tty", port, b64) }
	}
	backend, err := app.New(app.Options{
		Version:   buildinfo.Version,
		Runtime:   buildinfo.Runtime(false) + " (browser)",
		Assets:    os.DirFS(filepath.Join(*dist, "assets")),
		ConfigDir: *configDir,
		NoIntro:   *nointro,
		NoCursor:  *nocursor,
		Quit:      func() { quit <- syscall.SIGTERM },
		EmitTTY:   emitTTY,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Shutdown()

	srv := backend.Bridge()
	srv.ServeStatic(distFS)
	if err := srv.Start(*addr); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("eDEX-UI-GO UI: %s/?token=%s\n", srv.URL(), srv.Token)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}
