#!/usr/bin/env bash
set -euo pipefail

force=false
tag=""

while [ $# -gt 0 ]; do
  case "$1" in
    --force|-f) force=true; shift ;;
    -*)         echo "Unknown option: $1" >&2; exit 1 ;;
    *)          tag="$1"; shift ;;
  esac
done

if [ -z "$tag" ]; then
  echo "Usage: $0 [--force] <tag>" >&2
  echo "Example: $0 v0.2.0" >&2
  exit 1
fi

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9](-rc[0-9]+)+$ ]]; then
  echo "Error: tag must match vX.Y.Z (got: $tag)" >&2
  exit 1
fi

module_tag="function-hcl/${tag}"
ls_module_tag="function-hcl-ls/${tag}"

# Ensure clean working tree
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is dirty. Commit or stash changes first." >&2
  exit 1
fi

# Check for existing tags
if [ "$force" = false ]; then
  if git rev-parse "$tag" >/dev/null 2>&1; then
    echo "Error: tag $tag already exists. Use --force to update it." >&2
    exit 1
  fi
  if git rev-parse "$module_tag" >/dev/null 2>&1; then
    echo "Error: tag $module_tag already exists. Use --force to update it." >&2
    exit 1
  fi
  if git rev-parse "$ls_module_tag" >/dev/null 2>&1; then
    echo "Error: tag $ls_module_tag already exists. Use --force to update it." >&2
    exit 1
  fi
fi

root="$(git rev-parse --show-toplevel)"

# 1. Update hugo.toml
sed -i '' "s|latest_version = \".*\"|latest_version = \"${tag}\"|" "$root/docs-site/hugo.toml"
echo "Updated docs-site/hugo.toml → latest_version = \"${tag}\""

# 2. Update Formula
sed -i '' "s|url \"https://github.com/.*/archive/refs/tags/.*\.tar\.gz\"|url \"https://github.com/crossplane-contrib/function-hcl/archive/refs/tags/${tag}.tar.gz\"|" "$root/Formula/fn-hcl-tools.rb"
echo "Updated Formula/fn-hcl-tools.rb → url for ${tag}"

# 3. Commit
git add "$root/docs-site/hugo.toml" "$root/Formula/fn-hcl-tools.rb"
git commit -s -m "Release ${tag}"

# 4. Create tags (force-update if --force)
tag_flags="-s"
if [ "$force" = true ]; then
  tag_flags="-s -f"
fi

git tag $tag_flags "$tag" -m "Release ${tag}"
echo "Created tag: ${tag}"

git tag $tag_flags "$module_tag" -m "Release ${module_tag}"
echo "Created tag: ${module_tag}"

git tag $tag_flags "$ls_module_tag" -m "Release ${ls_module_tag}"
echo "Created tag: ${ls_module_tag}"

echo ""
echo "Done. Push with:"
if [ "$force" = true ]; then
  echo "  git push origin main ${tag} ${module_tag} ${ls_module_tag} --force"
else
  echo "  git push origin main ${tag} ${module_tag} ${ls_module_tag}"
fi
