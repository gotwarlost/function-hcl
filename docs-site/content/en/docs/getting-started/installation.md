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
Install it with `go install`:

```bash
go install github.com/crossplane-contrib/function-hcl/function-hcl/cmd/fn-hcl-tools@{{< version >}}
```

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
