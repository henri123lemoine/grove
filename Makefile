.PHONY: build run clean install

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

# Run linter
lint:
	golangci-lint run

# Run tests
test:
	go test -v ./...
