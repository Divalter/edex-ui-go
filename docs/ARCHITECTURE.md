# Architecture

eDEX-UI-GO keeps the frontend of eDEX-UI and replaces its Electron main process and Node.js
modules with a Go backend. The guiding rule is **fidelity**: the original UI code runs almost
unchanged, on top of small replacements for the Node/Electron APIs it used.

```
┌──────────────────────── Wails window (OS webview) ───────────────────────┐
│  frontend/src/renderer.js + classes/*.class.js   ← eDEX-UI 2.2.8 code    │
│        │ require("fs"), window.si, ipcRenderer, electron.remote…         │
│  frontend/src/host/node.js     Node/Electron replacements                │
│  frontend/src/host/bridge.js   transport                                 │
└────────┬───────────────────────────────┬─────────────────────────────────┘
         │ Host.Call (Wails binding)     │ Wails events "ipc" / "tty"
┌────────▼───────────────────────────────▼─────────────────────────────────┐
│ main.go                Wails app, asset handler for local files          │
│ internal/app           RPC handlers (what _boot.js and ipcMain did)      │
│ internal/terminal      shells, cwd/process tracking, output sinks        │
│ internal/pty           creack/pty (Unix) / ConPTY (Windows)              │
│ internal/sysinfo       systeminformation replacement (gopsutil…)         │
│ internal/files         fs replacement (readdir+lstat, watch…)            │
│ internal/netinfo       external IP, TCP ping, GeoLite2                   │
│ internal/config        user data dir, default files, asset mirroring     │
│ internal/bridge        RPC registry; HTTP server for edex-serve only     │
└──────────────────────────────────────────────────────────────────────────┘
```

## No network port

The original started one WebSocket server per terminal (`127.0.0.1:3000` and `3002`–`3005`)
without authentication, which let any website run commands. In eDEX-UI-GO:

- **RPC**: the UI calls `window.go.main.Host.Call(name, args)`; `internal/app/handlers.go` lists
  the methods (`boot`, `ipc`, `si`, `fs.*`, `tty.*`, `net.*`…).
- **Events**: the backend emits the Wails event `ipc` with `(channel, args)`. The frontend
  dispatches it to `ipcRenderer.on(channel)` listeners, so the original code
  (`terminal_channel-3000` → `New cwd`, `ttyspawn-reply`…) works unchanged.
- **Terminals**: output is sent as the Wails event `tty` `(port, base64)`, input with
  `tty.write`. `IpcTTYSocket` (bridge.js) mimics a WebSocket so that `terminal.class.js` and
  xterm's `AttachAddon` do not need to know. The "ports" survive as terminal identifiers.
- **Local files** (fonts, images, audio/video, PDF) are served by the Wails asset handler at
  `/edex-file?path=`, the replacement for `file://` URLs.

### Terminal flow control

A program flooding the terminal (`yes`, `cat` of a huge file) must not exhaust the webview.
`terminal.EventSink` batches output for 5 ms (up to 64 KiB) and counts unacknowledged bytes;
the UI acknowledges each batch after writing it to xterm (`tty.ack`). Above 1 MiB in flight,
`Send` blocks, which stops reading the PTY and applies backpressure to the program, like a real
terminal. Output produced while no UI is attached (startup, reload) is buffered (256 KiB).

## The ported frontend

- `frontend/src/renderer.js` is `_renderer.js`; `frontend/src/classes/` are the original classes.
  Each file starts with a notice listing its modifications (required by the GPLv3). Most of them
  only changed `module.exports` into `export`.
- `frontend/src/css/` holds the original stylesheets, unmodified, imported in the order of the
  original `ui.html`. `edex_go.css` contains the few additions (keyboard toggle, and two
  workarounds for rendering differences between Electron 12 and current engines).
- `frontend/public/assets/` holds the original fonts, sounds, themes, keyboard layouts, boot log
  and the Encom globe. At startup the backend copies themes, layouts and fonts to the user data
  directory, like the original, so users can edit them.
- `frontend/src/host/node.js` provides `require()` for the modules the original code loads:
  `fs`, `path`, `os`, `electron`, `@electron/remote`, `smoothie`, `howler`, `color`, `nanoid`,
  `pretty-bytes`, `mime-types`, `xterm*`…, plus `globalShortcut` (matched on `keydown`).
- `window.si` forwards every `systeminformation` call to `internal/sysinfo`, which returns the
  same JSON shapes.

## Development server

`cmd/edex-serve` runs the same backend without Wails and serves the UI to a normal browser, over
HTTP and WebSockets on loopback, protected by a random token and an Origin check. It exists for
development and automated UI testing only; the desktop app never starts it. With `-ipc` it streams
terminals as events, to exercise the Wails transport code in a browser.
