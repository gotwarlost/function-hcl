class FnHclTools < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "https://github.com/crossplane-contrib/function-hcl"
  license "Apache-2.0"
  head "https://github.com/crossplane-contrib/function-hcl.git", branch: "docs"

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
