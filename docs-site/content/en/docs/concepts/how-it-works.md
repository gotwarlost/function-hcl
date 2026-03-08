---
title: "How It Works"
linkTitle: "How It Works"
weight: 1
description: >
  How function-hcl fits into the Crossplane composition pipeline.
---

## The Crossplane Function Pipeline

Crossplane composition functions run as steps in a pipeline. Each step receives the current desired state,
may modify it, and passes it on to the next step. function-hcl is one such step.

```
Crossplane
    |
    v
+---------------------+
|   function-hcl      |  <-- reads req.composite, evaluates HCL, writes desired resources
+---------------------+
    |
    v
 (next function or final reconcile)
```

## What function-hcl Does

When Crossplane calls function-hcl, it:

1. **Unpacks the input** -- reads the HCL script (or txtar bundle) from the Composition pipeline step's `input` field.
2. **Evaluates the HCL** -- runs the script with access to the current composite resource, observed resources, and context.
3. **Builds desired resources** -- each `resource` block that evaluates successfully contributes a desired composed resource to the output.
4. **Handles incomplete resources** -- resource blocks that reference values not yet available (e.g. a status field from a resource that hasn't been created yet) are _silently deferred_ rather than causing an error.
5. **Writes back status / connection details / context** -- `composite status`, `composite connection`, and `context` blocks write values back to the composite resource or pipeline context.

## The Reconcile Loop

Crossplane runs composition functions repeatedly as the state of the world changes. This is important for
understanding function-hcl's behavior:

1. **First reconcile**: Your HCL runs. Resources that can be fully evaluated are rendered. Resources with
   incomplete dependencies are deferred (silently dropped from the desired output).
2. **Subsequent reconciles**: As Crossplane creates resources and providers populate their status fields,
   previously-deferred resources can now resolve their dependencies and are rendered.
3. **Steady state**: All resources are rendered, all status values are populated. The `FullyResolved`
   condition is `True`.

This loop is what makes [automatic dependency resolution](../dependency-resolution/) possible.
