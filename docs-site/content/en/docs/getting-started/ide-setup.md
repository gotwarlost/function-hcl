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

## Installation

### Visual Studio Code

Install the [Crossplane Function HCL](https://marketplace.visualstudio.com/items?itemName=function-hcl-authors.function-hcl) extension shown below.

{{< figure src="../vscode-extension.png"  >}}

### Jetbrains products

Install the [function-hcl plugin](https://plugins.jetbrains.com/plugin/30965-function-hcl) as shown below.

{{< figure src="../jetbrains-plugin.png"  >}}

### Post-install

Follow the instructions [on this page](../crd-setup) to register types with the language server.
