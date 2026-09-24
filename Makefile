.PHONY: all dev build clean test lint frontend-install frontend-build

WAILS ?= $(shell which wails 2>/dev/null || echo $(HOME)/go/bin/wails)

all: build

# Inicia o servidor em modo de desenvolvimento com hot-reloading
dev:
	$(WAILS) dev

# Compila o binário de produção
build: frontend-build
	$(WAILS) build -clean -ldflags "-s -w"

# Instala dependências do frontend
frontend-install:
	cd frontend && npm install

# Constrói os assets estáticos do frontend (Svelte 5)
frontend-build:
	cd frontend && npm run build

# Executa testes unitários
test:
	go test -v -race ./...

# Verifica código Go com vet
lint:
	go vet ./...
	cd frontend && npm run check

# Limpa binários e diretórios de build
clean:
	rm -rf build/bin/
	rm -rf frontend/dist/
