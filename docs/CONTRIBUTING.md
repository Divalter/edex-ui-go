# 🤝 Contributing to eDEX-UI-Go

Thank you for your interest in contributing to eDEX-UI-Go! 

## 📜 Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct (standard Contributor Covenant). Please be respectful and constructive in issues and pull requests.

## 🛠️ Setting up the Development Environment

1. Install Go 1.22+
2. Install Node.js 20+
3. Install Wails v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
4. Clone the repository: `git clone https://github.com/dcasula/edex-ui-go.git`
5. Run Wails dev mode: `wails dev`

## 📁 Project Structure

- `main.go`: Application entry point.
- `internal/`: Go backend modules (app, terminal, sysinfo, etc.).
- `frontend/`: Svelte 5 frontend source code.
  - `frontend/src/lib`: Reusable Svelte components and TS modules.
  - `frontend/src/routes`: SvelteKit routes (we primarily use the root layout).

## 💅 Code Style Guidelines

### Go
- Code must be formatted using `gofmt`.
- We use `golangci-lint` to catch common issues. Please run it before committing.
- Document exported functions and types.

### Frontend
- Use Svelte 5 Runes (`$state`, `$derived`, `$effect`).
- Write logic in TypeScript.
- Run `npm run lint` and `npm run format` (ESLint + Prettier).

## 🔄 Pull Request Process

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests.
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Create a descriptive Pull Request outlining your changes and the reasoning behind them.

## 📝 Issue Templates

When opening an issue, please use one of our templates:
- **Bug Report**: Provide clear reproduction steps, OS info, and screenshots.
- **Feature Request**: Detail the feature, its use case, and how it aligns with the project goals.
