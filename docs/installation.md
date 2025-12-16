# Installation

## Requirements

- Git 2.5+ (for worktree support)
- Go 1.21+ (for building from source)

## Install Methods

### Go Install (Recommended)

```bash
go install github.com/henrilemoine/grove/cmd/grove@latest
```

Make sure `$GOPATH/bin` is in your `PATH`.

### Build from Source

```bash
git clone https://github.com/henrilemoine/grove.git
cd grove
go build -o grove ./cmd/grove
sudo mv grove /usr/local/bin/
```

### Homebrew (macOS/Linux)

Coming soon.

## Verify Installation

```bash
grove --version
```

## Shell Completion

Grove doesn't require shell completion as it's a TUI application. Simply run `grove` in any git repository.

## Updating

```bash
go install github.com/henrilemoine/grove/cmd/grove@latest
```
