GOLANGCI_LINT_VERSION := v2.5.0
AIR_VERSION           := v1.67.1
SQLC_VERSION          := v1.30.0

BIN           := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint
AIR           := $(BIN)/air
SQLC          := $(BIN)/sqlc
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS       := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

# Compose Postgres (compose.yaml); POSTGRES_PORT matches its host port.
POSTGRES_PORT ?= 5432
DEV_DATABASE_URL := postgres://nize:nize@localhost:$(POSTGRES_PORT)/nize_echo?sslmode=disable

.PHONY: help build run dev dev-go dev-admin test lint fmt generate image tidy tools clean

help: ## List make targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: ## Build the server binary into bin/site
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN)/site ./cmd/site

run: build ## Build and run the server (DATABASE_URL defaults to the compose Postgres)
	DATABASE_URL="$${DATABASE_URL:-$(DEV_DATABASE_URL)}" $(BIN)/site

# Local development defaults; override on the command line or in the env.
dev dev-go: export DATABASE_URL  ?= $(DEV_DATABASE_URL)
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

# Database tests use the compose Postgres and skip when it is not running;
# TEST_DATABASE_REQUIRED=1 turns those skips into failures (CI).
test: export TEST_DATABASE_URL ?= $(DEV_DATABASE_URL)
test: ## Run all Go tests with the race detector
	go test -race ./...

lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

fmt: $(GOLANGCI_LINT) ## Format code (gofumpt, goimports)
	$(GOLANGCI_LINT) fmt ./...

generate: $(SQLC) ## Generate Go code from db/queries with sqlc
	@if ls db/queries/*.sql >/dev/null 2>&1; then $(SQLC) generate; \
	else echo "generate: no queries in db/queries yet, nothing to do"; fi

image: ## Build the Docker image nize-echo:dev (deploy/Dockerfile)
	docker build -f deploy/Dockerfile --build-arg VERSION=$(VERSION) -t nize-echo:dev .

tidy: ## Tidy go.mod and go.sum
	go mod tidy

tools: $(GOLANGCI_LINT) $(AIR) $(SQLC) ## Install pinned dev tools into bin/

$(GOLANGCI_LINT):
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(AIR):
	GOBIN=$(BIN) go install github.com/air-verse/air@$(AIR_VERSION)

$(SQLC):
	GOBIN=$(BIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)

clean: ## Remove build output and local tools
	rm -rf $(BIN)
