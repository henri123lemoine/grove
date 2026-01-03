# Installation

## Install Methods

### Homebrew (Recommended)

```bash
brew install henri123lemoine/tap/grove
```

Or add the tap first:

```bash
brew tap henri123lemoine/tap
brew install grove
```

### Go Install

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
