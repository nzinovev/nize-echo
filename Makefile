GOLANGCI_LINT_VERSION := v2.5.0

BIN           := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS       := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help build run test lint fmt tidy tools clean

help: ## List make targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

build: ## Build the server binary into bin/site
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN)/site ./cmd/site

run: build ## Build and run the server
	$(BIN)/site

test: ## Run all Go tests with the race detector
	go test -race ./...

lint: tools ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

fmt: tools ## Format code (gofumpt, goimports)
	$(GOLANGCI_LINT) fmt ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

tools: $(GOLANGCI_LINT) ## Install pinned dev tools into bin/

$(GOLANGCI_LINT):
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

clean: ## Remove build output and local tools
	rm -rf $(BIN)
