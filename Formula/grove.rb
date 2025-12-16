# Homebrew formula for grove
# This is a template - the actual formula is maintained in henrilemoine/homebrew-tap
# and auto-updated by goreleaser

class Grove < Formula
  desc "Terminal UI for Git worktrees"
  homepage "https://github.com/henrilemoine/grove"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/henrilemoine/grove/releases/download/v#{version}/grove_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_ARM64"
    end
    on_intel do
      url "https://github.com/henrilemoine/grove/releases/download/v#{version}/grove_#{version}_darwin_x86_64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_AMD64"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/henrilemoine/grove/releases/download/v#{version}/grove_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
    on_intel do
      url "https://github.com/henrilemoine/grove/releases/download/v#{version}/grove_#{version}_linux_x86_64.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "grove"
  end

  test do
    assert_match "grove", shell_output("#{bin}/grove --version")
  end
end
