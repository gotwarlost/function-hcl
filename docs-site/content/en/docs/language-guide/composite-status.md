---
title: "Composite Status"
linkTitle: "Composite Status"
weight: 9
description: >
  Writing values back to the composite resource status.
---

The `composite status` block writes values to the composite resource's (XR) status. It can be
specified any number of times, and each block can update specific fields.

## At the Top Level

```hcl
composite status {
  body = {
    foobarId = req.resource.foobar.status.atProvider.id
  }
}
```

## Inside a Resource Block

This is the more common pattern, since `self.resource` gives you direct access to the observed
state of the current resource:

```hcl
resource foobar {
  body = { ... }

  composite status {
    body = {
      foobarId = self.resource.status.atProvider.id
    }
  }
}
```

## Merging Multiple Status Blocks

Multiple `composite status` blocks can update different fields. Objects are merged:

```hcl
resource vpc {
  body = { ... }
  composite status {
    body = {
      network = { vpcId: self.resource.status.atProvider.id }
    }
  }
}

resource subnet {
  body = { ... }
  composite status {
    body = {
      network = { subnetId: self.resource.status.atProvider.id }
    }
  }
}
# Result: status.network = { vpcId: "...", subnetId: "..." }
```

## Conflict Detection

If two `composite status` blocks produce the **same non-object attribute** with **different values**,
the function returns an error:

```hcl
# This is an ERROR:
composite status {
  body = { clash = 10 }
}
composite status {
  body = { clash = 20 } # different value for same key
}
```

Object values at the same path are merged recursively. Only leaf value conflicts are errors.

## Automatic Deferral

If any expression in a `composite status` block is incomplete (e.g. `self.resource` is null because
the resource hasn't been created yet), the entire status block is silently deferred. The status
will be written once the values become available.
