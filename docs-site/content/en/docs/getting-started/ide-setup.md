---
title: "IDE Setup"
linkTitle: "IDE Setup"
weight: 3
description: >
  Set up your editor for function-hcl development with language server support.
---

function-hcl has a language server (`function-hcl-ls`) that provides IDE features for `.hcl` files
used in function-hcl compositions. Extensions are available for VS Code and IntelliJ.

## Features

The language server provides:
- Code completion
- Diagnostics (error reporting)
- Hover information
- Go to definition
- Document symbols
- Semantic token highlighting

## VS Code

### Install the Extension

The function-hcl VS Code extension is available in the
[function-hcl-vscode-extension](https://github.com/crossplane-contrib/function-hcl/tree/main/function-hcl-vscode-extension)
directory of the repository.

The extension:
- Provides syntax highlighting for `.hcl` files
- Connects to the `function-hcl-ls` language server
- Supports configuring a custom language server path

### Configuration

The extension provides these settings:

| Setting | Description |
|---------|-------------|
| Language server path | Path to a custom `function-hcl-ls` binary |
| Language server version | Version of the language server to use |

## IntelliJ

An IntelliJ plugin is available in the
[function-hcl-intellij](https://github.com/crossplane-contrib/function-hcl/tree/main/function-hcl-intellij)
directory. It uses [LSP4IJ](https://github.com/redhat-developer/lsp4ij) for language server integration.

{{% alert title="Work in progress" color="warning" %}}
The IntelliJ plugin is in early development. Some features may not yet be available.
{{% /alert %}}

## Building the Language Server

If you need to build `function-hcl-ls` from source:

```bash
cd function-hcl-ls
go build -o function-hcl-ls .
```

Place the resulting binary on your `PATH` or configure your IDE extension to point to it.
