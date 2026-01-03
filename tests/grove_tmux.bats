#!/usr/bin/env bats
# Tests for grove.tmux TPM plugin

setup() {
    # Create temp directory for each test
    TEST_DIR=$(mktemp -d)
    export GROVE_BIN_DIR="$TEST_DIR/bin"
    export GROVE_BIN="$GROVE_BIN_DIR/grove"
    export VERSION_FILE="$GROVE_BIN_DIR/.version"

    # Source the functions from grove.tmux (without running main)
    # We extract just the functions we need to test
    source <(sed -n '/^get_tmux_option/,/^}/p; /^detect_platform/,/^}/p; /^get_latest_version/,/^}/p; /^get_installed_version/,/^}/p' "$BATS_TEST_DIRNAME/../grove.tmux")
}

teardown() {
    rm -rf "$TEST_DIR"
}

# Mock tmux command
tmux() {
    case "$1" in
        display-message) echo "$2" ;;
        show-option)
            if [[ "$3" == "@grove-key" ]]; then
                echo "$MOCK_GROVE_KEY"
            fi
            ;;
        bind-key) echo "bound $2" ;;
    esac
}
export -f tmux

@test "detect_platform returns valid format" {
    run detect_platform
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^(darwin|linux)_(amd64|arm64)$ ]]
}

@test "detect_platform matches current system" {
    run detect_platform
    [ "$status" -eq 0 ]

    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    [[ "$output" == ${os}_* ]]
}

@test "get_latest_version returns a version tag" {
    # Skip if no network
    if ! curl -sf "https://api.github.com" > /dev/null 2>&1; then
        skip "No network access"
    fi

    run get_latest_version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]
}

@test "get_installed_version returns empty when no version file" {
    run get_installed_version
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "get_installed_version returns version from file" {
    mkdir -p "$GROVE_BIN_DIR"
    echo "v1.2.3" > "$VERSION_FILE"

    run get_installed_version
    [ "$status" -eq 0 ]
    [ "$output" = "v1.2.3" ]
}

@test "get_tmux_option returns default when option not set" {
    MOCK_GROVE_KEY=""
    run get_tmux_option "@grove-key" "g"
    [ "$status" -eq 0 ]
    [ "$output" = "g" ]
}

@test "get_tmux_option returns custom value when set" {
    MOCK_GROVE_KEY="w"
    run get_tmux_option "@grove-key" "g"
    [ "$status" -eq 0 ]
    [ "$output" = "w" ]
}

@test "download URL is accessible" {
    # Skip if no network
    if ! curl -sf "https://api.github.com" > /dev/null 2>&1; then
        skip "No network access"
    fi

    platform=$(detect_platform)
    version=$(get_latest_version)
    version_num="${version#v}"
    url="https://github.com/henri123lemoine/grove/releases/download/${version}/grove_${version_num}_${platform}.tar.gz"

    run curl -sfI "$url"
    [ "$status" -eq 0 ]
}

@test "full download and install works" {
    # Skip if no network
    if ! curl -sf "https://api.github.com" > /dev/null 2>&1; then
        skip "No network access"
    fi

    # Source download function
    source <(sed -n '/^download_grove/,/^}/p' "$BATS_TEST_DIRNAME/../grove.tmux")

    run download_grove
    [ "$status" -eq 0 ]
    [ -x "$GROVE_BIN" ]
    [ -f "$VERSION_FILE" ]

    # Verify it's actually executable
    run "$GROVE_BIN" --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^grove ]]
}

@test "prefers system grove when available" {
    # Create a fake grove in PATH
    mkdir -p "$TEST_DIR/fake_bin"
    cat > "$TEST_DIR/fake_bin/grove" << 'EOF'
#!/bin/bash
echo "fake grove"
EOF
    chmod +x "$TEST_DIR/fake_bin/grove"

    # Add to PATH
    export PATH="$TEST_DIR/fake_bin:$PATH"

    # Source ensure_grove
    source <(sed -n '/^ensure_grove/,/^}/p' "$BATS_TEST_DIRNAME/../grove.tmux")

    run ensure_grove
    [ "$status" -eq 0 ]
    [ "$output" = "grove" ]  # Should return "grove" not full path
}

@test "uses local binary when no system grove" {
    # Ensure grove is not in PATH for this test
    export PATH="/usr/bin:/bin"

    # Create a pre-installed local binary
    mkdir -p "$GROVE_BIN_DIR"
    cat > "$GROVE_BIN" << 'EOF'
#!/bin/bash
echo "local grove"
EOF
    chmod +x "$GROVE_BIN"
    echo "v0.1.0" > "$VERSION_FILE"

    # Source ensure_grove (need detect_platform and get_latest_version too)
    source <(sed -n '/^detect_platform/,/^}/p; /^get_latest_version/,/^}/p; /^get_installed_version/,/^}/p; /^download_grove/,/^}/p; /^ensure_grove/,/^}/p' "$BATS_TEST_DIRNAME/../grove.tmux")

    run ensure_grove
    [ "$status" -eq 0 ]
    [ "$output" = "$GROVE_BIN" ]
}
