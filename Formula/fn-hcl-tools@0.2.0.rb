class FnHclToolsAT0_2_0 < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "8a822b8d63e28374e0697335e2046f877cdb1dadc9ee86196834aab3872b2bce"
  version "0.2.0"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    cd "function" do
      ldflags = %W[
        -X main.Version=0.2.0
        -X main.Commit=ee529ce
        -X main.BuildDate=2026-07-05T04:53:21Z
      ]
      system "go", "build", *std_go_args(ldflags:, output: bin/"fn-hcl-tools"), "./cmd/fn-hcl-tools"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fn-hcl-tools version")
  end
end
