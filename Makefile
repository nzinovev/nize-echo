GOLANGCI_LINT_VERSION := v2.5.0
AIR_VERSION           := v1.67.4

BIN           := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint
AIR           := $(BIN)/air
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS       := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help build run dev dev-go dev-admin test lint fmt tidy tools clean

help: ## List make targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: ## Build the server binary into bin/site
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN)/site ./cmd/site

run: build ## Build and run the server
	$(BIN)/site

# Local development defaults; override on the command line or in the env.
dev dev-go: export LOG_LEVEL     ?= debug
dev dev-go: export ADMIN_DEV_URL ?= http://localhost:5173

dev: ## Run the Go server with live reload and the Vite dev server (needs docker compose up)
	@$(MAKE) --no-print-directory -j2 dev-go dev-admin

dev-go: $(AIR) ## Run the Go server with live reload
	$(AIR) -c .air.toml

dev-admin: admin/node_modules ## Run the Vite dev server for the admin
	npm --prefix admin run dev

admin/node_modules: admin/package.json admin/package-lock.json
	npm --prefix admin ci
	@touch $@

test: ## Run all Go tests with the race detector
	go test -race ./...

lint: tools ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

fmt: tools ## Format code (gofumpt, goimports)
	$(GOLANGCI_LINT) fmt ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

tools: $(GOLANGCI_LINT) $(AIR) ## Install pinned dev tools into bin/

$(GOLANGCI_LINT):
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(AIR):
	GOBIN=$(BIN) go install github.com/air-verse/air@$(AIR_VERSION)

clean: ## Remove build output and local tools
	rm -rf $(BIN)
