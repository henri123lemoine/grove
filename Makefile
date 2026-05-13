.PHONY: build build-check run clean install deps test test-quick test-bash lint fmt fmt-check mod-tidy-check check setup-hooks release

# Version info for local builds
VERSION := $(shell git describe --tags --exact-match 2>/dev/null | sed 's/^v//' || true)
VERSION := $(or $(VERSION),dev)
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT)
LOCAL_BIN := $(HOME)/.local/bin
GO ?= go

# Build the binary and link to ~/.local/bin for local development
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o grove ./cmd/grove
	@mkdir -p $(LOCAL_BIN)
	@ln -sf $(CURDIR)/grove $(LOCAL_BIN)/grove

# Build all packages without installing or linking local artifacts
build-check:
	$(GO) build -v ./...

# Run directly
run:
	$(GO) run -ldflags "$(LDFLAGS)" ./cmd/grove

# Clean build artifacts
clean:
	rm -f grove

# Install to $GOPATH/bin
install:
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/grove

# Get dependencies
deps:
	$(GO) mod download

# Format code
fmt:
	$(GO) fmt ./...

# Fail if gofmt would change files (fast pre-commit check)
fmt-check:
	@files="$$(git ls-files '*.go')"; \
	if [ -z "$$files" ]; then exit 0; fi; \
	unformatted="$$(gofmt -l $$files)"; \
	test -z "$$unformatted" || (echo "$$unformatted"; exit 1)

# Fail if go.mod or go.sum are not tidy
mod-tidy-check:
	@set -e; \
	before="$$(mktemp -d)"; \
	cp go.mod go.sum "$$before/"; \
	trap 'cp "$$before/go.mod" go.mod; cp "$$before/go.sum" go.sum; rm -rf "$$before"' EXIT; \
	$(GO) mod tidy; \
	if ! cmp -s go.mod "$$before/go.mod" || ! cmp -s go.sum "$$before/go.sum"; then \
		echo "go.mod/go.sum are not tidy; run 'go mod tidy'"; \
		diff -u "$$before/go.mod" go.mod || true; \
		diff -u "$$before/go.sum" go.sum || true; \
		exit 1; \
	fi

# Run linter (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not found; install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run

# Run tests
test:
	$(GO) test -v -race ./...

# Quick sanity tests for pre-commit (aim to keep under ~1s)
test-quick:
	$(GO) test ./internal/config -run TestDefaultConfig

# Run bash/shell script tests
test-bash:
	@command -v bats >/dev/null || { echo "bats not found; install bats-core to run shell tests"; exit 1; }
	bats tests/

# Pre-push check: format + module tidy + build + test + lint
check: fmt-check mod-tidy-check build-check test lint
	@echo "✓ All checks passed"

# Setup git hooks for pre-commit sanity checks and pre-push validation
setup-hooks:
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'echo "Running pre-commit checks..."' >> .git/hooks/pre-commit
	@echo 'make fmt-check || exit 1' >> .git/hooks/pre-commit
	@echo 'make test-quick || exit 1' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Pre-commit hook installed"
	@echo '#!/bin/sh' > .git/hooks/pre-push
	@echo 'echo "Running pre-push checks..."' >> .git/hooks/pre-push
	@echo 'make check || exit 1' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✓ Pre-push hook installed"

# Create a new release (prompts for version bump type)
release:
	@./scripts/release.sh
