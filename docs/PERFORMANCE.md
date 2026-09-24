# Performance

Measured on 2026-09-24, eDEX-UI-GO (main after 0.0.2) against the original eDEX-UI 2.2.8
(Electron 12.2.2, official AppImage), with [`scripts/perf/measure.py`](../scripts/perf/measure.py).

**Machine:** Intel Core i5-1135G7 (4 cores / 8 threads), 11 GB RAM, Debian 13, GNOME on X11,
WebKitGTK 2.52.6, 1920×1080.

**Method:** each app starts with a fresh configuration (defaults, `nointro`, no audio), fullscreen,
inside an isolated network namespace (no network for either: the original exposes an
unauthenticated shell on every interface). Three alternating runs per app; the table shows the
median.

| | eDEX-UI 2.2.8 (Electron) | eDEX-UI-GO | |
|---|---|---|---|
| Startup, until the terminal is connected | 4.31 s | **4.00 s** | −7% |
| Memory, PSS of all processes | 524 MB | **380 MB** | **−27%** |
| CPU at idle, % of one core | 50.5% | 49.6% | same |
| Processes | 15 | 4 | |
| Download (Linux) | 96.6 MB (AppImage) | **9.7 MB** (tar.gz) | **−90%** |
| Installed size | 236 MB | **22.9 MB** | **−90%** |

Runs: startup 4.00 / 4.00 / 3.98 s vs 4.18 / 4.37 / 4.31 s; CPU 50.8 / 49.6 / 46.1% vs
57.4 / 50.5 / 47.4%; PSS 394 / 380 / 373 MB vs 543 / 524 / 523 MB.

## Reading the numbers

- **Idle CPU is the UI, not the backend.** eDEX-UI animates all the time: the globe is a WebGL
  scene redrawn 30 times per second, the CPU and network graphs scroll continuously and the
  panels refresh every second or two. In eDEX-UI-GO about two thirds of the CPU is the
  WebKit web process and most of the rest is GTK compositing its frames. Both apps run the same
  UI code, so they cost about the same; the Go backend itself is a small part.
- **Startup is mostly the original boot sequence.** Even with `nointro`, the UI plays its
  opening animation (about 3 s of fixed delays) before the terminal connects.
- **Memory** is lower because there is one WebKit web process instead of Chromium's browser,
  GPU, zygote, utility and renderer processes plus a Node.js main process.

## Where the idle CPU goes

Measured per thread and by switching parts of the UI off (2026-09-24):

- The JavaScript is cheap: a CPU profile of the UI shows about 5% of a core, mostly the globe.
- The cost is the rendering pipeline of the webview. WebKitGTK composites the whole page for
  every frame in which anything changed, whatever its size, and GTK 3 redraws the whole window
  on the CPU. The globe (~30 fps) and the four graphs (30 and 40 fps) run on their own clocks,
  so something changes on almost every vsync.
- The same UI in a GTK 4 + WebKitGTK 6 test host costs 10 to 15 points less: GTK 4 composites
  the window on the GPU (about 7% of a core instead of 17 to 23%). The web process itself costs
  the same with both.

## Eco mode and covered windows

Same machine, busier than for the table above (so higher absolute numbers), CPU of the whole
process tree in percent of one core, 20 s samples after a 35 s warm-up:

| | normal | eco mode | covered window |
|---|---|---|---|
| Runs | 69.6 / 62.2 (then 67.1) | 29.3 / 29.1 | 12.0 |
| | | **−55%** | **−82%** |

- **Eco mode** (`ecoMode`) draws the globe and the graphs together at 10 fps through one frame
  clock (`frontend/src/host/frameclock.js`). Aligned frames matter: three loops at 10 fps out
  of phase would still make up to 30 frames per second. Off by default, since it changes the
  look of the animations; with it off, every animation keeps its original timing.
- **Covered windows** (fully below other windows, minimized or on another workspace) pause those
  animations. On X11 with a compositing window manager every window is drawn off screen, so
  WebKitGTK never knows it is covered; the backend checks the window stack twice per second
  (`internal/occlusion`) and tells the UI. WebView2, WKWebView and Wayland compositors already
  stop drawing covered windows. The panels still update once or twice per second.

## Backend optimizations

The process list (Top processes panel, task counter, process list window) is refreshed every one
to two seconds. With gopsutil, listing ~500 processes read about nine `/proc` files per process
and was the largest CPU cost of the backend (profiled with `pprof`). On Linux the backend now
reads `/proc/<pid>/stat` once per process:

| Scan of ~500 processes | time | allocated |
|---|---|---|
| gopsutil | 90.4 ms | 48.3 MB |
| `/proc/<pid>/stat` | **5.9 ms** | **1.7 MB** |

(`go test ./internal/sysinfo -bench Processes -benchmem`)

## Terminal output

The terminal output is sent to the UI in batches with flow control: above 1 MiB not yet displayed,
the backend stops reading the PTY, so a command printing faster than the UI can draw is slowed
down instead of filling the memory of the webview (the original had no such limit). Throughput
figures are therefore not comparable between the two apps and are not listed here.
