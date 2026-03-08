---
title: "Groups"
linkTitle: "Groups"
weight: 7
description: >
  Grouping related resources with shared locals.
---

The `group` block allows you to group related resources together and share local variables among them
without polluting the top-level namespace.

## Syntax

```hcl
group {
  condition = <bool>    # optional: skip the entire group
  locals { ... }        # optional: shared locals for resources in this group

  resource <name> { }
  resource <name> { }
  resources <name> { }
}
```

## Example

```hcl
group {
  locals {
    vpcParams = req.composite.spec.parameters.vpc
    region    = vpcParams.region
  }

  resource my-vpc {
    body = {
      apiVersion = "ec2.aws.upbound.io/v1beta1"
      kind       = "VPC"
      metadata   = { name = "${req.composite.metadata.name}-vpc" }
      spec       = { forProvider = { region = region, cidrBlock = vpcParams.cidr } }
    }
  }

  resource my-subnet {
    body = {
      apiVersion = "ec2.aws.upbound.io/v1beta1"
      kind       = "Subnet"
      metadata   = { name = "${req.composite.metadata.name}-subnet" }
      spec = {
        forProvider = {
          region    = region
          vpcId     = req.resource.my-vpc.status.atProvider.id
          cidrBlock = vpcParams.subnetCidr
        }
      }
    }
  }
}

# vpcParams and region are NOT available here
```

## Scoping

- Locals defined in a `group` block are available to all resources within the group.
- They are **not** available outside the group.
- Resources in the group still have access to top-level locals.
- Groups can contain `resource`, `resources`, and nested `group` blocks.

## Conditional Groups

A `group` can have a `condition` attribute to skip the entire group:

```hcl
group {
  condition = req.composite.spec.parameters.createVpc

  resource my-vpc { ... }
  resource my-subnet { ... }
}
```

See [Conditions](../conditions/) for details.
