.PHONY: build run clean install test lint fmt check setup-hooks

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

# Run linter (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	golangci-lint run

# Run tests
test:
	go test -v -race ./...

# Pre-push check: build + test + lint (run before pushing)
# This mirrors exactly what CI runs
check: build test lint
	@echo "✓ All checks passed"

# Setup git hooks for pre-push validation
setup-hooks:
	@echo '#!/bin/sh' > .git/hooks/pre-push
	@echo 'echo "Running pre-push checks..."' >> .git/hooks/pre-push
	@echo 'make check || exit 1' >> .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "✓ Pre-push hook installed"
