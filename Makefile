SHELL := /usr/bin/env bash -e

MODULE := github.com/Sanmo-Labs/rumpty-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
API_URL ?= https://api.rumptycloud.com

ROOT_DIR := $(abspath .)
TOOLS_DIR := $(abspath hack/tools)
export PATH := $(TOOLS_DIR):$(PATH)

GOLANGCI_LINT_VER ?= v2.10.1
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint-$(GOLANGCI_LINT_VER)
GOLANGCI_LINT_FLAGS ?=

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE) \
	-X $(MODULE)/internal/config.DefaultAPIURL=$(API_URL)

.PHONY: all build build-dev install test generate tools imports lint fix-lint verify verify-imports ldflags release-snapshot
all: build

$(GOLANGCI_LINT):
	@mkdir -p $(TOOLS_DIR)
	GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VER)
	@ln -sf golangci-lint $(GOLANGCI_LINT)

tools: $(GOLANGCI_LINT) ## Install developer tools

imports: WHAT ?= ./...
imports: $(GOLANGCI_LINT) ## Format imports and gofmt (like kcp make imports)
	$(GOLANGCI_LINT) fmt --enable gci -c $(ROOT_DIR)/.golangci.yaml $(WHAT)

lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(GOLANGCI_LINT) run $(GOLANGCI_LINT_FLAGS) -c $(ROOT_DIR)/.golangci.yaml --timeout 10m ./...

fix-lint: $(GOLANGCI_LINT) ## Run golangci-lint with --fix
	GOLANGCI_LINT_FLAGS="--fix" $(MAKE) lint

verify: lint test ## Lint and test

verify-imports: WHAT ?= ./...
verify-imports: $(GOLANGCI_LINT) ## Ensure imports and gofmt are up to date
	@$(GOLANGCI_LINT) fmt --enable gci --diff -c $(ROOT_DIR)/.golangci.yaml $(WHAT)

test:
	go test -v -race ./...

generate:
	go generate ./...

build:
	go build -ldflags "$(LDFLAGS)" -o bin/rumpty .

build-dev:
	$(MAKE) build API_URL=http://localhost:8889

install:
	go install -ldflags "$(LDFLAGS)" .

ldflags: ## Print linker flags (used by GoReleaser and local builds)
	@echo '$(LDFLAGS)'

release-snapshot: ## Build release artifacts locally without publishing
	@command -v goreleaser >/dev/null || { echo "install goreleaser: https://goreleaser.com"; exit 1; }
	goreleaser release --snapshot --clean
