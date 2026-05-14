# Installation

## macOS (Homebrew)

```bash
brew install henri123lemoine/tap/grove
```

Or tap first:

```bash
brew tap henri123lemoine/tap
brew install grove
```

Upgrade with `brew upgrade grove`.

## Linux / cross-platform install script

```bash
curl -fsSL https://raw.githubusercontent.com/henri123lemoine/grove/main/scripts/install.sh | sh
```

Installs the latest release to `~/.local/bin/grove`. Re-run the same command to upgrade. Works on macOS too if you'd rather not use Homebrew.

Pin a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/henri123lemoine/grove/main/scripts/install.sh | sh -s -- v0.2.0
```

Install somewhere else (e.g. system-wide):

```bash
curl -fsSL https://raw.githubusercontent.com/henri123lemoine/grove/main/scripts/install.sh \
    | GROVE_INSTALL_DIR=/usr/local/bin sh
```

The script supports `darwin` and `linux` on `x86_64` and `arm64`. It uses `curl` and `tar`, and prints a `PATH` hint if the install directory isn't already on `$PATH`.

## Linux packages

`.deb` and `.rpm` packages are published with every release at
<https://github.com/henri123lemoine/grove/releases>. Pick the asset matching your distro
and architecture (`amd64` or `arm64`), then install it. For example:

```bash
VERSION=v0.2.0   # check the releases page for the latest
NUM=${VERSION#v}
curl -fsSL -o /tmp/grove.deb \
    "https://github.com/henri123lemoine/grove/releases/download/${VERSION}/grove_${NUM}_linux_amd64.deb"
sudo dpkg -i /tmp/grove.deb
```

Replace `.deb`/`dpkg` with `.rpm`/`rpm -U` on Fedora / RHEL.

## Windows

Download the appropriate archive from the [releases page](https://github.com/henri123lemoine/grove/releases)
(`grove_<version>_windows_x86_64.tar.gz` or `_arm64`), extract `grove.exe`, and put it
somewhere on your `PATH`.

## From source

Requires Go 1.24+.

```bash
go install github.com/henri123lemoine/grove/cmd/grove@latest
```

Or check out the repo and build:

```bash
git clone https://github.com/henri123lemoine/grove.git
cd grove
make build       # produces ./grove and symlinks ~/.local/bin/grove
```

## Updating

- Homebrew users: `brew upgrade grove`.
- Quick-install users: re-run the curl one-liner.
- Package users: download the newer `.deb` / `.rpm` and reinstall.
