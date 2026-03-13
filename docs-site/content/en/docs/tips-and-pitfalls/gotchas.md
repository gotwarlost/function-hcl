---
title: "Gotchas"
linkTitle: "Gotchas"
weight: 2
description: >
  Common surprises and how to avoid them.
---

## Dashes in identifiers mean you can't subtract without spaces

HCL allows dashes in identifiers. This means `a-b` is a single identifier, not `a` minus `b`.
Use spaces for arithmetic: `a - b`.

```hcl
locals {
  my-var = 10     # this is a variable named "my-var"
  result = my-var # refers to the variable, not "my" minus "var"
  diff   = a - b  # subtraction -- spaces required
}
```

## body is an attribute, not a block

`body` must use `=` for assignment. Omitting `=` turns it into a block, which is a different
HCL construct and will produce a schema error.

```hcl
# Correct
resource my-bucket {
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
  }
}

# Wrong -- this is block syntax, not attribute assignment
resource my-bucket {
  body {
    apiVersion = "s3.aws.upbound.io/v1beta1"
  }
}
```

## Locals cannot shadow parent scope names

Unlike many programming languages, a local variable in an inner scope cannot reuse a name
from an outer scope. This is an error, not a shadowed variable:

```hcl
locals {
  region = "us-east-1"
}

resource my-bucket {
  locals {
    region = "eu-west-1" # ERROR: shadows top-level 'region'
  }
}
```

Rename the inner variable instead:

```hcl
resource my-bucket {
  locals {
    bucket-region = "eu-west-1" # OK
  }
}
```

## Incomplete conditions are treated as false, not deferred

Unlike resource bodies (which are deferred when incomplete), a `condition` that evaluates to
an incomplete value is treated as `false` -- the resource is silently skipped. Use `try` to
provide a default:

```hcl
resource my-bucket {
  # If params.createBucket doesn't exist yet, this silently skips the resource
  condition = req.composite.spec.parameters.createBucket

  # Better: default to true if the field doesn't exist
  condition = try(req.composite.spec.parameters.createBucket, true)
}
```

## An existing resource becoming incomplete is a fatal error

If a resource was previously rendered (it exists in the observed state) but now evaluates to
an incomplete value, function-hcl returns a fatal error rather than silently dropping it. This
is a safety mechanism, but it means:

- A typo in a code change can cause the composition to error out
- Check `kubectl describe` on the XR for the `HclDiagnostics` condition to see which expression failed

## Extra resources are always arrays

Even when using `matchName` (which matches at most one object), `req.extra_resources.<name>`
is always an array. You need `[0]` to access the first element:

```hcl
# Wrong
locals {
  config = req.extra_resources.my-config.data
}

# Correct
locals {
  config = req.extra_resources.my-config[0].data
}
```

## resource vs resources -- singular matters

`resource` (singular) creates one resource. `resources` (plural) creates a collection with
`for_each`. Using the wrong one is a schema error. The observed state variables follow the
same pattern: `req.resource` (singular) vs `req.resources` (plural).

## User functions cannot access req, self, or each

Functions are pure -- they only have access to their declared arguments and their own locals.
If you need request data inside a function, pass it as an argument:

```hcl
# Wrong -- this will fail
function makeBucket {
  arg region {}
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata   = { name = req.composite.metadata.name } # ERROR: req not accessible
  }
}

# Correct -- pass what you need as arguments
function makeBucket {
  arg name {}
  arg region {}
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata   = { name = name }
    spec       = { forProvider = { region = region } }
  }
}
```

## invoke requires a static string for the function name

The first argument to `invoke` must be a string literal. You cannot use a variable:

```hcl
# Wrong
locals {
  fn     = "myFunction"
  result = invoke(fn, { a = 1 }) # ERROR
}

# Correct
locals {
  result = invoke("myFunction", { a = 1 })
}
```

## Conflicting status/connection values from different blocks are errors

If two `composite status` blocks set the same leaf attribute to different values, the function
errors out. Object values are merged recursively, but leaf conflicts are fatal:

```hcl
# This is fine -- objects merge
composite status { body = { network = { vpcId = "vpc-123" } } }
composite status { body = { network = { subnetId = "sub-456" } } }
# Result: network = { vpcId = "vpc-123", subnetId = "sub-456" }

# This is an ERROR -- scalar conflict
composite status { body = { region = "us-east-1" } }
composite status { body = { region = "eu-west-1" } }
```

## Connection details must be base64 encoded

All values in `composite connection` blocks must be base64-encoded strings. Forgetting to
encode them is an error:

```hcl
# Wrong
composite connection {
  body = { url = self.resource.status.atProvider.url }
}

# Correct
composite connection {
  body = { url = base64encode(self.resource.status.atProvider.url) }
}
```
