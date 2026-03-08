---
title: "Composite Connection Details"
linkTitle: "Connection Details"
weight: 10
description: >
  Writing connection details to the composite resource.
---

The `composite connection` block writes connection details to the composite resource.
It works the same way as `composite status` in terms of scoping and merging.

## Syntax

```hcl
composite connection {
  body = {
    <key> = <base64-encoded-string>
  }
}
```

All values **must** be strings that are base64 encoded. The function returns an error otherwise.

## At the Top Level

```hcl
composite connection {
  body = {
    url = base64encode(req.resource.foobar.status.atProvider.url)
  }
}
```

## Inside a Resource Block

```hcl
resource my-database {
  body = { ... }

  composite connection {
    body = {
      connectionString = base64encode(self.resource.status.atProvider.connectionString)
      password         = self.connection["password"]  # already base64 from the provider
    }
  }
}
```

## Merging and Conflict Detection

Multiple `composite connection` blocks can set different keys. Two blocks cannot set the
same key to different values -- this is an error.

## Automatic Deferral

Like other blocks, if any expression is incomplete, the entire `composite connection` block
is deferred to a later reconcile cycle.
