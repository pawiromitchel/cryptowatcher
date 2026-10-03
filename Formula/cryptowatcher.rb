class Cryptowatcher < Formula
  desc "Real-time TUI cryptocurrency and stocks dashboard inspired by macOS Widgets"
  homepage "https://github.com/pawiromitchel/cryptowatcher"
  license "MIT"

  livecheck do
    url :stable
    strategy :github_latest
  end

  on_macos do
    on_arm do
      url "https://github.com/pawiromitchel/cryptowatcher/releases/download/v1.2.0/cryptowatcher-darwin-arm64.tar.gz"
      sha256 "574948aec984e5e98a1720dbd8e53863a2f83bd7334d06394e45437090a5527a"
    end
    on_intel do
      url "https://github.com/pawiromitchel/cryptowatcher/releases/download/v1.2.0/cryptowatcher-darwin-amd64.tar.gz"
      sha256 "c776e6c3fda9abef996c9cd1ad7b06737e455ed7c40a9ea7fa3293219619f497"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/pawiromitchel/cryptowatcher/releases/download/v1.2.0/cryptowatcher-linux-arm64.tar.gz"
      sha256 "7ad5424c20fd45eb75526826d2c7651a9db12ecfa16b84f8dca3a950a8082517"
    end
    on_intel do
      url "https://github.com/pawiromitchel/cryptowatcher/releases/download/v1.2.0/cryptowatcher-linux-amd64.tar.gz"
      sha256 "3d0942b49a21377f462015dde47f620d19005f3519b6c994a48af91d2bfeb94e"
    end
  end

  def install
    # The release tarball holds a single binary named cryptowatcher-<os>-<arch>.
    bin.install Dir["cryptowatcher-*"].first => "cryptowatcher"
  end

  test do
    assert_match "config.json", shell_output("#{bin}/cryptowatcher -config-path")
    assert_match version.to_s, shell_output("#{bin}/cryptowatcher -version")
  end
end
