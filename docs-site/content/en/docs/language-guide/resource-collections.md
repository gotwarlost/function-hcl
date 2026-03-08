---
title: "Resource Collections"
linkTitle: "Resource Collections"
weight: 6
description: >
  Creating multiple resources with for_each.
---

The `resources` (plural) block creates multiple resources by iterating over a collection.

## Syntax

```hcl
resources <base-name> {
  locals { ... }          # optional: collection-scoped locals

  for_each = <expression> # required: list, set, or map to iterate over

  name = <expression>     # optional: how to generate each crossplane name
                          # default: "${self.basename}-${each.key}"

  template {              # required: the template for each resource
    locals { ... }        # optional: template-scoped locals
    body = { ... }        # required: the Kubernetes manifest
    composite status { }  # optional
    composite connection { } # optional
    ready { }             # optional
  }
}
```

## Example

```hcl
resources additional_buckets {
  locals {
    params   = req.composite.spec.parameters
    suffixes = params.suffixes
  }

  for_each = suffixes

  template {
    locals {
      resourceName = "${req.composite.metadata.name}-${self.name}"
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
        }
      }
    }
  }
}
```

## The `for_each` Attribute

Must evaluate to a list, set, or map.

## The `name` Attribute

Controls how the crossplane name is generated for each resource in the collection. Defaults to
`"${self.basename}-${each.key}"`.

```hcl
resources my-buckets {
  for_each = ["prod", "staging"]
  name = "${self.basename}-${each.value}"  # produces "my-buckets-prod", "my-buckets-staging"

  template {
    body = { ... }
  }
}
```

## The `self` Variable

Inside a `resources` block, `self` provides collection-level metadata and observed state:

| Expression | Where available | Type | Description |
|-----------|----------------|------|-------------|
| `self.basename` | `name`, `template` | string | The name given to the `resources` block |
| `self.name` | `template` only | string | The generated crossplane name for the current resource |
| `self.resources` | `name`, `template` | list or incomplete | The observed resource collection |
| `self.connections` | `name`, `template` | list or incomplete | Connection details of the collection |

## The `each` Variable

Inside a `resources` block, `each` provides the current iterator state. It is available in
both the `name` expression and the `template` block.

| Expression | Description |
|-----------|-------------|
| `each.key` | Index for lists, map key for maps, value for sets |
| `each.value` | Value at the current position |

The meaning of `each.key` and `each.value` depends on the collection type passed to `for_each`:

| Collection type | `each.key` | `each.value` |
|----------------|-----------|-------------|
| List | Index (0, 1, 2, ...) | Value at that index |
| Map | Map key | Map value |
| Set | The value itself | The value itself |

## The `template` Block

The `template` block has exactly the same semantics as a `resource` block. Anything you can do in a
`resource` block is allowed inside `template`: locals, body, composite status, composite connection, ready.

## Using range() for Counted Resources

To create a fixed number of resources (like Terraform's `count`), use `range()`:

```hcl
resources my-buckets {
  locals {
    numBuckets = try(req.composite.spec.parameters.numBuckets, 1)
  }

  for_each = range(numBuckets)

  template {
    locals {
      name = "${req.composite.metadata.name}-bucket-${each.value}"
    }
    body = {
      apiVersion = "s3.aws.upbound.io/v1beta1"
      kind       = "Bucket"
      metadata   = { name = name }
      spec       = { forProvider = { region = req.composite.spec.parameters.region } }
    }
  }
}
```
