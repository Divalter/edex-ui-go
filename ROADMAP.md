# eDEX-UI-GO 2026 rework

This branch explores a native rewrite of the interface: same look as eDEX-UI, drawn directly on
the GPU with [Gio](https://gioui.org) instead of HTML in a webview. The faithful port, which runs
the original UI code, stays on `main`.

## Why

Measurements on `main` (see [docs/PERFORMANCE.md](docs/PERFORMANCE.md)) show that the idle CPU
is not spent by the UI code but by the webview rendering pipeline: the page is composited for
every frame in which anything changed, about half a core while the globe and the graphs animate.
Eco mode and pausing covered windows cut it, but the floor stays high. Native GPU renderers of
comparable dashboards and terminals stay at a few percent.

## Principles

- **Same look.** Colors, fonts, layout, sounds and boot sequence of eDEX-UI. Themes keep their
  JSON format (colors, fonts, terminal colors); `injectCSS` cannot be supported.
- **Same backend.** `internal/*` (terminal, pty, sysinfo, files, netinfo, config, hotkey,
  occlusion) is reused as is. The UI calls it directly: no IPC, no JSON serialization.
- **Measure before building.** Each phase ends with the same measurement script as `main`.

## Phases

1. **Spike (go/no-go).** A Gio window with the clock, the two CPU graphs and a terminal
   (VT parser from `github.com/charmbracelet/x/vt`), tron theme, fullscreen. Measure idle CPU
   and memory against `main`. Target: under 10% of a core and under 100 MB. If it is not
   clearly better, the rework stops here.
2. **Terminal.** Scrollback, selection and copy/paste, wide characters and emoji, IME (dead keys,
   ABNT2), tabs, cwd tracking, bell and feedback sounds.
3. **Panels.** System (clock, sysinfo, hardware, CPU, memory map, top processes), network
   (status, globe, traffic graphs), file browser, on-screen keyboard (same layout JSON files).
4. **Windows and modals.** Settings and shortcuts editors, process list, fuzzy finder, file
   viewers (text editor, images, audio/video, PDF).
5. **Boot and polish.** Boot log and title screen, theme switching, audio, drop-down mode
   (GlobalShortcuts portal on Wayland), packaging for Linux, Windows and macOS.

## Open questions

- 3D globe: Gio has no 3D API; the points can be projected on the CPU (a few thousand per frame)
  or drawn with a custom GPU shader.
- Video playback and PDF rendering without a webview.
- Accessibility (screen readers) of a custom-drawn UI.
