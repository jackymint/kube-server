class KubeServer < Formula
  desc "Lightweight Kubernetes cluster manager for macOS"
  homepage "https://github.com/jackymint/kube-server"
  version "0.1.3"

  on_arm do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-arm64.tar.gz"
    sha256 "8ce5c0e39ad120f9313110a7fef3f1ec87e866e30623cc1232275d6d27a9b191"
  end

  on_intel do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-amd64.tar.gz"
    sha256 "2a2b536b7d15e8e1f6c7121213bcd22d3857199e733f3648d48b12f011d8c480"
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
