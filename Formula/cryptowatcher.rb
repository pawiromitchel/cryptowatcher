class Cryptowatcher < Formula
  desc "Real-time TUI cryptocurrency and stocks dashboard inspired by macOS Widgets"
  homepage "https://github.com/pawiromitchel/cryptowatcher"
  url "https://github.com/pawiromitchel/cryptowatcher/archive/refs/tags/v1.2.0.tar.gz"
  sha256 "eb3de94bab2ba944f6f816c0ec151fe1bfb19456e41176b34717c02473d75194"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=v#{version}"), "./cmd/cryptowatcher"
  end

  test do
    assert_match "config.json", shell_output("#{bin}/cryptowatcher -config-path")
    assert_match version.to_s, shell_output("#{bin}/cryptowatcher -version")
  end
end
