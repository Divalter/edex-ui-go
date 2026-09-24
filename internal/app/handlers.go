package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"edex-ui-go/internal/bridge"
	"edex-ui-go/internal/files"
	"edex-ui-go/internal/netinfo"
	"edex-ui-go/internal/terminal"
)

// Call runs an RPC method; it is exposed to the UI through the Wails binding.
func (a *App) Call(name string, args []json.RawMessage) (any, error) {
	return a.bridge.Dispatch(name, args)
}

func (a *App) registerHandlers() {
	b := a.bridge
	str := func(args []json.RawMessage, i int) string {
		var s string
		_ = bridge.Arg(args, i, &s)
		return s
	}

	b.Handle("boot", func([]json.RawMessage) (any, error) {
		// A (re)loaded UI starts with only the main shell, like the original.
		a.ttys.CloseExtras()
		return a.bootInfo()
	})

	tty := func(args []json.RawMessage) (*terminal.TTY, error) {
		var port int
		_ = bridge.Arg(args, 0, &port)
		t := a.ttys.Get(port)
		if t == nil {
			return nil, fmt.Errorf("no tty on port %d", port)
		}
		return t, nil
	}
	b.Handle("tty.attach", func(args []json.RawMessage) (any, error) {
		t, err := tty(args)
		if err != nil {
			return nil, err
		}
		if a.opts.EmitTTY == nil {
			return nil, fmt.Errorf("terminal events are not available")
		}
		sink := terminal.NewEventSink(func(b64 string) { a.opts.EmitTTY(t.Port, b64) })
		a.sinksMu.Lock()
		a.sinks[t.Port] = sink
		a.sinksMu.Unlock()
		t.Attach(sink)
		return nil, nil
	})
	b.Handle("tty.write", func(args []json.RawMessage) (any, error) {
		t, err := tty(args)
		if err != nil {
			return nil, err
		}
		return nil, t.Write([]byte(str(args, 1)))
	})
	b.Handle("tty.writeBinary", func(args []json.RawMessage) (any, error) {
		t, err := tty(args)
		if err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(str(args, 1))
		if err != nil {
			return nil, err
		}
		return nil, t.Write(data)
	})
	b.Handle("tty.ack", func(args []json.RawMessage) (any, error) {
		var port, n int
		_ = bridge.Arg(args, 0, &port)
		_ = bridge.Arg(args, 1, &n)
		a.sinksMu.Lock()
		sink := a.sinks[port]
		a.sinksMu.Unlock()
		if sink != nil {
			sink.Ack(n)
		}
		return nil, nil
	})
	b.Handle("ipc", a.handleIPC)

	b.Handle("si", func(args []json.RawMessage) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("missing systeminformation method")
		}
		return a.si.Call(str(args, 0), args[1:])
	})

	b.Handle("fs.readdir", func(args []json.RawMessage) (any, error) { return files.ReadDir(str(args, 0)) })
	b.Handle("fs.lstat", func(args []json.RawMessage) (any, error) { return files.Lstat(str(args, 0)), nil })
	b.Handle("fs.readFile", func(args []json.RawMessage) (any, error) { return files.ReadFile(str(args, 0)) })
	b.Handle("fs.writeFile", func(args []json.RawMessage) (any, error) {
		return nil, files.WriteFile(str(args, 0), str(args, 1))
	})
	b.Handle("fs.exists", func(args []json.RawMessage) (any, error) { return files.Exists(str(args, 0)), nil })
	b.Handle("fs.watch", func(args []json.RawMessage) (any, error) { return a.watcher.Watch(str(args, 0)) })
	b.Handle("fs.unwatch", func(args []json.RawMessage) (any, error) {
		var id int
		_ = bridge.Arg(args, 0, &id)
		a.watcher.Unwatch(id)
		return nil, nil
	})

	b.Handle("shell.openPath", func(args []json.RawMessage) (any, error) { return nil, OpenPath(str(args, 0)) })

	b.Handle("net.externalIP", func(args []json.RawMessage) (any, error) { return a.net.FetchExternalIP(str(args, 0)) })
	b.Handle("net.ping", func(args []json.RawMessage) (any, error) {
		var port int
		_ = bridge.Arg(args, 1, &port)
		return netinfo.Ping(str(args, 0), port, str(args, 2))
	})
	b.Handle("net.geoLookup", func(args []json.RawMessage) (any, error) {
		var ips []string
		_ = bridge.Arg(args, 0, &ips)
		return a.net.LookupAll(ips), nil
	})
	b.Handle("net.geoReady", func([]json.RawMessage) (any, error) { return a.net.Ready(), nil })

	b.Handle("update.latest", func([]json.RawMessage) (any, error) { return LatestRelease() })

	b.Handle("app.relaunch", func([]json.RawMessage) (any, error) { return nil, a.Relaunch() })
	b.Handle("app.quit", func([]json.RawMessage) (any, error) {
		if a.opts.Quit != nil {
			go a.opts.Quit()
		}
		return nil, nil
	})
}

// handleIPC implements the ipcMain channels of the original main process.
// It returns the replies addressed to the sender (e.sender.send), which the
// UI dispatches locally; asynchronous messages go through the event stream.
func (a *App) handleIPC(args []json.RawMessage) (any, error) {
	var channel string
	var params []json.RawMessage
	_ = bridge.Arg(args, 0, &channel)
	_ = bridge.Arg(args, 1, &params)
	param := func(i int) string {
		var s string
		if i < len(params) && json.Unmarshal(params[i], &s) != nil {
			s = strings.Trim(string(params[i]), `"`)
		}
		return s
	}
	reply := func(ch string, v ...any) []bridge.Event {
		if v == nil {
			v = []any{}
		}
		return []bridge.Event{{Channel: ch, Args: v}}
	}

	switch {
	case channel == "log":
		log.Printf("[%s] %s", param(0), param(1))
	case strings.HasPrefix(channel, "terminal_channel-"):
		port, err := strconv.Atoi(strings.TrimPrefix(channel, "terminal_channel-"))
		if err != nil {
			return nil, err
		}
		t := a.ttys.Get(port)
		if t == nil {
			return nil, nil
		}
		switch param(0) {
		case "Renderer startup":
			t.RendererStartup()
		case "Resize":
			cols, _ := strconv.Atoi(param(1))
			rows, _ := strconv.Atoi(param(2))
			t.Resize(cols, rows)
		}
	case channel == "ttyspawn":
		port, err := a.ttys.Spawn()
		if err != nil {
			log.Printf("TTY spawn denied (Reason: %v)", err)
			return reply("ttyspawn-reply", "ERROR: "+err.Error()), nil
		}
		return reply("ttyspawn-reply", fmt.Sprintf("SUCCESS: %d", port)), nil
	case channel == "getThemeOverride":
		a.overrideMu.Lock()
		defer a.overrideMu.Unlock()
		return reply("getThemeOverride", a.themeOverride), nil
	case channel == "getKbOverride":
		a.overrideMu.Lock()
		defer a.overrideMu.Unlock()
		return reply("getKbOverride", a.kbOverride), nil
	case channel == "setThemeOverride":
		v := param(0)
		a.overrideMu.Lock()
		a.themeOverride = &v
		a.overrideMu.Unlock()
	case channel == "setKbOverride":
		v := param(0)
		a.overrideMu.Lock()
		a.kbOverride = &v
		a.overrideMu.Unlock()
	default:
		log.Printf("Unknown IPC channel %q", channel)
	}
	return nil, nil
}
