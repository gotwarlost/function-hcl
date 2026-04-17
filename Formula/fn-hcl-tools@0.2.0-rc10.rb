class FnHclToolsAT0_2_0_rc10 < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0-rc10.tar.gz"
  sha256 "ce1d5b5c014d40f78a530d37005d1a0f5d6c346eb782d657a812a042760c6876"
  version "0.2.0-rc10"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    cd "function" do
      ldflags = %W[
        -X main.Version=0.2.0-rc10
        -X main.Commit=13a335f
        -X main.BuildDate=2026-04-17T20:34:48Z
      ]
      system "go", "build", *std_go_args(ldflags:, output: bin/"fn-hcl-tools"), "./cmd/fn-hcl-tools"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fn-hcl-tools version")
  end
end
