---
title: "Automatic Dependency Resolution"
linkTitle: "Dependency Resolution"
weight: 2
description: >
  How function-hcl automatically defers resources with unresolved dependencies.
---

Automatic dependency resolution is the headline feature of function-hcl. It eliminates the boilerplate
conditional logic that is common in Crossplane compositions.

## The Problem

In a typical Crossplane composition, if resource B depends on a status field from resource A, you need to handle
the case where resource A hasn't been created yet. This often requires `try()`, `can()`, or manual conditional
logic scattered throughout your composition.

## How function-hcl Solves It

function-hcl uses a simple rule: if **any expression** in a block evaluates to an incomplete value
(null, missing attribute, missing index), the **entire block is dropped** from the output for that
reconcile cycle.

This applies to:
- `resource` blocks (the resource is not rendered)
- `composite status` blocks (the status update is skipped)
- `composite connection` blocks (the connection detail is skipped)
- `context` blocks (the context value is skipped)
- `requirement` blocks (the requirement is skipped)

On the next reconcile, once the missing value becomes available, the block is evaluated normally.

## Example

```hcl
resource my-vpc {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "VPC"
    metadata   = { name = "${name}-vpc" }
    spec       = { forProvider = { region = params.region, cidrBlock = params.vpcCidr } }
  }

  composite status {
    body = {
      vpcId = self.resource.status.atProvider.id
    }
  }
}

resource my-subnet {
  body = {
    apiVersion = "ec2.aws.upbound.io/v1beta1"
    kind       = "Subnet"
    metadata   = { name = "${name}-subnet" }
    spec = {
      forProvider = {
        region    = params.region
        vpcId     = req.composite.status.vpcId # deferred until VPC status is written
        cidrBlock = params.subnetCidr
      }
    }
  }
}
```

| Reconcile | `my-vpc` | `composite status` | `my-subnet` |
|-----------|----------|-------------------|-------------|
| 1st | Rendered | Deferred (no observed VPC yet) | Deferred (no `vpcId` in XR status) |
| 2nd | Rendered | Rendered (VPC now has status) | Deferred (XR status updated, but not yet visible) |
| 3rd | Rendered | Rendered | Rendered (vpcId now available) |

## Safety Guarantees

function-hcl will **never silently drop a resource that already exists** in the observed state. If a resource
was previously rendered and created by Crossplane, but now evaluates to an incomplete value, function-hcl
returns a **fatal error** instead of dropping it.

This prevents accidental deletion due to:
- Typos introduced in code changes
- Temporary unavailability of upstream status fields
- Incorrect refactoring

## Observability

When blocks are deferred, function-hcl provides full visibility:

### Status Conditions

The function maintains two status conditions on the composite resource:

- **`FullyResolved`** -- `True` only when no blocks were deferred. When `False`, the `message` field
  lists the deferred items.
- **`HclDiagnostics`** -- contains HCL diagnostic information including warnings about incomplete values.

### Events

Warning events are emitted on the composite resource for every deferred block. Each event includes:
- The source file and line number of the deferred block
- The specific expressions that were incomplete
- The reason each expression couldn't be evaluated

You can inspect these with `kubectl describe` on your composite resource.

### Example Status

```yaml
conditions:
  - type: FullyResolved
    status: "False"
    reason: IncompleteItemsPresent
    message: "composite-status incomplete"
  - type: HclDiagnostics
    status: "False"
    reason: Eval
    message: "hcl.Diagnostics contains 1 warnings; main.hcl:20,32-39: Attempt to get attribute from null value"
```

When everything resolves:

```yaml
conditions:
  - type: FullyResolved
    status: "True"
    reason: AllItemsProcessed
    message: "all items complete"
  - type: HclDiagnostics
    status: "True"
    reason: Eval
    message: "hcl.Diagnostics contains no warnings"
```
