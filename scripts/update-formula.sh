#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# update-formula.sh
#
# Creates a new versioned Homebrew formula for function-hcl and updates the
# canonical (unversioned) formula to point at the same release.
#
# Usage:
#   ./scripts/update-formula.sh          # auto-detects latest vX.Y.Z tag
#   ./scripts/update-formula.sh v0.1.0   # uses the specified tag
# ---------------------------------------------------------------------------

REPO="crossplane-contrib/function-hcl"
BINARY="function-hcl"
FORMULA_DIR="Formula"
GITHUB_URL="https://github.com/${REPO}"

# Ruby class name
CLASS_NAME="FnHclTools"

# ---------------------------------------------------------------------------
# Resolve tag
# ---------------------------------------------------------------------------

if [[ $# -ge 1 ]]; then
  TAG="$1"
else
  # Auto-detect: find the latest tag matching vX.Y.Z only (ignore module proxy tags)
  TAG=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname | head -n1)
  if [[ -z "$TAG" ]]; then
    echo "error: no vX.Y.Z tags found in this repo" >&2
    exit 1
  fi
  echo "Auto-detected tag: ${TAG}"
fi

# Validate tag format
if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: tag '${TAG}' is not in vX.Y.Z format" >&2
  exit 1
fi

# Strip leading 'v' for use in formula version field
VERSION="${TAG#v}"

# ---------------------------------------------------------------------------
# Sanity checks
# ---------------------------------------------------------------------------

# Confirm the tag exists in the repo
if ! git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "error: tag '${TAG}' does not exist in this repo" >&2
  exit 1
fi

# Confirm working tree is clean
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "error: working tree is not clean — commit or stash changes before running this script" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Compute SHA256 of the GitHub-generated source tarball for this tag
# ---------------------------------------------------------------------------

TARBALL_URL="${GITHUB_URL}/archive/refs/tags/${TAG}.tar.gz"
echo "Fetching tarball to compute SHA256: ${TARBALL_URL}"

SHA256=$(curl -sL "$TARBALL_URL" | shasum -a 256 | awk '{print $1}')

if [[ -z "$SHA256" ]]; then
  echo "error: failed to compute SHA256 — is the tag pushed to GitHub?" >&2
  exit 1
fi

echo "SHA256: ${SHA256}"

# ---------------------------------------------------------------------------
# Render formula content
# ---------------------------------------------------------------------------

render_formula() {
  local class="$1"
  cat <<EOF
class $class < Formula
  desc "CLI tools for function-hcl: format, analyze, and package HCL compositions"
  homepage "${GITHUB_URL}"
  url "${TARBALL_URL}"
  sha256 "${SHA256}"
  version "${VERSION}"
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
EOF
}

# ---------------------------------------------------------------------------
# Write versioned formula  e.g. Formula/fn-hc-tools@0.1.0.rb
# ---------------------------------------------------------------------------

VERSIONED_FILE="${FORMULA_DIR}/${BINARY}@${VERSION}.rb"
VERSIONED_CLASS="${CLASS_NAME}AT${VERSION//\./_}"  # FunctionHclToolsAT0_1_0

if [[ -f "$VERSIONED_FILE" ]]; then
  echo "error: ${VERSIONED_FILE} already exists — has this version already been released?" >&2
  exit 1
fi

echo "Writing ${VERSIONED_FILE}"
render_formula "$VERSIONED_CLASS" > "$VERSIONED_FILE"

# ---------------------------------------------------------------------------
# Update canonical formula  Formula/function-hcl.rb
# ---------------------------------------------------------------------------

CANONICAL_FILE="${FORMULA_DIR}/${BINARY}.rb"

echo "Updating ${CANONICAL_FILE}"
render_formula "$CLASS_NAME" > "$CANONICAL_FILE"

# ---------------------------------------------------------------------------
# Commit both files
# ---------------------------------------------------------------------------

git add "$VERSIONED_FILE" "$CANONICAL_FILE"
git commit -s -m "Formula: ${BINARY} ${TAG}"

echo ""
echo "Done. Next steps:"
echo "  1. Review the commit:  git show HEAD"
echo "  2. Push to main:       git push origin main"
echo ""
echo "Users can install with:"
echo "  brew tap ${REPO} ${GITHUB_URL}"
echo "  brew install ${REPO}/${BINARY}          # latest"
echo "  brew install ${REPO}/${BINARY}@${VERSION}  # this version"