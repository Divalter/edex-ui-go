package app

import (
	"context"
	"log"

	"edex-ui-go/internal/config"
	"edex-ui-go/internal/terminal"
	"edex-ui-go/internal/theme"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	config      *config.Config
	termManager *terminal.Manager
	wsServer    *terminal.WebSocketServer
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		cfg = config.DefaultConfig()
	}
	a.config = cfg

	a.termManager = terminal.NewManager(a.config)
	a.wsServer, err = terminal.NewWebSocketServer(a.termManager)
	if err != nil {
		log.Fatalf("Failed to create WebSocket server: %v", err)
	}

	if err := a.wsServer.Start(); err != nil {
		log.Fatalf("Failed to start WebSocket server: %v", err)
	}
}

func (a *App) Shutdown(ctx context.Context) {
	if a.termManager != nil {
		a.termManager.CloseAll()
	}
}

func (a *App) DomReady(ctx context.Context) {
	runtime.WindowShow(ctx)
}

func (a *App) GetConfig() config.Config {
	if a.config == nil {
		return *config.DefaultConfig()
	}
	return *a.config
}

func (a *App) GetTheme(name string) (theme.Theme, error) {
	t, err := theme.LoadTheme(name)
	if err != nil {
		return theme.Theme{}, err
	}
	return *t, nil
}

func (a *App) ListThemes() []string {
	return theme.ListThemes()
}

func (a *App) GetTerminalWSInfo() terminal.WSInfo {
	if a.wsServer == nil {
		return terminal.WSInfo{}
	}
	return terminal.WSInfo{
		Port:  a.wsServer.Port,
		Token: a.wsServer.Token,
	}
}

func (a *App) CreateTerminalSession(id string) error {
	_, err := a.termManager.CreateSession(id)
	return err
}

func (a *App) ResizeTerminal(id string, cols, rows uint16) error {
	session, err := a.termManager.GetSession(id)
	if err != nil {
		return err
	}
	return session.Resize(cols, rows)
}

func (a *App) CloseTerminalSession(id string) error {
	return a.termManager.CloseSession(id)
}
