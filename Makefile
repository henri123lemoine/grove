.PHONY: build run clean install test test-quick test-bash lint fmt fmt-check check setup-hooks release

# Build the binary
build:
	go build -o grove ./cmd/grove

# Run directly
run:
	go run ./cmd/grove

# Clean build artifacts
clean:
	rm -f grove

# Install to $GOPATH/bin
install:
	go install ./cmd/grove

# Get dependencies
deps:
	go get github.com/charmbracelet/bubbletea
	go get github.com/charmbracelet/lipgloss
	go get github.com/charmbracelet/bubbles
	go get github.com/pelletier/go-toml/v2
	go get github.com/sahilm/fuzzy

# Format code
fmt:
	go fmt ./...

# Fail if gofmt would change files (fast pre-commit check)
fmt-check:
	@files="$$(git ls-files '*.go')"; \
	if [ -z "$$files" ]; then exit 0; fi; \
	unformatted="$$(gofmt -l $$files)"; \
	test -z "$$unformatted" || (echo "$$unformatted"; exit 1)

# Run linter (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run

# Run tests
test:
	go test -v -race ./...

# Quick sanity tests for pre-commit (aim to keep under ~1s)
test-quick:
	go test ./internal/config -run TestDefaultConfig

# Run bash/shell script tests
test-bash:
	bats tests/

# Pre-push check: build + test + lint (run before pushing)
# This mirrors exactly what CI runs
check: build test lint
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
