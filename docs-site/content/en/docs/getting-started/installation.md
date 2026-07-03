---
title: "Installation"
linkTitle: "Installation"
weight: 1
description: >
  Install function-hcl into your Crossplane cluster.
---

## Prerequisites

Before installing function-hcl you need:

- A Kubernetes cluster with [Crossplane](https://docs.crossplane.io/latest/software/install/) v1.14 or later installed.
- `kubectl` configured to point at your cluster.

## Install via Crossplane Package

The recommended way to install function-hcl is as a Crossplane Function package:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Function
metadata:
  name: function-hcl
spec:
  package: xpkg.upbound.io/crossplane-contrib/function-hcl:{{< version >}}
```

Apply it to your cluster:

```bash
kubectl apply -f function.yaml
```

Verify the function is healthy:

```bash
kubectl get function function-hcl
```

You should see `HEALTHY: True` and `INSTALLED: True` in the output.

## Install fn-hcl-tools

`fn-hcl-tools` is the companion CLI for packaging, formatting, and analyzing your HCL files.

### Prebuilt release

Install it by downloading the [appropriate file for your OS for the latest release](https://github.com/crossplane-contrib/function-hcl/releases)

### Homebrew

For MacOS, you can also use homebrew to install it.

```bash
# since the formula comes from the same monorepo, you need to run the `tap` subcommand as follows
brew tap crossplane-contrib/function-hcl https://github.com/crossplane-contrib/function-hcl
brew trust crossplane-contrib/function-hcl
brew install fn-hcl-tools
```

To upgrade the version:

```bash
brew update
brew upgrade fn-hcl-tools
```

### Install from source

```bash
go install github.com/crossplane-contrib/function-hcl/function/cmd/fn-hcl-tools@{{< version >}}
```

Note that the version printed by `fn-hcl-tools version` will be incorrect using this method.

Verify it works:

```bash
fn-hcl-tools version
```

## Verify Installation

Check that the function pod is running:

```bash
kubectl get pods -n crossplane-system -l pkg.crossplane.io/revision=function-hcl
```

## Upgrading

To upgrade to a new version, update the `spec.package` tag in your Function manifest and re-apply:

```bash
kubectl apply -f function.yaml
```

Crossplane will pull the new image and roll out the update automatically.
