---
title: "Source Format"
linkTitle: "Source Format"
weight: 1
description: >
  How HCL source code is packaged and provided as input to function-hcl.
---

## HCL Syntax

function-hcl source code is written in [HCL syntax](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md).
Code may be spread across multiple files -- all files are treated as one unit. The only difference between
having multiple files versus a single file is that line numbers will differ in error messages.

## txtar Format

The function accepts its input as text in the `input` field of a Composition pipeline step.
This **must** be in [txtar format](https://pkg.go.dev/golang.org/x/tools/txtar#hdr-Txtar_format)
so that original file names are maintained and line numbers match the source code.

A txtar bundle looks like this:

```
-- main.hcl --

locals {
  params = req.composite.spec.parameters
}

-- bucket.hcl --

resource my-bucket {
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata   = { name = "${params.name}-bucket" }
    spec       = { forProvider = { region = params.region } }
  }
}
```

Each file section starts with `-- filename.hcl --` on its own line.

## Using txtar in a Composition

Embed the txtar bundle in the `input` field of your pipeline step:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: example
spec:
  compositeTypeRef:
    apiVersion: example.crossplane.io/v1
    kind: XR
  mode: Pipeline
  pipeline:
  - step: create-resources
    functionRef:
      name: function-hcl
    input: |
      -- main.hcl --
      locals {
        name = req.composite.metadata.name
      }
      -- bucket.hcl --
      resource my-bucket {
        body = { ... }
      }
```

## Packaging with fn-hcl-tools

You should always use the `fn-hcl-tools package` command to produce the txtar script from your
HCL source files -- even for a single file. Do not hand-craft txtar bundles.

```bash
fn-hcl-tools package ./my-composition/*.hcl
```

Before producing the txtar output, `fn-hcl-tools package` **analyzes** every HCL file and will
abort with errors if it finds problems. This catches issues at authoring time rather than at
runtime in the cluster:

- Typos in variable names (e.g. referencing `parms` instead of `params`)
- Bad block structure (e.g. a `resource` block missing its `body` attribute)
- Invalid HCL syntax
- Incorrect nesting of blocks

{{% alert title="Recommended" color="info" %}}
Always run `fn-hcl-tools package` as part of your workflow. It acts as a linter and packager
in one step, giving you fast feedback before you embed the script in a Composition YAML and
apply it to a cluster.
{{% /alert %}}

See the [fn-hcl-tools CLI reference](../../reference/fn-hcl-tools/) for all available commands.
