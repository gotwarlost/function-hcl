---
title: "Conditions"
linkTitle: "Conditions"
weight: 8
description: >
  Conditionally creating resources, groups, and collections.
---

Use a `condition` attribute to create resources only when specific conditions are met.
This is the function-hcl equivalent of Terraform's `count = var.create ? 1 : 0`.

## On a Resource

```hcl
resource s3_acl {
  condition = try(req.composite.spec.parameters.createAcls, true)

  body = {
    # ...
  }
}
```

The `condition` expression must evaluate to a boolean value. If the value is `false`, the resource
is skipped entirely.

## On a Resource Collection

When applied to a `resources` block, the condition controls the **entire collection**. To filter
individual elements, filter the collection that `for_each` iterates over instead.

```hcl
resources s3_acls {
  condition = req.composite.spec.parameters.createAcls

  for_each = req.composite.spec.parameters.suffixes
  template {
    body = { ... }
  }
}
```

## On a Group

A condition on a `group` skips all resources in the group:

```hcl
group {
  condition = req.composite.spec.parameters.createMoreBuckets

  resource additional-bucket-one { ... }
  resource additional-bucket-two { ... }
}
```

## Incomplete Conditions

Unlike resource bodies, where incomplete values cause deferral, an **incomplete condition
value is treated as `false`**. Use `try` and `can` if the condition depends on values that
might not exist yet:

```hcl
resource my-resource {
  # If createFoo doesn't exist in the parameters, default to true
  condition = try(req.composite.spec.parameters.createFoo, true)

  body = { ... }
}
```

{{% alert title="Important" color="warning" %}}
If a condition value evaluates to something that is neither `true` nor `false` (e.g. a string),
it is treated as an error.
{{% /alert %}}
