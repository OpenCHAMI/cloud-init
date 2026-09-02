# SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
#
# SPDX-License-Identifier: MIT

.PHONY: help build clean docker-build fmt mod-install lint lint-fix release-check release-snapshot run test test-race test-stress tidy tidy-check vet vuln check

GO ?= go
GIT ?= git
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
CONTAINER_PROG ?= docker

BINARY_NAME ?= cloud-init-server
CONTAINER_TAG ?= latest
TEST_TIMEOUT ?= 5m
STRESS_TEST_TIMEOUT ?= 10m
GOFLAGS ?= -v
GO_MOD_VERSION ?= $(shell awk '/^go / {print $$2; exit}' go.mod)
GO_TOOLCHAIN_VERSION ?= $(GO_MOD_VERSION)
GOTOOLCHAIN ?= go$(GO_TOOLCHAIN_VERSION)
GO_ENV := GOTOOLCHAIN=$(GOTOOLCHAIN)
GORELEASER_VERSION ?= v2.11.2
GORELEASER ?= $(GO_ENV) $(GO) run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

VERSION ?= $(shell $(GIT) describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell $(GIT) rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_BRANCH ?= $(shell $(GIT) rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
GIT_TAG ?= $(shell $(GIT) describe --tags --abbrev=0 2>/dev/null || echo unknown)
GIT_STATE ?= $(shell if $(GIT) diff-index --quiet HEAD -- 2>/dev/null; then echo clean; else echo dirty; fi)
BUILD_HOST ?= $(shell hostname)
GO_VERSION ?= $(shell $(GO_ENV) $(GO) env GOVERSION 2>/dev/null || echo unknown)
BUILD_USER ?= $(shell whoami)

LDFLAGS := -ldflags "-X 'main.GitCommit=$(COMMIT)' \
	-X 'main.BuildTime=$(DATE)' \
	-X 'main.Version=$(VERSION)' \
	-X 'main.GitBranch=$(GIT_BRANCH)' \
	-X 'main.GitTag=$(GIT_TAG)' \
	-X 'main.GitState=$(GIT_STATE)' \
	-X 'main.BuildHost=$(BUILD_HOST)' \
	-X 'main.GoVersion=$(GO_VERSION)' \
	-X 'main.BuildUser=$(BUILD_USER)'"

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m[VAR=val]... <target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_\/.-]+:.*##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

mod-install: ## Download and verify Go modules
	$(GO_ENV) $(GO) mod download
	$(GO_ENV) $(GO) mod verify

build: ## Build the cloud-init server binary
	mkdir -p bin
	$(GO_ENV) $(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cloud-init-server

run: build ## Build and run the service
	./bin/$(BINARY_NAME)

test: ## Run unit tests without the race detector
	$(GO_ENV) $(GO) test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -count=1 -shuffle=on ./...

test-race: ## Run unit tests with the race detector
	$(GO_ENV) GORACE="halt_on_error=1" $(GO) test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -race -count=1 -shuffle=on ./...

test-stress: ## Run release stress tests
	$(GO_ENV) $(GO) test $(GOFLAGS) -tags=stress -timeout $(STRESS_TEST_TIMEOUT) -count=1 ./cmd/cloud-init-server ./pkg/wgtunnel ./internal/memstore ./internal/smdclient

tidy: ## Tidy go.mod and go.sum
	$(GO_ENV) $(GO) mod tidy

tidy-check: ## Verify go.mod and go.sum are tidy
	@tmpdir=$$(mktemp -d); \
	cp go.mod "$$tmpdir/go.mod"; \
	cp go.sum "$$tmpdir/go.sum"; \
	$(GO_ENV) $(GO) mod tidy; \
	if ! cmp -s go.mod "$$tmpdir/go.mod" || ! cmp -s go.sum "$$tmpdir/go.sum"; then \
		echo "go.mod or go.sum changed after go mod tidy"; \
		rm -rf "$$tmpdir"; \
		exit 1; \
	fi; \
	rm -rf "$$tmpdir"

fmt: ## Format Go source
	$(GO_ENV) $(GO) fmt ./...

vet: ## Run go vet
	$(GO_ENV) $(GO) vet ./...

lint: ## Run golangci-lint
	$(GO_ENV) $(GOLANGCI_LINT) run ./...

lint-fix: ## Run golangci-lint with autofix
	$(GO_ENV) $(GOLANGCI_LINT) run --fix ./...

vuln: ## Run govulncheck
	$(GO_ENV) $(GOVULNCHECK) ./...

docker-build: ## Build the runtime container image locally
	$(CONTAINER_PROG) build -t $(BINARY_NAME):$(CONTAINER_TAG) .

release-check: ## Validate GoReleaser configuration
	$(GORELEASER) check

release-snapshot: ## Build a local GoReleaser snapshot without publishing
	$(GO_ENV) GIT_STATE=$(GIT_STATE) BUILD_HOST=$(BUILD_HOST) GO_VERSION=$(GO_VERSION) BUILD_USER=$(BUILD_USER) \
		$(GORELEASER) release --snapshot --clean --skip=publish

clean: ## Clean local build artifacts
	rm -rf bin dist coverage.out coverage.html
	$(GO_ENV) $(GO) clean -cache

check: tidy-check lint test-race build vuln ## Run local pre-PR checks

.DEFAULT_GOAL := help
