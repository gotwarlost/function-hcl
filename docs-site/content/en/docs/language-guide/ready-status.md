---
title: "Resource Ready Status"
linkTitle: "Ready Status"
weight: 11
description: >
  Controlling resource readiness.
---

The `ready` block lets you assign a ready state for a resource.
Usually you would use [function-auto-ready](https://github.com/crossplane-contrib/function-auto-ready)
to automatically set the resource status.

This allows you to set the ready status explicitly.

## Syntax

```hcl
resource foo {
  body = { /* ... */ }

  ready {
    value = "READY_TRUE"
  }
}
```

## Valid Values

The `value` attribute must evaluate to a string and be one of:

| Value                 | Meaning                  |
|-----------------------|--------------------------|
| `"READY_UNSPECIFIED"` | Ready state is not known |
| `"READY_TRUE"`        | Resource is ready        |
| `"READY_FALSE"`       | Resource is not ready    |

## Example: Custom Readiness Check

```hcl
resource my-database {
  body = { /* ... */ }

  ready {
    value = self.resource.status.atProvider.state == "available" ? "READY_TRUE" : "READY_FALSE"
  }
}
```

If `self.resource` is incomplete (resource not yet created), the `ready` block follows
the same deferral rules as other blocks.
