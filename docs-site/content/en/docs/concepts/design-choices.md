---
title: "Design Choices"
linkTitle: "Design Choices"
weight: 3
description: >
  The reasoning behind function-hcl's DSL design decisions.
---

This page documents the design choices made when creating the function-hcl DSL. Understanding these
helps explain why things work the way they do.

## How close to Terraform?

We started with "as close as possible" but realized there were problems with that approach. Things
looked Terraform-like but had subtly different semantics. The compromise:

- **Make it look different enough** to signal that this is not Terraform.
- **Maintain a rough mapping** between Terraform and function-hcl constructs, so that a converter
  could be written with some heuristics in the future.
- **Support all Terraform functions** that can reasonably be supported (pure, non-file-based functions from v1.5.7).
- One remaining similarity is the `resource` keyword -- we couldn't come up with a better name, even though
  function-hcl uses one label (crossplane name) versus Terraform's two (type + name).

## `resource` vs `resources`

function-hcl uses `resource` (singular) for individual resources and `resources` (plural) for collections.
Terraform uses a single `resource` block with `for_each` or `count` for both cases.

We chose separate block types because:
- It makes the intent explicit at a glance.
- The observed variables follow the same naming pattern: `req.resource` (singular) vs `req.resources` (plural).
- Collection semantics (iterator variables, name expressions, templates) are different enough to warrant
  their own block structure.

## Local variables without namespace

In Terraform, locals are accessed as `local.foo`. In function-hcl, they're accessed as just `foo`.

The reasoning:
- Locals are the most frequently accessed variables, so they should be the shortest to type.
- In Terraform, the unqualified namespace is used for resources. In function-hcl, resources are accessed
  via `req.resource.<name>`, so the unqualified namespace is free for locals.
- Locals can always be precomputed before user code accesses them, making this safe.

## Scoped locals

function-hcl allows `locals` blocks at multiple levels (top-level, inside `resource`, inside `group`,
inside `resources` templates). This is a departure from Terraform's single flat namespace per module.

The motivation:
- Truly local temporary variables inside a resource block without polluting the global namespace.
- Can be specified in any order (like Terraform) -- this is necessary to support file-scoped locals
  across multiple files.
- Inner scopes cannot shadow outer scope names to prevent confusion.

## `condition` instead of `count`

Terraform uses `count` to conditionally create resources (e.g. `count = var.create ? 1 : 0`).
function-hcl uses `condition`, which is a boolean value.

The reasoning:
- A boolean is more expressive for conditional creation (which is what `count` is usually used for).
- If you actually need `count` semantics (create N copies), use a `resources` block with `for_each = range(n)`.

## `each` as a reserved word

Inside `resources` blocks, the iterator is accessed as `each.key` and `each.value` (similar to Terraform).
We could have used `self.each` instead, but chose `each` as a top-level reserved word for ergonomics.

## Status unification

The ability to merge `composite status` values from multiple `resource` blocks -- with conflict
detection -- was inspired by the [CUE language](https://cuelang.org/)'s approach to value unification.

## txtar format

Using [txtar](https://pkg.go.dev/golang.org/x/tools/txtar) as the packaging format was also inspired
by CUE, which uses txtar extensively for testing. It allows multiple files to be embedded in a single
YAML string while preserving file names and line numbers in error messages.
