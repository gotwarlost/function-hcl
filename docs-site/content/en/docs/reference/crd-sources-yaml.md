---
title: ".crd-sources.yaml"
linkTitle: ".crd-sources.yaml"
weight: 6
description: >
  Reference for the .crd-sources.yaml language server configuration file.
---

`.crd-sources.yaml` is an optional configuration file that tells the language server where to find
CRD and XRD YAML files, and which resource scopes to include in completions.
It is the advanced alternative to placing files directly in a [`.crds/` directory](../../getting-started/crd-setup#simple-setup-the-crds-directory).

## Location

Place `.crd-sources.yaml` in the root of your compositions repository (or any ancestor directory).
The language server walks up the directory tree from the open file and uses the first
`.crd-sources.yaml` it finds. If none is found, it falls back to looking for a `.crds/` directory.

```
repo-root/
  .crd-sources.yaml   ← found for all compositions below
  compositions/
    postgres/
      composition.hcl
    redis/
      composition.hcl
```

## Fields

```yaml
scope: <"namespaced" | "cluster" | "both">

paths:
  - <path-or-glob>
```

### `scope`

Controls which resource scopes are included in completions.

| Value         | Description                                                  |
|---------------|--------------------------------------------------------------|
| `both`        | Include all resources regardless of scope. **(default)**    |
| `namespaced`  | Include only namespace-scoped resources.                     |
| `cluster`     | Include only cluster-scoped resources.                       |

When omitted, `both` is used.

### `paths`

A list of file paths or glob patterns pointing to YAML files that contain CRD or XRD definitions.
Relative paths are resolved from the directory that contains `.crd-sources.yaml`.
Absolute paths are also supported.

The `**` double-star glob matches across directory boundaries:

```yaml
paths:
  - .crds/*.yaml               # all YAML files directly in .crds/
  - crds/**/*.yaml             # all YAML files anywhere under crds/
  - /shared/platform-crds/*.yaml  # absolute path
```

Only `.yaml` files are loaded; other extensions are silently skipped.

## Example

```yaml
scope: both
paths:
  - .crds/*.yaml
  - ../shared/xrds/*.yaml
```

## Behaviour

The language server loads the matching files in the background when you open an HCL file.
It watches for changes to the matched files and reloads automatically — no restart required.

If `.crd-sources.yaml` exists but `paths` matches no files, completions for dynamic resource
types will be empty until matching files are added.
