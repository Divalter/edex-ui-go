# Contributing

Thanks for helping keep eDEX-UI alive!

## Principles

1. **Fidelity first.** eDEX-UI-GO reproduces eDEX-UI 2.2.8. Changes to the look or the behavior
   of the original features should fix bugs or platform differences, not redesign them. New
   features are welcome when they are optional and fit the original spirit.
2. **Keep the original code recognizable.** The files in `frontend/src/classes/` and
   `frontend/src/renderer.js` are ported from eDEX-UI. Prefer adapting the Node/Electron
   replacements in `frontend/src/host/` or the Go backend over rewriting them. When you modify
   one of these files, update the notice at its top (the GPLv3 requires modified files to say so).
   The original stylesheets in `frontend/src/css/` stay unmodified; put CSS changes in
   `edex_go.css`.
3. **No network listener in the app.** Everything goes through the Wails IPC. Do not add servers.

## Setup

Requirements: Go 1.25+, Node.js 20+, Wails CLI v2, and on Linux `libgtk-3-dev` and
`libwebkit2gtk-4.1-dev`.

```bash
make deps     # npm install
make dev      # wails dev (desktop app with live reload)
make serve    # backend + UI in a browser, no Wails needed
make test     # Go tests with the race detector
make lint
```

## Layout

- `main.go`: Wails entry point (bindings, window options, local file handler).
- `internal/`: Go backend, see [ARCHITECTURE.md](ARCHITECTURE.md).
- `cmd/edex-serve`: development server.
- `frontend/src/`: ported UI (`renderer.js`, `classes/`, `css/`) and the host layer (`host/`).
- `frontend/public/assets/`: original fonts, sounds, themes, keyboard layouts.

## Style

- Go: `gofmt`, `go vet`, documented exported identifiers, tests next to the code.
- JavaScript: follow the style of the surrounding (original) code.
- Commits: small and focused, with a message explaining why.

## Pull requests

1. Fork and branch from `master`.
2. Add tests for backend changes; describe how you checked UI changes (screenshots welcome).
3. Make sure `make test` and `make lint` pass.
