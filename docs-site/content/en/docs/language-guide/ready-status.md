---
title: "Resource Ready Status"
linkTitle: "Ready Status"
weight: 11
description: >
  Controlling resource readiness.
---

The `ready` block lets you override Crossplane's default readiness check for a resource.

## Syntax

```hcl
resource foo {
  body = { ... }

  ready {
    value = "READY_TRUE"
  }
}
```

## Valid Values

The `value` attribute must evaluate to a string and be one of:

| Value | Meaning |
|-------|---------|
| `"READY_UNSPECIFIED"` | Let Crossplane determine readiness (default behavior) |
| `"READY_TRUE"` | Mark the resource as ready |
| `"READY_FALSE"` | Mark the resource as not ready |

## Example: Custom Readiness Check

```hcl
resource my-database {
  body = { ... }

  ready {
    value = self.resource.status.atProvider.state == "available" ? "READY_TRUE" : "READY_FALSE"
  }
}
```

If `self.resource` is incomplete (resource not yet created), the `ready` block follows
the same deferral rules as other blocks.
