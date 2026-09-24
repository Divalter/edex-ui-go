.PHONY: all deps frontend dev build serve test lint clean

WAILS ?= $(shell which wails 2>/dev/null || echo $(HOME)/go/bin/wails)
VERSION ?= 0.1.0
LDFLAGS := -s -w -X edex-ui-go/internal/buildinfo.Version=$(VERSION)

# Linux distributions ship WebKitGTK 4.1 (libwebkit2gtk-4.1-dev)
ifeq ($(shell uname -s),Linux)
TAGS := -tags webkit2_41
endif

all: build

# Installs the frontend dependencies
deps:
	cd frontend && npm install

# Builds the frontend (Vite)
frontend:
	cd frontend && npm run build

# Desktop app with live reload
dev:
	$(WAILS) dev $(TAGS)

# Production binary in build/bin
build:
	$(WAILS) build -clean $(TAGS) -ldflags "$(LDFLAGS)"

# Backend + UI in a regular browser, without Wails (development / testing)
serve: frontend
	go run ./cmd/edex-serve -dist frontend/dist

test:
	go test -race ./internal/...

lint:
	gofmt -l internal cmd main.go
	go vet ./internal/... ./cmd/...

clean:
	rm -rf build/bin frontend/dist
