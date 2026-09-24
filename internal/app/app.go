// Package app wires the eDEX-UI-GO backend together. It is the Go
// counterpart of the original _boot.js main process: it prepares the user
// data directory, spawns the main shell and answers the UI's calls.
//
// The package does not depend on Wails, so it can also run behind the
// browser-based development server (cmd/edex-serve).
package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"
	"time"

	"edex-ui-go/internal/bridge"
	"edex-ui-go/internal/config"
	"edex-ui-go/internal/files"
	"edex-ui-go/internal/netinfo"
	"edex-ui-go/internal/sysinfo"
	"edex-ui-go/internal/terminal"
)

// Name is the product name.
const Name = "eDEX-UI-GO"

// UpdateRepo is the GitHub repository checked for new releases.
const UpdateRepo = "Divalter/edex-ui-go"

// Options configures the backend.
type Options struct {
	Version string
	// Runtime describes the host runtime, shown in the welcome message.
	Runtime string
	// Assets holds the bundled "themes", "kb_layouts" and "fonts" folders.
	Assets fs.FS
	// ConfigDir overrides the user data directory.
	ConfigDir string
	// NoIntro and NoCursor are the --nointro and --nocursor CLI flags.
	NoIntro, NoCursor bool
	// Quit asks the host to close the application.
	Quit func()
	// Emit forwards events to the host IPC (Wails events) in addition to
	// the bridge event stream.
	Emit func(channel string, args ...any)
	// EmitTTY delivers terminal output (base64) through the host IPC.
	EmitTTY func(port int, b64 string)
}

// App is the running backend.
type App struct {
	opts     Options
	paths    config.Paths
	settings config.Settings

	bridge  *bridge.Server
	ttys    *terminal.Manager
	si      *sysinfo.SI
	net     *netinfo.Service
	watcher *files.Watcher

	overrideMu    sync.Mutex
	themeOverride *string
	kbOverride    *string

	sinksMu sync.Mutex
	sinks   map[int]*terminal.EventSink
}

// New prepares the user data directory and spawns the main shell. It does
// not open any port; the development server calls Bridge().Start itself.
func New(opts Options) (*App, error) {
	dir := opts.ConfigDir
	if dir == "" {
		dir = config.DefaultDir()
	}
	a := &App{opts: opts, paths: config.NewPaths(dir), si: sysinfo.New(), sinks: map[int]*terminal.EventSink{}}

	log.Printf("Starting %s v%s (%s)", Name, opts.Version, opts.Runtime)
	if err := config.Init(a.paths, opts.Assets, opts.Version); err != nil {
		return nil, err
	}
	log.Printf("Base config dir is %s", a.paths.Dir)
	settings, err := config.LoadSettings(a.paths)
	if err != nil {
		return nil, err
	}
	a.settings = settings

	a.bridge, err = bridge.New()
	if err != nil {
		return nil, err
	}
	a.watcher = files.NewWatcher(a.emit)
	a.net = netinfo.New(a.paths.GeoIPCache)
	go func() {
		if err := a.net.LoadGeoDB(); err != nil {
			log.Printf("GeoIP database unavailable: %v", err)
		}
	}()

	if err := a.startTerminal(); err != nil {
		return nil, err
	}
	a.registerHandlers()
	a.bridge.HandleTTY(func(port int, w http.ResponseWriter, r *http.Request) {
		t := a.ttys.Get(port)
		if t == nil {
			http.Error(w, "no such tty", http.StatusNotFound)
			return
		}
		t.ServeWS(w, r)
	})
	return a, nil
}

func (a *App) startTerminal() error {
	shell := a.settings.String("shell", "bash")
	resolved, err := exec.LookPath(shell)
	if err != nil {
		return fmt.Errorf("shell %q not found: %w", shell, err)
	}
	log.Printf("Shell found at %s", resolved)

	cwd := a.settings.String("cwd", a.paths.Dir)
	if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
		return errors.New("configured cwd path does not exist")
	}

	args := a.settings.ShellArgs()
	if len(args) == 0 && runtime.GOOS != "windows" {
		args = []string{"--login"}
	}

	vars := map[string]string{
		"TERM":                 "xterm-256color",
		"COLORTERM":            "truecolor",
		"TERM_PROGRAM":         Name,
		"TERM_PROGRAM_VERSION": a.opts.Version,
	}
	for k, v := range a.settings.Env() {
		vars[k] = v
	}
	env := terminal.MergeEnv(terminal.ShellEnv(resolved), vars)

	a.ttys = terminal.NewManager(terminal.Config{
		Shell:    resolved,
		Args:     args,
		Cwd:      cwd,
		Env:      env,
		BasePort: a.settings.Int("port", 3000),
	}, emitterFunc(a.emit))
	a.ttys.OnMainExit = func(int) {
		if a.opts.Quit != nil {
			a.opts.Quit()
		}
	}
	return a.ttys.StartMain()
}

