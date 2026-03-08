---
title: "fn-hcl-tools CLI"
linkTitle: "fn-hcl-tools"
weight: 4
description: >
  CLI tooling for packaging, formatting, and analyzing HCL files.
---

`fn-hcl-tools` is a companion CLI for function-hcl that helps package, format, and analyze
HCL compositions.

{{% alert title="Early stage" color="warning" %}}
`fn-hcl-tools` is early-stage tooling. The interface may change in future releases.
{{% /alert %}}

## Installation

```bash
go install github.com/crossplane-contrib/function-hcl/cmd/fn-hcl-tools@latest
```

## Commands

### `pack`

Packages a directory of HCL files into a single txtar bundle, suitable for embedding in a Composition's
`input` field.

```bash
fn-hcl-tools pack <directory>
```

The tool runs basic static analysis on the HCL before packing, catching syntax errors early.

**Example:**

Given a directory:

```
my-composition/
  main.hcl
  bucket.hcl
  database.hcl
```

Run:

```bash
fn-hcl-tools pack my-composition/
```

Output (suitable for pasting into a Composition YAML `input` field):

```
-- main.hcl --
...contents of main.hcl...

-- bucket.hcl --
...contents of bucket.hcl...

-- database.hcl --
...contents of database.hcl...
```

### `format`

Formats HCL files.

```bash
fn-hcl-tools format <file-or-directory>
```

### `analyze`

Analyzes HCL syntax files and reports diagnostics.

```bash
fn-hcl-tools analyze <file-or-directory>
```

### `version`

Displays the tool version.

```bash
fn-hcl-tools version
```

## Workflow

A typical development workflow:

1. Write your HCL in separate files in a local directory.
2. Format with `fn-hcl-tools format`.
3. Analyze with `fn-hcl-tools analyze` to catch issues early.
4. Test locally using `crossplane beta render`.
5. Pack to txtar with `fn-hcl-tools pack`.
6. Embed the txtar output in your Composition YAML.
7. Apply to your cluster.

```bash
# Pack and copy to clipboard (macOS)
fn-hcl-tools pack ./my-composition | pbcopy
```
