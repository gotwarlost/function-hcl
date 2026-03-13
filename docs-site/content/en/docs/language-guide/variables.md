---
title: "The req Variable"
linkTitle: "The req Variable"
weight: 3
description: >
  Accessing the Crossplane request state with the req variable.
---

function-hcl provides a built-in `req` variable that gives you access to the Crossplane request
state. It is available everywhere in the script -- top-level locals, resource blocks, groups,
and nested blocks.

## Fields

| Expression                 | Type                                  | Description                                                      |
|----------------------------|---------------------------------------|------------------------------------------------------------------|
| `req.composite`            | k8s object                            | The observed composite resource (XR)                             |
| `req.composite_connection` | map(string, bytes)                    | Connection details of the composite resource                     |
| `req.resource`             | map(string, k8s object)               | Observed composed resources, keyed by crossplane resource name   |
| `req.connection`           | map(string, map(string, bytes))       | Connection details of observed resources, keyed by resource name |
| `req.resources`            | map(string, list(k8s object))         | Observed resource collections, keyed by collection base name     |
| `req.connections`          | map(string, list(map(string, bytes))) | Connection details of resource collections                       |
| `req.context`              | map(string, any)                      | Pipeline context values from upstream functions                  |
| `req.extra_resources`      | map(string, list(k8s object))         | Extra resources fetched via `requirement` blocks                 |

## Example

```hcl
locals {
  comp     = req.composite
  compName = comp.metadata.name
  params   = comp.spec.parameters
  region   = params.region

  # Access an observed resource by its crossplane name
  observedBucket = req.resource.my-bucket

  # Access pipeline context
  upstreamValue = req.context["upstream-function/some-key"]
}
```

## Accessing Observed Resources

Use `req.resource.<name>` to access any observed composed resource by its crossplane name.
This is useful when one resource needs to reference the status of another:

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

If the referenced resource hasn't been created yet, the expression is incomplete, and the
block that uses it will be [deferred](../../concepts/deferred-rendering/).

{{% alert title="Note" color="info" %}}
Inside `resource` and `resources` blocks, additional variables (`self` and `each`) become
available. These are introduced in [Resource Blocks](../resource-blocks/) and
[Resource Collections](../resource-collections/) respectively.
{{% /alert %}}
