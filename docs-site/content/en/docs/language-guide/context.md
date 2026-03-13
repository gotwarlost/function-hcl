---
title: "Context"
linkTitle: "Context"
weight: 12
description: >
  Sharing data with downstream pipeline steps.
---

The `context` block writes values into the Crossplane pipeline context, making them available
to downstream functions in the pipeline.

## Syntax

```hcl
context {
  key   = <string>
  value = <any>
}
```

## Example

```hcl
context {
  key = "example.com/foo-bar-baz"
  value = {
    foo = { bar = "baz" }
    bar = 10
    baz = "quux"
  }
}
```

## Merging Multiple Context Blocks

You can write to the same context key from multiple blocks. Object values are merged using
the same rules as `composite status`:

```hcl
context {
  key = "example.com/foo-bar-baz"
  value = {
    foo = { bar = "baz" }
  }
}

context {
  key = "example.com/foo-bar-baz"
  value = {
    foo = { baz = "bar" }
  }
}
# Result: { foo = { bar = "baz", baz = "bar" } }
```

Non-object values at the same path with different values are an error (same as status).

## Reading Context

Values written to the context by upstream pipeline steps can be read via `req.context`:

```hcl
locals {
  upstreamData = req.context["some-function/some-key"]
}
```

## Automatic Deferral

Like other blocks, if any expression in a `context` block is incomplete, the block is
deferred to a later reconcile cycle.
