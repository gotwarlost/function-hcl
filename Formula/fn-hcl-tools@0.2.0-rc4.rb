class FnHclToolsAT0_2_0-rc4 < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  url "https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/v0.2.0-rc4.tar.gz"
  sha256 "89b01e6334a0c8a0a2453d34b352bb022086f3434b1d4302f2c7575d72c52320"
  version "0.2.0-rc4"
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
