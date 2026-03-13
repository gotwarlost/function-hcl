---
title: "fn-hcl-tools CLI"
linkTitle: "fn-hcl-tools"
weight: 4
description: >
  CLI tooling for packaging, formatting, and analyzing HCL files.
---

`fn-hcl-tools` is a companion CLI for function-hcl that helps package, format, and analyze
HCL compositions.

## Installation

```bash
go install github.com/crossplane-contrib/function-hcl/function-hcl/cmd/fn-hcl-tools@{{< version >}}
```

## Commands

### `package`

Packages a directory of HCL files into a single txtar bundle, suitable for embedding in a Composition's
`input` field.

```bash
fn-hcl-tools package *.hcl
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
fn-hcl-tools package my-composition/*.hcl
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
fn-hcl-tools fmt <file-or-directory> [... <file-or-dorectory>]
```

This formats all HCL files in-place.

If you specify `-` as the file name it behaves like a filter, reading from stdin and writing to stdout.

The `--check` option allows you to check formatting of the supplied files. This will exit with an error code
if the supplied files are not correctly formatted.

### `analyze`

Analyzes HCL syntax files and reports diagnostics.

```bash
fn-hcl-tools analyze *.hcl
```

### `version`

Displays the tool version.

```bash
fn-hcl-tools version
```

## Workflow

A typical development workflow:

1. Write your HCL in separate files in a local directory.
2. Format with `fn-hcl-tools fmt`.
3. Package to txtar with `fn-hcl-tools package`.
4. Embed the txtar output in your Composition YAML.
5. Test locally using `crossplane beta render`.
6. Apply to your cluster.

```bash
# Package and copy to clipboard (macOS)
fn-hcl-tools package ./my-composition/*.hcl | pbcopy
```
