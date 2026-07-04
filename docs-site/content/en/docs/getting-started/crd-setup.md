---
title: "Set up CRDs"
linkTitle: "CRD setup"
weight: 4
description: >
  Create local CRD definitions for language server use.
---

The language server provides completion for language constructs by default.
To enable completions for XRD and CRD fields, you need to supply two pieces of information:

1. **The composite type** — tells the language server which XRD this composition targets, enabling
   completions for the `req.composite` variable.
2. **CRD definitions** — tells the language server what resource types exist, enabling completions
   for `apiVersion`, `kind`, and resource-specific fields.

## Declaring the composite type

Create a `composition.yaml` file in the same directory as your `.hcl` files.
This file tells the language server which XRD your composition targets:

```yaml
xrd:
  apiVersion: example.com/v1
  kind: XExample
```

The language server reads this file when you open any `.hcl` file in the directory.
It uses the `apiVersion` and `kind` to look up the matching XRD schema from the CRD definitions you have
configured (see below), and uses that schema to provide completions for the built-in `composite` variable.

If `composition.yaml` is absent, or if either field is empty, the language server still works but cannot
provide composite-specific field completions.

## Setting up CRD definitions

You use the `fn-hcl-tools extract-crds` command to extract CRD definitions from your YAML files and OCI images,
then place those definitions where the language server can find them.

## How the language server discovers CRDs

When you open an HCL file, the language server walks up the directory tree from the file's location
looking for one of two things (in this order):

1. A file named `.crd-sources.yaml` — an explicit configuration listing which YAML files to load.
2. A directory named `.crds/` — the default drop location for extracted CRD files.

Once found, it loads matching YAML files in the background and starts providing completions for the
resource types it finds. It watches for file changes and reloads automatically.

## Extracting CRDs

`fn-hcl-tools extract-crds` reads YAML files and pulls out CRD and XRD definitions. It can also
follow `Provider` and `Configuration` object references and pull CRDs from the referenced OCI images.

**Install `fn-hcl-tools`** if you have not already — see [Installation](../installation#install-fn-hcl-tools).

### From local YAML files

Point the command at any YAML files that contain CRDs, XRDs, Providers, or Configurations:

```bash
fn-hcl-tools extract-crds --output-dir .crds providers.yaml crossplane.yaml
```

The tool writes one YAML file per processed image into `.crds/`. Objects from local files
are grouped under a file named `local-objects.yaml`.

### From stdin (filter mode)

When called with no arguments, the command acts as a filter: reads YAML from stdin and writes to stdout.
Use `-` to explicitly request stdin while also passing file arguments:

```bash
cat providers.yaml | fn-hcl-tools extract-crds --output-dir .crds -
fn-hcl-tools extract-crds --output-dir .crds - extra.yaml < providers.yaml
```

Note: the tool processes individual YAML documents, not Kubernetes `List` objects.
If your source produces a `List`, extract the individual items first before piping.

### From a Helm chart

`helm template` renders chart manifests as individual YAML documents, which the extractor handles
directly. This is useful for pulling CRDs out of provider charts:

```bash
helm template --include-crds my-release oci://registry-1.docker.io/bitnamicharts/crossplane \
  | fn-hcl-tools extract-crds --output-dir .crds -
```

```bash
helm template my-release ./my-provider-chart --include-crds \
  | fn-hcl-tools extract-crds --output-dir .crds -
```

## Simple setup: the `.crds/` directory

The simplest configuration is to create a `.crds/` directory in (or above) your compositions directory
and run `extract-crds` into it. No further configuration is needed.

```
my-compositions/
  .crds/
    local-objects.yaml      # extracted from local YAML files
    xpkg-upbound-io-...yaml # extracted from a Provider image
   compositions/
      composition1/
         composition.yaml
         main.hcl
```

The language server loads all `*.yaml` files from `.crds/` and makes all resource types
(both cluster-scoped and namespace-scoped) available for completion.

## Advanced setup: `.crd-sources.yaml`

Create a `.crd-sources.yaml` file when you need more control — for example, to load files from
a different location or to limit completions to a specific resource scope.

```yaml
scope: both        # "namespaced", "cluster", or "both" (default: "both")
paths:
  - .crds/*.yaml   # paths are relative to this file's directory
  - /absolute/path/to/more/crds/**/*.yaml
```

| Field   | Values                              | Default  | Description                                           |
|---------|-------------------------------------|----------|-------------------------------------------------------|
| `scope` | `namespaced`, `cluster`, `both`     | `both`   | Which resource scopes to include in completions.      |
| `paths` | list of file paths or glob patterns | required | Files to load; relative paths resolve from this file. |

The `paths` field supports `**` double-star globs for recursive matching.

### Example: separate CRD directories per scope

```yaml
scope: cluster
paths:
  - cluster-crds/*.yaml
  - /shared/xrds/*.yaml
```

## Placement

Put `.crds/` or `.crd-sources.yaml` in the root of your compositions repository so that all
composition files in subdirectories share a single CRD configuration.

```
repo-root/
  .crds/                      # or .crd-sources.yaml here
    providers.yaml
    xrds.yaml
  compositions/
    postgres/
      composition.hcl         # language server finds .crds/ by walking up
    redis/
      composition.hcl         # same
```

## Keeping CRDs up to date

Re-run `extract-crds` whenever you add a new Provider, Configuration, or XRD to your setup.
The language server detects file changes and reloads without requiring a restart.

```bash
fn-hcl-tools extract-crds --output-dir .crds crossplane.yaml
```

Use `--progress=false` and `--warnings=false` to suppress diagnostic output in scripts.
