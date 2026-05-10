class KubeServer < Formula
  desc "Lightweight Kubernetes cluster manager for macOS"
  homepage "https://github.com/jackymint/kube-server"
  version "0.1.0"

  on_arm do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-arm64.tar.gz"
    sha256 "PLACEHOLDER_ARM64_SHA256"
  end

  on_intel do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-amd64.tar.gz"
    sha256 "PLACEHOLDER_AMD64_SHA256"
  end

  depends_on :macos => :ventura
  depends_on "vfkit"
  depends_on "socket_vmnet"
  depends_on "qemu"

  def install
    bin.install "kube-server-darwin-arm64" => "kube-server" if Hardware::CPU.arm?
    bin.install "kube-server-darwin-amd64" => "kube-server" if Hardware::CPU.intel?
  end

  def post_install
    (var/"kube-server").mkpath
  end

  service do
    run [opt_bin/"kube-server"]
    keep_alive false
  end

  test do
    assert_match "kube-server", shell_output("#{bin}/kube-server --help")
  end
end
