# Installation

## Requirements

- Git 2.5+ (for worktree support)
- Go 1.21+ (for building from source)

## Install Methods

### Go Install (Recommended)

```bash
go install github.com/henri123lemoine/grove/cmd/grove@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

### Build from Source

```bash
git clone https://github.com/henri123lemoine/grove.git
cd grove
go build -o grove ./cmd/grove
sudo mv grove /usr/local/bin/
```

### Homebrew (macOS/Linux)

```bash
brew install henri123lemoine/tap/grove
```

Or add the tap first:

```bash
brew tap henri123lemoine/tap
brew install grove
```

## Verify Installation

```bash
grove --version
```

## Shell Completion

Grove doesn't require shell completion as it's a TUI application. Simply run `grove` in any git repository.

## Updating

With Homebrew:

```bash
go install github.com/henri123lemoine/grove/cmd/grove@latest
```
