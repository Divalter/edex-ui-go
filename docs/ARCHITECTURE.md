# 🏗️ Architecture

eDEX-UI-Go uses Wails v2 to bridge a high-performance Go backend with a modern Svelte 5 frontend.

## 📊 System Architecture Diagram

```mermaid
flowchart TD
    subgraph Frontend [Frontend (Svelte 5 + TS)]
        UI[UI Components]
        Term[xterm.js Component]
        Sys[SysInfo Visuals]
        WailsJS[Wails JS Runtime]
    end

    subgraph Backend [Backend (Go)]
        App[App Controller]
        TerminalM[Terminal Manager]
        SysM[SysInfo Manager]
        FileM[FileBrowser Manager]
        ThemeM[Theme/Config Manager]
        PTY[PTY Subsystem]
    end

    UI <--> |Wails Bindings| App
    Sys <--> |Wails Events| SysM
    FileM <--> |Wails Bindings| UI
    ThemeM <--> |Wails Bindings| UI

    Term <--> |WebSocket IPC| TerminalM
    TerminalM <--> |StdIn/Out/Err| PTY
    PTY <--> |OS Shell| Shell[(Bash / Zsh / PowerShell)]
```

## 🧩 Go Backend Modules

- **App (`internal/app`)**: The main application lifecycle controller and entry point for Wails.
- **Terminal (`internal/terminal`)**: Manages pseudo-terminal (PTY) sessions, routing input/output.
- **SysInfo (`internal/sysinfo`)**: Collects CPU, RAM, network, and disk metrics via OS APIs and broadcasts them via Wails events.
- **Config & Theme (`internal/config`, `internal/theme`)**: Handles loading, saving, and parsing of `settings.json` and theme files.
- **FileBrowser (`internal/filebrowser`)**: Provides directory traversal and file manipulation APIs to the frontend.

## 💻 Frontend Components (Svelte 5)

We utilize the latest Svelte 5 runes (`$state`, `$derived`, `$effect`) for reactive and efficient rendering.
- **Terminal View**: Wraps `xterm.js` and manages WebSocket connections to the backend.
- **Globe/GlobeView**: Renders the 3D or 2D globe network visualization.
- **Stat Graphs**: Uses Chart.js for rendering historical system stats.
- **Keyboard**: A custom-built SVG/HTML virtual keyboard component.

## 🔌 IPC Strategy

### Wails Bindings vs WebSocket
- **Wails Bindings**: Used for standard request/response operations like file system navigation, fetching settings, and changing themes.
- **Wails Events**: Used for one-way broadcasting of system stats from backend to frontend.
- **WebSocket (Terminal)**: To ensure ultra-low latency and prevent the Wails IPC bridge from choking on massive stdout streams (like running `cat` on a huge file), terminal I/O is handled over a local WebSocket server hosted by the Go backend.

## 🖧 PTY Management

The `internal/terminal` package uses cross-platform pseudo-terminal libraries (`creack/pty` for Unix, `conpty` for Windows) to spawn and interact with the user's default shell.

## 🎨 Config and Theme System

Configuration and themes are loaded at startup. The backend parses the theme file, calculates derived colors if necessary, and injects CSS variables into the frontend to dynamically style the app.

## 🔒 Security Considerations

### WebSocket Auth Token
Because the WebSocket server is bound to `localhost`, any application on the machine could theoretically connect to it. To prevent unauthorized terminal access, the Go backend generates a cryptographically secure random token on startup. This token is securely passed to the frontend via Wails bindings, and the frontend must provide it when establishing the WebSocket connection.
