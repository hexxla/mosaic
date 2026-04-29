# Makefile for LLM Folder Bootstrap CLI

.PHONY: help build build-mosaic-mcp build-mosaic-seed build-mosaic-create-db test test-all integration lint fmt ci clean install install-mosaic-mcp install-mosaic-seed install-mosaic-create-db seed reseed mosaic-dev run run-mosaic-mcp run-mosaic-seed run-mosaic-create-db create-db tidy vet update govulncheck \
	build-linux build-darwin build-windows build-all

# Bare `make` runs the full CI pipeline (same as `make ci`). Use `make help` to list targets.
.DEFAULT_GOAL := ci

# Detect host OS and architecture for output directory naming.
GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BINDIR  = bin/$(GOOS)-$(GOARCH)
# Windows binaries get .exe suffix; everything else gets none.
EXE     = $(if $(filter windows,$(GOOS)),.exe,)

# Build settings
BINARY_NAME := go-llm-project-structure
MOSAIC_BINARY := mosaic-mcp
MOSAIC_SEED_BINARY := mosaic-seed
MOSAIC_CREATE_DB_BINARY := mosaic-create-db
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Local Mosaic/MCP defaults (repository-relative `./.tmp`, not system `/tmp`). Override when invoking make.
MOSAIC_DB_PATH       ?= .tmp/mosaic-seed.hexxla
MOSAIC_MCP_ADDR      ?= 127.0.0.1:8787
MOSAIC_MCP_PATH      ?= /mcp
MOSAIC_OLLAMA_URL    ?= http://127.0.0.1:11434
MOSAIC_EMBED_MODEL   ?= all-minilm

# Extra arguments appended to `go run` for Mosaic commands (quote when passing multiple flags).
# Examples:
#   make create-db MOSAIC_CREATE_DB_FLAGS='-replace -policy configs/config.yaml'
#   make seed MOSAIC_SEED_FLAGS='-page-size 4096'
#   make run-mosaic-mcp MOSAIC_MCP_FLAGS='-policy configs/config.yaml'
MOSAIC_CREATE_DB_FLAGS ?=
MOSAIC_SEED_FLAGS      ?=
MOSAIC_MCP_FLAGS       ?=

# Build for the host OS/arch.
build:
	@mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINDIR)/$(BINARY_NAME)$(EXE) ./cmd/go-llm-project-structure
	@echo "  → $(BINDIR)/$(BINARY_NAME)$(EXE)"

build-mosaic-mcp:
	@mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINDIR)/$(MOSAIC_BINARY)$(EXE) ./cmd/mosaic-mcp
	@echo "  → $(BINDIR)/$(MOSAIC_BINARY)$(EXE)"

build-mosaic-seed:
	@mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINDIR)/$(MOSAIC_SEED_BINARY)$(EXE) ./cmd/mosaic-seed
	@echo "  → $(BINDIR)/$(MOSAIC_SEED_BINARY)$(EXE)"

build-mosaic-create-db:
	@mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINDIR)/$(MOSAIC_CREATE_DB_BINARY)$(EXE) ./cmd/mosaic-create-db
	@echo "  → $(BINDIR)/$(MOSAIC_CREATE_DB_BINARY)$(EXE)"

# Cross-compile helpers — override GOARCH if needed (e.g. make build-linux GOARCH=arm64).
build-linux:
	$(MAKE) build GOOS=linux GOARCH=$(or $(GOARCH),amd64)

build-darwin:
	$(MAKE) build GOOS=darwin GOARCH=$(or $(GOARCH),amd64)

build-windows:
	$(MAKE) build GOOS=windows GOARCH=$(or $(GOARCH),amd64)

# Build for all three major platforms (amd64).
build-all: build-linux build-darwin build-windows
	@echo "==> Built all platforms into bin/"

test:
	go test -race -count=1 ./...

test-all:
	@echo "Running unit tests..."
	@go test -race -count=1 ./...
	@echo ""
	@echo "Running integration tests..."
	@go test -race -count=1 -tags=integration ./...

integration:
	go test -count=1 -race -tags=integration ./...

lint:
	golangci-lint run --timeout=5m

fmt:
	gofmt -l -s -w .
	goimports -l -w . 2>/dev/null || true

ci:
	./scripts/ci/ci.sh

# LLM Tool Setup
llm-setup:
	@./scripts/llm/llm-setup.sh

# Cleanup LLM tool folders
clean-llm-cursor:
	rm -rf .cursor

clean-llm-claude:
	rm -rf .claude

clean-llm-windsurf:
	rm -rf .windsurf

clean-llm-continue:
	rm -rf .continue

clean-llm-copilot:
	rm -rf .github/agents .github/skills .github/instructions 2>/dev/null || true

clean-llm-all:
	rm -rf .cursor .claude .windsurf .continue .codex
	rm -rf .github/agents .github/skills .github/instructions 2>/dev/null || true

clean-llm: clean-llm-all

# Alias
clean-llm: clean-llm-all

install:
	go install ./cmd/go-llm-project-structure

