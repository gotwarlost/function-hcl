# Maintainer Guide

## Repository layout

This is a monorepo with two Go modules:

| Directory | Go module | Description |
|-----------|-----------|-------------|
| `function/` | `github.com/crossplane-contrib/function-hcl/function` | Crossplane composition function |
| `language-server/` | `github.com/crossplane-contrib/function-hcl/language-server` | LSP language server |

Other top-level directories (`vscode/`, `jetbrains/`, `docs-site/`, `Formula/`) are not Go modules and are released as part of the same workflow.

## Release process

Releases are driven by the `scripts/tag-release.sh` script. Pushing the resulting tags triggers the GitHub Actions release workflow (`.github/workflows/release.yaml`), which builds all artifacts and publishes them.

### Prerequisites

- GPG signing key configured in git (`git config user.signingkey`)
- Write access to the repository (push tags permission)
- Clean working tree on `main`

### Step 1 — Run the tagging script

From the repo root:

```bash
./scripts/tag-release.sh v0.3.0-rc1
```

The script:
1. Validates the tag format (`vX.Y.Z-rcN` for release candidates, `vX.Y.Z` for GA)
2. Updates `docs-site/hugo.toml` with `latest_version`
3. Updates `Formula/fn-hcl-tools.rb` with the new tarball URL
4. Creates a signed commit with those two file changes
5. Creates three signed git tags:
   - `v0.3.0-rc1` — the primary release tag (triggers CI)
   - `function/v0.3.0-rc1` — Go module proxy tag for `function/`
   - `language-server/v0.3.0-rc1` — Go module proxy tag for `language-server/`

The script prints the push command on completion.

### Step 2 — Push tags

```bash
git push origin main v0.3.0-rc1 function/v0.3.0-rc1 language-server/v0.3.0-rc1
```

All three tags must be pushed together. The `v*` tag triggers the release workflow; the module-prefixed tags allow consumers to `go get` specific module versions via the Go module proxy.

### Step 3 — Monitor the release workflow

The workflow (`.github/workflows/release.yaml`) runs automatically on the `v*` tag and:

1. **`compute-versions`** — derives two version strings from the tag:
   - `tag_version`: the tag without the leading `v` (e.g. `0.3.0-rc1`)
   - `artifact_version`: a pure semver for marketplace submissions. For RCs, `vX.Y.Z-rcN` is encoded as `X.0.(1000*Y + 100*Z + N)` to satisfy VS Code and JetBrains version constraints (e.g. `v0.3.0-rc1` → `0.0.3001`). GA releases pass through unchanged.

2. **`build`** — cross-compiles binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64`:
   - `fn-hcl-tools` from `function/cmd/fn-hcl-tools`
   - `function-hcl-ls` from `language-server/`
   - Each binary is archived as `<name>-<os>-<arch>.tar.gz` with a per-archive checksum, plus a combined `checksums.txt`

3. **`vscode-package`** — packages the VS Code extension as `function-hcl-vscode-<version>.vsix`

4. **`jetbrains-package`** — builds the JetBrains plugin as `function-hcl-jetbrains-<version>.zip`

5. **`release`** — creates the GitHub Release with auto-generated notes and attaches all artifacts

6. **`vscode-publish`** — publishes the `.vsix` to the VS Code Marketplace (requires `VSCE_PAT` secret)

7. **`jetbrains-publish`** — publishes the plugin to the JetBrains Marketplace (requires `JETBRAINS_PUBLISH_TOKEN`, `JETBRAINS_CERTIFICATE_CHAIN`, `JETBRAINS_PRIVATE_KEY`, `JETBRAINS_PRIVATE_KEY_PASSWORD` secrets)

### Step 4 — Update the Homebrew formula

After the release is published and the tarball is available on GitHub, run:

```bash
./scripts/update-formula.sh v0.3.0-rc1
```

This fetches the tarball to compute its SHA256, writes a versioned formula (`Formula/fn-hcl-tools@0.3.0-rc1.rb`), updates the canonical formula (`Formula/fn-hcl-tools.rb`), and commits both. Push the commit to `main`.

> **Note:** The script requires the tag to already be pushed to GitHub so it can fetch the tarball.

## Re-tagging (amending a release)

If you need to move a tag (e.g. to include a last-minute fix), pass `--force`:

```bash
./scripts/tag-release.sh --force v0.3.0-rc1
```

Then push with `--force`:

```bash
git push origin main v0.3.0-rc1 function/v0.3.0-rc1 language-server/v0.3.0-rc1 --force
```

> **Warning:** Force-pushing tags rewrites history visible to anyone who has already fetched them. The Go module proxy caches module versions permanently — once `function/v0.3.0-rc1` is fetched via the proxy it cannot be updated. Force-retagging is only safe before the tag has been published or consumed.

## Required repository secrets

| Secret | Used by |
|--------|---------|
| `XPKG_ACCESS_ID` | Push Crossplane package to xpkg.upbound.io |
| `XPKG_TOKEN` | Push Crossplane package to xpkg.upbound.io |
| `VSCE_PAT` | Publish VS Code extension |
| `JETBRAINS_PUBLISH_TOKEN` | Publish JetBrains plugin |
| `JETBRAINS_CERTIFICATE_CHAIN` | Sign JetBrains plugin |
| `JETBRAINS_PRIVATE_KEY` | Sign JetBrains plugin |
| `JETBRAINS_PRIVATE_KEY_PASSWORD` | Sign JetBrains plugin |

## Go module tagging

Both Go modules in this repo require module-prefixed tags so the Go module proxy can serve them independently:

```
function/vX.Y.Z          → github.com/crossplane-contrib/function-hcl/function
language-server/vX.Y.Z   → github.com/crossplane-contrib/function-hcl/language-server
```

The tagging script creates these automatically. If you ever need to tag a module independently (e.g. a patch to `language-server` only), create the module tag manually — but note that the release workflow only fires on the bare `v*` tag, so a module-only tag will not trigger a release build.
