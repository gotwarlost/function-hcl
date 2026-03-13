---
title: "Resource Blocks"
linkTitle: "Resource Blocks"
weight: 5
description: >
  Declaring individual composed resources.
---

The `resource` block is the primary way to declare a desired composed resource.

## Syntax

```hcl
resource <crossplane-name> {
  locals { ... }          # optional: resource-scoped locals

  body = { ... }          # required: the Kubernetes manifest

  composite status { }    # optional: write to composite status
  composite connection { } # optional: write connection details
  ready { }               # optional: set readiness
}
```

## The `body` Attribute

`body` is an HCL object expression that produces the full Kubernetes manifest for the composed resource.
It must be assigned with `=`.

```hcl
resource my-s3-bucket {
  locals {
    resourceName = "${req.composite.metadata.name}-bucket"
    params       = req.composite.spec.parameters
    tagValues    = { foo = "bar" }
  }

  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata = {
      name = resourceName
    }
    spec = {
      forProvider = {
        forceDestroy = true
        region       = params.region
        tags         = tagValues
      }
    }
  }
}
```

## The `self` Variable

Inside a `resource` block, the `self` variable gives you access to the resource's own metadata
and observed state. It is available in the resource body and all nested blocks (`composite status`,
`composite connection`, etc.).

| Expression | Type | Description |
|-----------|------|-------------|
| `self.name` | string | The crossplane name of this resource (e.g. `"my-s3-bucket"`) |
| `self.resource` | object or incomplete | The observed version of this resource from the previous reconcile cycle |
| `self.connection` | map or incomplete | The connection details of this resource |

### Accessing Observed State

`self.resource` is the most common way to access status fields written by the provider after
the resource is created:

```hcl
resource my-bucket {
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata   = { name = "${req.composite.metadata.name}-bucket" }
    spec       = { forProvider = { region = req.composite.spec.parameters.region } }
  }

  composite status {
    body = {
      bucketARN = self.resource.status.atProvider.arn
    }
  }
}
```

If the resource hasn't been created yet, `self.resource` is incomplete, and any block that references
it will be [automatically deferred](../../concepts/dependency-resolution/).

### Accessing Other Resources

You're not limited to `self.resource`. You can reference any observed resource via `req.resource`:

```hcl
resource my-subnet {
  body = {
    # ...
    spec = {
      forProvider = {
        vpcId = req.resource.my-vpc.status.atProvider.id
      }
    }
  }
}
```
