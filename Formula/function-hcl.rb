class FnHclTools < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0-rc3.tar.gz"
  sha256 "d45f7476e1d69826d7f35077b91f4a73ea9b534c87a9ab3ecb997a0fe53d6047"
  version "0.2.0-rc3"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    commit = Utils.git_short_head
    cd "function-hcl" do
      ldflags = %W[
        -X main.Version=#{version}
        -X main.Commit=#{commit}
        -X main.BuildDate=#{time.iso8601}
      ]
      system "go", "build", *std_go_args(ldflags:, output: bin/"fn-hcl-tools"), "./cmd/fn-hcl-tools"
    end
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fn-hcl-tools version")
  end
end
