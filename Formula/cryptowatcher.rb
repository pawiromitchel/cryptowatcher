class Cryptowatcher < Formula
  desc "Real-time TUI cryptocurrency and stocks dashboard inspired by macOS Widgets"
  homepage "https://github.com/pawiromitchel/cryptowatcher"
  url "https://github.com/pawiromitchel/cryptowatcher/archive/refs/tags/v1.1.0.tar.gz"
  sha256 "1cea357562d23157f59804182e13b6a8024b195dfb5286184515dc68ed63d558"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/cryptowatcher"
  end

  test do
    assert_match "config.json", shell_output("#{bin}/cryptowatcher -config-path")
  end
end
