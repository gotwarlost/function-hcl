class FnHclTools < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0-rc9.tar.gz"
  sha256 "b68bf7cec176ea35cf6dc0fbe7e2e26540080449ee407575fedafcfd8cfc19ab"
  version "0.2.0-rc5"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    cd "function" do
      ldflags = %W[
        -X main.Version=0.2.0-rc5
        -X main.Commit=83a5ab6
        -X main.BuildDate=2026-03-15T23:08:18Z
      ]
      system "go", "build", *std_go_args(ldflags:, output: bin/"fn-hcl-tools"), "./cmd/fn-hcl-tools"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fn-hcl-tools version")
  end
end
