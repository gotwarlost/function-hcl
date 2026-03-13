---
title: "Local Variables"
linkTitle: "Local Variables"
weight: 4
description: >
  Defining and scoping local variables.
---

Local variables in function-hcl work like Terraform locals, with some important differences.

## Basics

Define locals in a `locals` block. Access them directly by name -- no `local.` prefix needed.

```hcl
locals {
  baseName     = req.composite.metadata.name
  computedName = "${baseName}-bucket"
}
```

## Ordering Doesn't Matter

All `locals` blocks in a given scope are processed together and variable ordering doesn't matter.
Dependencies are resolved automatically. This is the same behavior as Terraform.

```hcl
# These two blocks are treated identically:
locals {
  computedName = "${baseName}-bucket"
}

locals {
  baseName = req.composite.metadata.name
}
```

Circular references (e.g. `locals { a = b; b = a }`) are an error.

## Scoped Locals

Unlike Terraform, function-hcl allows `locals` blocks at multiple levels:

- **Top-level** -- available everywhere in the script
- **Inside `resource` blocks** -- available only within that resource (has access to `self`)
- **Inside `resources` template blocks** -- available within the template (has access to `self` and `each`)
- **Inside `group` blocks** -- available to all resources in the group
- **Inside `requirement` blocks** -- available within the requirement
- **Inside `function` blocks** -- available within the function body

Inner scopes can access variables from outer scopes:

```hcl
locals {
  compName = req.composite.metadata.name # top-level
}

resource my-bucket {
  locals {
    resourceName = "${compName}-bucket" # can use top-level locals
  }

  body = {
    # ...
    metadata = { name = resourceName }
  }
}
```

## No Shadowing

A local variable **cannot shadow** a name from a parent scope. This is an error:

```hcl
locals {
  x = "top-level"
}

resource my-resource {
  locals {
    x = "resource-level"  # ERROR: shadows top-level local 'x'
  }
  body = { ... }
}
```

It **is** valid to use the same local variable name in different resource blocks, as long as there is
no top-level local with that name:

```hcl
resource bucket-a {
  locals {
    region = "us-east-1"  # OK
  }
  body = { ... }
}

resource bucket-b {
  locals {
    region = "eu-west-1"  # OK -- different scope from bucket-a
  }
  body = { ... }
}
```

## What Locals Can Access

Locals can access:
- The `req` variable
- Other local variables (in the same or parent scope)
- `self` and `each` variables (when inside a resource or resources block)
