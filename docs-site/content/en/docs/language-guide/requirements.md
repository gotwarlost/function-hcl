---
title: "Requirements (Extra Resources)"
linkTitle: "Requirements"
weight: 13
description: >
  Requesting extra resources from Crossplane.
---

The `requirement` block lets you request extra resources from Crossplane. These are typically
used to read configuration data (like `EnvironmentConfig` objects) that aren't part of the
composition's resource tree.

## Syntax

```hcl
requirement <name> {
  condition = <bool>    # optional
  locals { ... }        # optional

  select {
    apiVersion  = <string>
    kind        = <string>
    matchName   = <string>       # match by name
    # OR
    matchLabels = <map(string)>  # match by labels
  }
}
```

You must specify either `matchName` or `matchLabels`, but not both.

## Match by Name

```hcl
requirement my-config {
  select {
    apiVersion = "apiextensions.crossplane.io/v1beta1"
    kind       = "EnvironmentConfig"
    matchName  = "foo-bar"
  }
}
```

## Match by Labels

```hcl
requirement my-config {
  select {
    apiVersion  = "apiextensions.crossplane.io/v1beta1"
    kind        = "EnvironmentConfig"
    matchLabels = { foo = "bar" }
  }
}
```

## Accessing Extra Resources

The name given to the requirement is the key in `req.extra_resources`. The value is always an
**array** of matching objects, even when matching by name:

```hcl
locals {
  # Access the first matching object
  myConfig     = req.extra_resources.my-config[0]
  myLabelValue = myConfig.metadata.labels["my-label"]
  myDataValue  = myConfig.data.labels
}
```

## Conditional Requirements

Use `condition` to skip the requirement if certain conditions aren't met:

```hcl
requirement labels-config {
  condition = req.composite.metadata.labels["special"] == "true"

  locals {
    ecName = "foo-bar"
  }

  select {
    apiVersion = "apiextensions.crossplane.io/v1beta1"
    kind       = "EnvironmentConfig"
    matchName  = ecName
  }
}
```

The requirement is skipped if the condition evaluates to `false`. The usual rules for
[conditions](../conditions/) apply.

## Locals in Requirements

Local variables in a `requirement` block can be used as temporary variables for complex
calculations that feed into the `select` block.

## When Extra Resources Are Absent

If the requested resource doesn't exist yet, `req.extra_resources.<name>` may be null or empty.
Any resource block that references missing extra resources will be
[automatically deferred](../../concepts/dependency-resolution/).

## Error Conditions

The function returns an error if:
- A requirement block specifies both `matchName` and `matchLabels`
- A requirement block specifies neither `matchName` nor `matchLabels`
- There is a data type mismatch in the `select` attributes
