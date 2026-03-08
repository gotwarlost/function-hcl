---
title: "Best Practices"
linkTitle: "Best Practices"
weight: 1
description: >
  Conventions and patterns for writing clean, maintainable function-hcl compositions.
---

## Always use fn-hcl-tools package

Run `fn-hcl-tools package` to produce your txtar script rather than hand-crafting it. This
validates your HCL before it reaches the cluster, catching typos in variable names, bad block
structure, and syntax errors at authoring time.

```bash
fn-hcl-tools package ./my-composition/
```

## Use locals to name things

Pull commonly-used expressions into top-level locals. This makes the code easier to read and
avoids long repeated expressions like `req.composite.spec.parameters` deep inside resource bodies.

```hcl
locals {
  comp     = req.composite
  compName = comp.metadata.name
  params   = comp.spec.parameters
}
```

## Scope locals tightly

Use resource-scoped locals for values that only one resource needs. This keeps the top-level
namespace clean and makes it obvious which resource uses which values.

```hcl
resource my-bucket {
  locals {
    bucketName = "${compName}-bucket"
  }
  body = {
    # ...
    metadata = { name = bucketName }
  }
}
```

## Use groups for related resources

When several resources share configuration, wrap them in a `group` with shared locals instead
of repeating values or polluting the top-level namespace.

```hcl
group {
  locals {
    region = params.vpc.region
    cidr   = params.vpc.cidr
  }

  resource my-vpc { ... }
  resource my-subnet { ... }
}
```

## Put composite status inside resource blocks

Write `composite status` inside the `resource` block where the data originates, not at the
top level. This keeps the status update next to the resource it depends on and gives you
access to `self.resource` for clean expressions.

```hcl
# Prefer this:
resource my-bucket {
  body = { ... }
  composite status {
    body = { bucketArn = self.resource.status.atProvider.arn }
  }
}

# Over this:
resource my-bucket {
  body = { ... }
}
composite status {
  body = { bucketArn = req.resource.my-bucket.status.atProvider.arn }
}
```

## Split large compositions into multiple files

Use separate `.hcl` files for logical groupings of resources. `fn-hcl-tools package` will
bundle them into a single txtar script. This keeps individual files focused and easier to review.

```
my-composition/
  locals.hcl        # shared locals
  networking.hcl    # VPC, subnets, security groups
  database.hcl      # RDS instances
  storage.hcl       # S3 buckets
  functions.hcl     # user-defined functions
```

## Use user functions for repeated patterns

If you're producing the same resource shape (e.g. wrapping manifests in a Kubernetes provider
Object), extract it into a `function` block rather than duplicating the body structure.

## Prefer explicit names for resource collections

Override the default `name` expression in `resources` blocks when the generated names should
be meaningful:

```hcl
resources regional-buckets {
  for_each = params.regions
  name     = "${self.basename}-${each.value}"  # "regional-buckets-us-east-1" instead of "regional-buckets-0"

  template {
    body = { ... }
  }
}
```

## Test locally with crossplane beta render

Use `crossplane beta render` to test your compositions without a cluster:

```bash
crossplane beta render xr.yaml composition.yaml functions.yaml
```