install-mosaic-mcp:
	go install ./cmd/mosaic-mcp

install-mosaic-seed:
	go install ./cmd/mosaic-seed

install-mosaic-create-db:
	go install ./cmd/mosaic-create-db

help:
	@echo "Available targets:"
	@echo "  make ci              Full pipeline (same as GitHub Actions: ./scripts/ci/ci.sh)"
	@echo "  make build           Build the CLI binary for host OS"
	@echo "  make build-mosaic-mcp  Build mosaic-mcp (local MCP Streamable HTTP server)"
	@echo "  make build-mosaic-seed Build mosaic-seed (HexxlaDB file with demo corpus)"
	@echo "  make build-mosaic-create-db Build mosaic-create-db (empty Mosaic-compatible HexxlaDB file)"
	@echo "  make build-all       Cross-compile for linux/darwin/windows (amd64)"
	@echo "  make build-linux     Cross-compile for linux/amd64"
	@echo "  make build-darwin    Cross-compile for darwin/amd64"
	@echo "  make build-windows   Cross-compile for windows/amd64"
	@echo "  make test            Run tests"
	@echo "  make test-all        Run both unit and integration tests"
	@echo "  make integration     Run integration tests (slower, external dependencies)"
	@echo "  make lint            Run golangci-lint"
	@echo "  make fmt             Format code"
	@echo "  make govulncheck     Vulnerability scan only"
	@echo "  make install         Install the CLI locally"
	@echo "  make install-mosaic-mcp Install mosaic-mcp to GOPATH/bin"
	@echo "  make install-mosaic-seed Install mosaic-seed to GOPATH/bin"
	@echo "  make install-mosaic-create-db Install mosaic-create-db to GOPATH/bin"
	@echo "  make run             Run the CLI via go run"
	@echo "  make seed            Seed HexxlaDB (+ Ollama; MOSAIC_* ; optional MOSAIC_SEED_FLAGS)"
	@echo "  make reseed          Seed with -force (optional MOSAIC_SEED_FLAGS)"
	@echo "  make mosaic-dev      seed then run mosaic-mcp (optional MOSAIC_SEED_FLAGS / MOSAIC_MCP_FLAGS)"
	@echo "  make run-mosaic-mcp Run mosaic-mcp (MOSAIC_* ; optional MOSAIC_MCP_FLAGS e.g. -policy)"
	@echo "  make run-mosaic-create-db | make create-db  Empty DB (MOSAIC_DB_PATH; MOSAIC_CREATE_DB_FLAGS)"
	@echo "  make clean           Remove build artifacts"
	@echo "  make tidy            go mod tidy"
	@echo "  make llm-setup       Setup LLM tool configurations"

run:
	go run ./cmd/go-llm-project-structure

# Seed `.tmp/mosaic-seed.hexxla` (or MOSAIC_DB_PATH). Skips if the file already exists; use `make reseed`.
seed run-mosaic-seed:
	MOSAIC_DB_PATH="$(MOSAIC_DB_PATH)" MOSAIC_OLLAMA_URL="$(MOSAIC_OLLAMA_URL)" MOSAIC_EMBED_MODEL="$(MOSAIC_EMBED_MODEL)" \
		go run ./cmd/mosaic-seed -db "$(MOSAIC_DB_PATH)" $(MOSAIC_SEED_FLAGS)

# Replace the DB and seed from scratch.
reseed:
	MOSAIC_DB_PATH="$(MOSAIC_DB_PATH)" MOSAIC_OLLAMA_URL="$(MOSAIC_OLLAMA_URL)" MOSAIC_EMBED_MODEL="$(MOSAIC_EMBED_MODEL)" \
		go run ./cmd/mosaic-seed -db "$(MOSAIC_DB_PATH)" -force $(MOSAIC_SEED_FLAGS)

# Seed (if needed) then start mosaic-mcp until Ctrl+C.
mosaic-dev: seed
	MOSAIC_DB_PATH="$(MOSAIC_DB_PATH)" MOSAIC_MCP_ADDR="$(MOSAIC_MCP_ADDR)" MOSAIC_MCP_PATH="$(MOSAIC_MCP_PATH)" \
		go run ./cmd/mosaic-mcp $(MOSAIC_MCP_FLAGS)

run-mosaic-mcp:
	MOSAIC_DB_PATH="$(MOSAIC_DB_PATH)" MOSAIC_MCP_ADDR="$(MOSAIC_MCP_ADDR)" MOSAIC_MCP_PATH="$(MOSAIC_MCP_PATH)" \
		go run ./cmd/mosaic-mcp $(MOSAIC_MCP_FLAGS)

run-mosaic-create-db create-db:
	MOSAIC_DB_PATH="$(MOSAIC_DB_PATH)" go run ./cmd/mosaic-create-db -db "$(MOSAIC_DB_PATH)" $(MOSAIC_CREATE_DB_FLAGS)

clean:
	rm -rf bin
	go clean

tidy:
	go mod tidy

vet:
	go vet ./...

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

update:
	go get -u ./...
	go mod tidy
