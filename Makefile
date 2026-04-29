# Makefile for LLM Folder Bootstrap CLI

.PHONY: help build test test-all integration lint fmt ci clean install run tidy vet update govulncheck \
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
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build for the host OS/arch.
build:
	@mkdir -p $(BINDIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINDIR)/$(BINARY_NAME)$(EXE) ./cmd/go-llm-project-structure
	@echo "  → $(BINDIR)/$(BINARY_NAME)$(EXE)"

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

help:
	@echo "Available targets:"
	@echo "  make ci              Full pipeline (same as GitHub Actions: ./scripts/ci/ci.sh)"
	@echo "  make build           Build the CLI binary for host OS"
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
	@echo "  make run             Run the CLI via go run"
	@echo "  make clean           Remove build artifacts"
	@echo "  make tidy            go mod tidy"
	@echo "  make llm-setup       Setup LLM tool configurations"

run:
	go run ./cmd/go-llm-project-structure

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
