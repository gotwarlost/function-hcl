# function-hcl Documentation Site

This directory contains the source for the [function-hcl documentation site](https://crossplane-contrib.github.io/function-hcl/),
built with [Hugo](https://gohugo.io/) and the [Docsy](https://www.docsy.dev/) theme.

## Prerequisites

- [Hugo Extended](https://gohugo.io/installation/) v0.110.0 or later
- [Go](https://go.dev/dl/) 1.21 or later (for Hugo Modules)
- [Node.js](https://nodejs.org/) 18 or later (for PostCSS)

## Local Development

```bash
# From this directory:
npm install
hugo server
```

Visit http://localhost:1313 to preview the site. Changes to content files are live-reloaded.

## Adding Content

All documentation lives under `content/en/docs/`. To add a new page:

```bash
# Create a new page in the getting-started section
hugo new docs/getting-started/my-new-page.md
```

Or just create a `.md` file directly in the appropriate folder.

### Front matter

Each page needs front matter at the top:

```yaml
---
title: "My Page Title"
linkTitle: "Short Title"  # used in sidebar nav
weight: 3                 # controls ordering within the section
description: >
  One-line description shown in section listings.
---
```

## Directory Structure

```
content/en/
  _index.md                  ← homepage
  docs/
    _index.md                ← docs landing page
    getting-started/
      _index.md
      installation.md
      quickstart.md
    concepts/
      _index.md
      how-it-works.md
      hcl-dsl.md
    reference/
      _index.md
      spec.md
      fn-hcl-tools.md
    examples/
      _index.md
      s3-bucket.md
      multi-resource.md
```

## Deployment

The site is deployed automatically via GitHub Actions when changes are pushed to `main` inside the
`docs-site/` directory. The workflow is at `.github/workflows/docs.yml`.

To enable GitHub Pages in the repository:
1. Go to **Settings → Pages**
2. Under **Source**, select **GitHub Actions**
