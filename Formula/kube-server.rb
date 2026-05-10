class KubeServer < Formula
  desc "Lightweight Kubernetes cluster manager for macOS"
  homepage "https://github.com/jackymint/kube-server"
  version "0.1.1"

  on_arm do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-arm64.tar.gz"
    sha256 "2b8eb7a7eec8eeba188dc9e53034a623c159db8af60ec53aba6d662092eeb8d8"
  end

  on_intel do
    url "https://github.com/jackymint/kube-server/releases/download/v#{version}/kube-server-darwin-amd64.tar.gz"
    sha256 "ccfd59798a12bb4c837f29ba2dc62e782c181b36cbb0b3dc6d3fff5975092b93"
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
