---
title: "composition.yaml"
linkTitle: "composition.yaml"
weight: 5
description: >
  Reference for the composition.yaml metadata file.
---

`composition.yaml` is an optional metadata file placed in the same directory as your `.hcl` files.
It tells function-hcl tools and the language server about the composition's composite type and any
shared library files the composition depends on.

## Location

Place `composition.yaml` in the root of your composition directory, alongside your `.hcl` files:

```
my-composition/
  composition.yaml
  main.hcl
  resources.hcl
```

## Fields

```yaml
xrd:
  apiVersion: <string>
  kind: <string>

libraryFiles:
  - <relative-path>
```

### `xrd`

Declares the composite type (XRD) that this composition targets.
The language server uses these values to look up the XRD schema and provide completions
for the built-in `composite` variable.

Both fields are optional but must both be non-empty for the language server to use them.

| Field        | Type   | Description                                          |
|--------------|--------|------------------------------------------------------|
| `apiVersion` | string | API version of the composite type, e.g. `example.io/v1` |
| `kind`       | string | Kind of the composite type, e.g. `XPostgresInstance` |

### `libraryFiles`

A list of HCL files outside the composition directory that are included when the composition
is packaged with `fn-hcl-tools package`. Paths are relative to the `composition.yaml` file
and must not be absolute. Directories are not allowed.

This is useful for sharing common HCL helpers across multiple compositions:

```yaml
libraryFiles:
  - ../shared/helpers.hcl
  - ../shared/tags.hcl
```

## Example

```yaml
xrd:
  apiVersion: database.example.io/v1alpha1
  kind: XPostgresInstance

libraryFiles:
  - ../lib/common-tags.hcl
```

## Behaviour when absent

If `composition.yaml` is absent, function-hcl tools process all `.hcl` files in the directory
with no library files and no XRD type information. The language server still provides completions
for resource types, but cannot provide completions for `composite` fields.
