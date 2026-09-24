<p align="center">
  <img alt="eDEX-UI" src="docs/logo.png" width="320" />
</p>

<h1 align="center">eDEX-UI-GO</h1>

> **eDEX-UI-GO is a port of [eDEX-UI](https://github.com/GitSquared/edex-ui), created by
> [Gabriel "Squared" SAILLARD](https://github.com/GitSquared).**
> The interface, the themes, the keyboard layouts, the sounds, the fonts and most of the UI
> code are his work (and of the eDEX-UI contributors), released under the GPLv3.
> eDEX-UI was archived in October 2021; this project keeps its philosophy alive by replacing
> Electron and Node.js with a Go backend running on [Wails](https://wails.io).
> All credit for the design goes to the original project. Go star it. ⭐

---

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" />
  <img alt="Wails" src="https://img.shields.io/badge/Wails-v2-red" />
  <img alt="License" src="https://img.shields.io/badge/License-GPLv3-blue.svg" />
</p>

eDEX-UI-GO is a fullscreen, cross-platform terminal emulator and system monitor that looks and
feels like a sci-fi computer interface, heavily inspired by the
[TRON Legacy movie effects](https://web.archive.org/web/20170511000410/http://jtnimoy.com/blogs/projects/14881671).
It runs your shell in a real pseudo-terminal and surrounds it with live system information,
a file browser that follows your working directory, a network globe and an on-screen keyboard.

## Goals

- **Fidelity.** Same look, same boot sequence, same modules, same sounds, same shortcuts and the
  same `settings.json`, themes and keyboard layout formats as eDEX-UI 2.2.8. The original UI code
  is ported almost line by line; each modified file says what changed.
- **No Electron.** A small Go binary and the operating system's webview instead of a bundled
  Chromium and Node.js with native modules to rebuild.
- **Secure by design.** The original ran the shell behind an unauthenticated local WebSocket
  server that any website could connect to. eDEX-UI-GO does not open any network port: the UI
  talks to the backend through the Wails IPC only. See [Security](#security).
- **Linux first, but everywhere.** Linux is the primary target; Windows (ConPTY) and macOS are
  supported by the same code.

## Features

Everything the original had:

- Boot log and animated title screen (skippable with `nointro`)
- Main shell with up to 4 extra tabs, xterm.js with the WebGL renderer
- System panel: clock, date/uptime/OS/power, hardware, CPU usage/temperature/frequency,
  memory map and swap, top processes (click for the full process list)
- Network panel: interface state, public IP and ping, world globe (GeoIP), traffic graphs
- File browser following the terminal's working directory, disk list, text editor, image,
  audio, video and PDF viewers, fuzzy finder
- On-screen keyboard with 19 layouts, touch support and password mode
- Themes (21 bundled), settings editor, keyboard shortcuts editor, sound effects

Plus: a shortcut (**Ctrl+Shift+Alt+K**) and a `hideKeyboard` setting to hide the on-screen keyboard
and give its space to the file browser.

## Installing

Pre-built binaries will be published on the Releases page.

### Building from source

Requirements: [Go](https://go.dev/dl/) 1.25+, [Node.js](https://nodejs.org/) 20+ and the
[Wails CLI](https://wails.io/docs/gettingstarted/installation) v2:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

On Linux, the WebKitGTK development files are also needed, for example on Debian/Ubuntu:

```bash
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

Then:

```bash
git clone https://github.com/dcasula/edex-ui-go.git
cd edex-ui-go
make deps
make build        # binary in build/bin/
```

## Usage

```bash
./build/bin/edex-ui-go             # or edex-ui-go --nointro / --nocursor
```

The configuration lives in the user data directory, like the original:

| OS      | Location                                       |
| ------- | ---------------------------------------------- |
| Linux   | `~/.config/eDEX-UI-GO/`                        |
| macOS   | `~/Library/Application Support/eDEX-UI-GO/`    |
| Windows | `%APPDATA%\eDEX-UI-GO\`                        |

It contains `settings.json`, `shortcuts.json`, and the `themes`, `keyboards` and `fonts` folders.
The file formats are the ones of eDEX-UI, so existing themes and layouts can be dropped in.
The settings editor opens with **Ctrl+Shift+S**, the list of shortcuts with **Ctrl+Shift+K**.
See [docs/THEMES.md](docs/THEMES.md) to write a theme.

## Development

```bash
make dev          # desktop app with live reload (wails dev)
make serve        # backend + UI in your browser, no Wails/WebKit needed
make test
```

`make serve` runs `cmd/edex-serve`, a development-only server that prints a URL containing a
random token. It is handy to work on the UI with browser dev tools and to run automated UI tests.
See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md).

## Security

In eDEX-UI, each terminal was served by a WebSocket server listening on `127.0.0.1:3000` and
following ports, without any authentication. Browsers do not apply the same-origin policy to
WebSockets, so any web page could connect to it and type commands in your shell.

eDEX-UI-GO does not listen on any port. Terminal input and output, system information and file
access all go through the Wails IPC, which is only reachable from the application window.
The optional development server (`edex-serve`) binds to loopback only, requires a random
192-bit token on every request and rejects requests coming from other origins.

Other issues inherited from eDEX-UI are fixed as well:

- **Shell injection from file names.** Clicking a folder typed `cd "<name>"` in the shell, so a
  folder named `$(command)` (from a cloned repository, an archive…) ran `command`. Paths are now
  quoted for the shell (POSIX or PowerShell).
- **HTML injection.** Process names and users (of any local user), disk labels, mount points and
  file paths were inserted as HTML or inside inline handlers. Since the UI drives the shell, an
  injection amounts to code execution; these values are now escaped.
- **Content Security Policy.** The UI cannot load or send anything to another origin.

Please report vulnerabilities privately through GitHub security advisories.

## Credits

- **[eDEX-UI](https://github.com/GitSquared/edex-ui)** by **Gabriel "Squared" SAILLARD**
  ([gaby.dev](https://gaby.dev)) and its contributors: design, UI code, themes, keyboard layouts
  and assets. [PixelyIon](https://github.com/PixelyIon) helped with its Windows support.
- Sound effects by [IceWolf](https://soundcloud.com/iamicewolf), from eDEX-UI.
- [Encom Globe](https://github.com/arscan/encom-globe) by Rob "arscan" Scanlon (network globe).
- [xterm.js](https://xtermjs.org), [augmented-ui](https://augmented-ui.com),
  [smoothie charts](http://smoothiecharts.org), [howler.js](https://howlerjs.com),
  [pdf.js](https://mozilla.github.io/pdf.js/), [file-icons](https://github.com/file-icons/atom).
- Go: [Wails](https://wails.io), [gopsutil](https://github.com/shirou/gopsutil),
  [creack/pty](https://github.com/creack/pty), [conpty](https://github.com/UserExistsError/conpty),
  [gorilla/websocket](https://github.com/gorilla/websocket), [fsnotify](https://github.com/fsnotify/fsnotify),
  [geoip2-golang](https://github.com/oschwald/geoip2-golang), [battery](https://github.com/distatus/battery).
- GeoLite2 data by [MaxMind](https://www.maxmind.com).
- Inspired by the TRON Legacy movie effects (the
  [Board Room sequence](https://gmunk.com/TRON-Board-Room)) and by
  [DEX-UI](https://github.com/seenaburns/dex-ui) by [Seena](https://github.com/seenaburns),
  like the original.

## License

[GNU General Public License v3.0](LICENSE), the license of eDEX-UI.

Copyright © 2017-2021 Gabriel "Squared" SAILLARD (eDEX-UI) and the eDEX-UI contributors.
Copyright © 2026 the eDEX-UI-GO contributors.
