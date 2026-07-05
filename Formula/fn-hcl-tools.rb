class FnHclTools < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "93c6caa4998e6f27004e51f203bca4af37af8b8ddd1d412c0a763641c2466a18"
  version "0.2.0-rc12"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    cd "function" do
      ldflags = %W[
        -X main.Version=0.2.0-rc12
        -X main.Commit=2e39102
        -X main.BuildDate=2026-07-03T17:59:41Z
      ]
      system "go", "build", *std_go_args(ldflags:, output: bin/"fn-hcl-tools"), "./cmd/fn-hcl-tools"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fn-hcl-tools version")
  end
end
