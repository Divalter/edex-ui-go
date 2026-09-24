# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Linux: the process list reads `/proc/<pid>/stat` directly instead of using gopsutil, 15 times
  faster (5.9 ms instead of 90 ms for ~500 processes) with 29 times fewer allocations.
- The backend logs when a terminal connects to the UI, like the original.

### Added
- `docs/PERFORMANCE.md` and `scripts/perf/measure.py`: startup, memory and CPU compared with
  eDEX-UI 2.2.8.

## [0.0.2] - 2026-09-24

### Added
- Drop-down mode: a global hotkey (`dropdownHotkey`, F12 by default) hides the window and brings
  it back on top, sliding down. Implemented for X11 (pure Go, focus granted through
  `_NET_ACTIVE_WINDOW`) and Windows (`RegisterHotKey`). `edex-ui-go --toggle` does the same from
  the command line, for Wayland and macOS desktop shortcuts.

### Fixed
- A second launch (`edex-ui-go --toggle`, or starting it again) no longer starts a shell and
  touches the configuration before handing over to the running instance.
- The update checker ignored pre-releases.

## [0.0.1] - 2026-09-24

First port of [eDEX-UI](https://github.com/GitSquared/edex-ui) 2.2.8 by Gabriel "Squared" SAILLARD
to Go and Wails.

### Added
- The original UI, ported with its stylesheets, fonts, sounds, 21 themes and 19 keyboard layouts:
  boot screen, main shell with 4 extra tabs, system and network panels, globe, file browser with
  media/PDF viewers and text editor, fuzzy finder, on-screen keyboard, settings and shortcuts
  editors, update checker.
- Go backend replacing Electron and Node.js: PTY shells (creack/pty on Unix, ConPTY on Windows)
  with working directory and foreground process tracking, systeminformation-compatible system
  data (gopsutil), file access and watching, external IP, ping and GeoLite2 lookups.
- `hideKeyboard` setting and `KB_TOGGLE` shortcut (Ctrl+Shift+Alt+K) to hide the on-screen keyboard.
- `edex-serve`, a development server to run the UI in a browser.

### Security
- The terminals are no longer exposed through unauthenticated WebSocket servers on
  `127.0.0.1:3000+`, which allowed any website to run commands. The app does not listen on
  any network port; the UI uses the Wails IPC.
- File browser: names and paths typed in the shell are shell-quoted (a folder named `$(cmd)`
  executed `cmd` when clicked in eDEX-UI).
- Process names, disk labels, mount points and file paths are HTML-escaped before being
  displayed or used in inline handlers.
- Content Security Policy in the built UI.
- Built with Go 1.25.14 (toolchain pinned in go.mod): the Go 1.25.0 standard library had known
  vulnerabilities (crypto/tls, net/http, crypto/x509…).
- nanoid 3.3.19 and smoothie 1.36.1 (GHSA-28wg-ghj8-5hjv, GHSA-g662-qq45-ppwm).

### Fixed
- The UI hung at startup with `nointro` on WebKit (fonts wait).

### Notes
- First public preview. Linux is the primary target; the Windows and macOS builds are
  experimental and have not been tested on real machines yet.
- Linux: requires WebKitGTK 4.1 (`libwebkit2gtk-4.1-0`, installed by default on Debian 13,
  Ubuntu 24.04 and Fedora 40+). Extract the archive and run `./edex-ui-go`.
- The binaries are not signed. On macOS, run `xattr -dr com.apple.quarantine eDEX-UI-GO.app`
  after extracting; on Windows, SmartScreen may ask for confirmation.
