package main

import (
	"context"

	internalApp "edex-ui-go/internal/app"
	"edex-ui-go/internal/config"
	"edex-ui-go/internal/terminal"
	"edex-ui-go/internal/theme"
)

// App is a thin wrapper around the internal App struct to satisfy Wails binding requirements
type App struct {
	inner *internalApp.App
}

func NewApp() *App {
	return &App{
		inner: internalApp.NewApp(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.inner.Startup(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.inner.Shutdown(ctx)
}

func (a *App) domReady(ctx context.Context) {
	a.inner.DomReady(ctx)
}

// Exported methods for Wails bindings

func (a *App) GetConfig() config.Config {
	return a.inner.GetConfig()
}

func (a *App) GetTheme(name string) (theme.Theme, error) {
	return a.inner.GetTheme(name)
}

func (a *App) ListThemes() []string {
	return a.inner.ListThemes()
}

func (a *App) GetTerminalWSInfo() terminal.WSInfo {
	return a.inner.GetTerminalWSInfo()
}

func (a *App) CreateTerminalSession(id string) error {
	return a.inner.CreateTerminalSession(id)
}

func (a *App) ResizeTerminal(id string, cols, rows uint16) error {
	return a.inner.ResizeTerminal(id, cols, rows)
}

func (a *App) CloseTerminalSession(id string) error {
	return a.inner.CloseTerminalSession(id)
}
