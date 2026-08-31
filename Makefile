# SPDX-FileCopyrightText: 2026 OpenCHAMI Contributors
#
# SPDX-License-Identifier: MIT

.PHONY: help build clean docker-build fmt install lint lint-fix release-check release-snapshot run test test-race test-stress tidy tidy-check vet vuln check

GO ?= go
GIT ?= git
GOLANGCI_LINT ?= golangci-lint
GORELEASER ?= goreleaser
GOVULNCHECK ?= govulncheck
CONTAINER_PROG ?= docker

BINARY_NAME ?= cloud-init-server
CONTAINER_TAG ?= latest
TEST_TIMEOUT ?= 5m
STRESS_TEST_TIMEOUT ?= 10m
GOFLAGS ?= -v

VERSION ?= $(shell $(GIT) describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell $(GIT) rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_BRANCH ?= $(shell $(GIT) rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
GIT_TAG ?= $(shell $(GIT) describe --tags --abbrev=0 2>/dev/null || echo unknown)
GIT_STATE ?= $(shell if $(GIT) diff-index --quiet HEAD -- 2>/dev/null; then echo clean; else echo dirty; fi)
BUILD_HOST ?= $(shell hostname)
GO_VERSION ?= $(shell $(GO) env GOVERSION 2>/dev/null || echo unknown)
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

install: ## Download and verify Go modules
	$(GO) mod download
	$(GO) mod verify

build: ## Build the cloud-init server binary
	mkdir -p bin
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/cloud-init-server

run: build ## Build and run the service
	./bin/$(BINARY_NAME)

test: ## Run unit tests without the race detector
	$(GO) test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -count=1 -shuffle=on ./...

test-race: ## Run unit tests with the race detector
	GORACE="halt_on_error=1" $(GO) test $(GOFLAGS) -timeout $(TEST_TIMEOUT) -race -count=1 -shuffle=on ./...

test-stress: ## Run release stress tests
	$(GO) test $(GOFLAGS) -tags=stress -timeout $(STRESS_TEST_TIMEOUT) -count=1 ./cmd/cloud-init-server ./pkg/wgtunnel ./internal/memstore ./internal/smdclient

tidy: ## Tidy go.mod and go.sum
	$(GO) mod tidy

tidy-check: ## Verify go.mod and go.sum are tidy
	@tmpdir=$$(mktemp -d); \
	cp go.mod "$$tmpdir/go.mod"; \
	cp go.sum "$$tmpdir/go.sum"; \
	$(GO) mod tidy; \
	if ! cmp -s go.mod "$$tmpdir/go.mod" || ! cmp -s go.sum "$$tmpdir/go.sum"; then \
		echo "go.mod or go.sum changed after go mod tidy"; \
		rm -rf "$$tmpdir"; \
		exit 1; \
	fi; \
	rm -rf "$$tmpdir"

fmt: ## Format Go source
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

lint-fix: ## Run golangci-lint with autofix
	$(GOLANGCI_LINT) run --fix ./...

vuln: ## Run govulncheck
	$(GOVULNCHECK) ./...

docker-build: ## Build the runtime container image locally
	$(CONTAINER_PROG) build -t $(BINARY_NAME):$(CONTAINER_TAG) .

release-check: ## Validate GoReleaser configuration
	$(GORELEASER) check

release-snapshot: ## Build a local GoReleaser snapshot without publishing
	GIT_STATE=$(GIT_STATE) BUILD_HOST=$(BUILD_HOST) GO_VERSION=$(GO_VERSION) BUILD_USER=$(BUILD_USER) \
		$(GORELEASER) release --snapshot --clean --skip=publish

clean: ## Clean local build artifacts
	rm -rf bin dist coverage.out coverage.html
	$(GO) clean -cache

check: tidy-check vet lint test-race build vuln ## Run local pre-PR checks

.DEFAULT_GOAL := help
