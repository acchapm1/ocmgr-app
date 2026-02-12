# Homebrew formula for ocmgr — OpenCode Profile Manager
#
# To use this formula as a tap:
#   brew tap acchapm1/ocmgr-app https://github.com/acchapm1/ocmgr-app
#   brew install ocmgr
#
# SHA256 values are updated by the release pipeline.

class Ocmgr < Formula
  desc "Manage reusable .opencode profiles across projects"
  homepage "https://github.com/acchapm1/ocmgr-app"
  version "0.0.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/acchapm1/ocmgr-app/releases/download/v#{version}/ocmgr_v#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/acchapm1/ocmgr-app/releases/download/v#{version}/ocmgr_v#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/acchapm1/ocmgr-app/releases/download/v#{version}/ocmgr_v#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
    on_intel do
      url "https://github.com/acchapm1/ocmgr-app/releases/download/v#{version}/ocmgr_v#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "ocmgr"
  end

  test do
    assert_match "OpenCode Profile Manager", shell_output("#{bin}/ocmgr --help")
  end
end