type emitterFunc func(channel string, args ...any)

func (f emitterFunc) Emit(channel string, args ...any) { f(channel, args...) }

func (a *App) emit(channel string, args ...any) {
	a.bridge.Emit(channel, args...)
	if a.opts.Emit != nil {
		a.opts.Emit(channel, args...)
	}
}

// Bridge returns the bridge server.
func (a *App) Bridge() *bridge.Server { return a.bridge }

// Settings returns the settings loaded at startup.
func (a *App) Settings() config.Settings { return a.settings }

// Paths returns the user data paths.
func (a *App) Paths() config.Paths { return a.paths }

// LastWindowState returns the saved window state.
func (a *App) LastWindowState() map[string]any {
	s, err := config.LoadLastWindowState(a.paths)
	if err != nil {
		return map[string]any{"useFullscreen": true}
	}
	return s
}

// Shutdown kills every shell and stops the servers.
func (a *App) Shutdown() {
	log.Printf("Shutting down...")
	a.ttys.CloseAll()
	a.watcher.Close()
	_ = a.bridge.Close()
}

// BootInfo is everything the UI needs synchronously at startup, which the
// original renderer read with require() and the Electron remote module.
type BootInfo struct {
	Version         string          `json:"version"`
	Runtime         string          `json:"runtime"`
	Platform        string          `json:"platform"`
	OSType          string          `json:"osType"`
	Arch            string          `json:"arch"`
	Paths           config.Paths    `json:"paths"`
	Settings        config.Settings `json:"settings"`
	Shortcuts       json.RawMessage `json:"shortcuts"`
	LastWindowState map[string]any  `json:"lastWindowState"`
	NoIntro         bool            `json:"nointroOverride"`
	NoCursor        bool            `json:"nocursorOverride"`
	Username        string          `json:"username"`
	IsArch          bool            `json:"isArch"`
	MainPort        int             `json:"mainPort"`
	BootTime        int64           `json:"bootTime"`
}

// nodePlatform maps GOOS to Node's process.platform values used by the UI.
func nodePlatform() (platform, osType string) {
	switch runtime.GOOS {
	case "windows":
		return "win32", "Windows_NT"
	case "darwin":
		return "darwin", "Darwin"
	case "linux":
		return "linux", "Linux"
	}
	return runtime.GOOS, runtime.GOOS
}

func (a *App) bootInfo() (*BootInfo, error) {
	// Re-read the files: the UI can be reloaded after the settings editor
	// saved new values.
	settings, err := config.LoadSettings(a.paths)
	if err != nil {
		return nil, err
	}
	shortcuts, err := config.LoadShortcuts(a.paths)
	if err != nil {
		return nil, err
	}
	platform, osType := nodePlatform()
	info := &BootInfo{
		Version:         a.opts.Version,
		Runtime:         a.opts.Runtime,
		Platform:        platform,
		OSType:          osType,
		Arch:            runtime.GOARCH,
		Paths:           a.paths,
		Settings:        settings,
		Shortcuts:       shortcuts,
		LastWindowState: a.LastWindowState(),
		NoIntro:         a.opts.NoIntro,
		NoCursor:        a.opts.NoCursor,
		MainPort:        a.ttys.MainPort(),
		BootTime:        time.Now().Unix() - int64(sysinfo.Time().Uptime),
	}
	if u, err := user.Current(); err == nil {
		info.Username = u.Username
		if i := strings.LastIndexByte(info.Username, '\\'); i >= 0 {
			info.Username = info.Username[i+1:] // DOMAIN\user on Windows
		}
	}
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		info.IsArch = strings.Contains(string(data), "arch")
	}
	return info, nil
}

// OpenPath opens a file or folder with the default application, like
// Electron's shell.openPath.
func OpenPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// Relaunch starts a new instance of the app and quits this one.
func (a *App) Relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	if a.opts.Quit != nil {
		a.opts.Quit()
	}
	return nil
}

// Release is the part of GitHub's latest release response used by the UI.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// LatestRelease queries GitHub for the most recent release, pre-releases
// included (the /releases/latest endpoint ignores them).
func LatestRelease() (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+UpdateRepo+"/releases?per_page=1", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", Name+" UpdateChecker")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, errors.New("no release published yet")
	}
	return &releases[0], nil
}
