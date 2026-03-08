---
title: "HCL Basics"
linkTitle: "HCL Basics"
weight: 2
description: >
  Understanding HCL's fundamental constructs: blocks, attributes, expressions, and identifiers.
---

Before diving into function-hcl specifics, it helps to understand the fundamental building blocks of
HCL syntax. This page covers the essentials; for the full language specification see the
[HCL Native Syntax Specification](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md).

People who have worked with Terraform before should feel free to skip this section.

## Blocks

A block is a container that has a **type**, zero or more **labels**, and a **body** enclosed
in braces. The body contains attributes and/or nested blocks.

```
<type> [<label1> <label2> ...] {
  <body: attributes and nested blocks>
}
```

Examples:

```hcl
# Block with no labels
locals {
  name = "foo"
}

# Block with one label
resource my-bucket {
  body = { /* ... */ }
}

# Block with two labels
resource my-bucket {
  # 'composite' is the block type, 'status' is a label
  composite status {
    body = { /* ... */ }
  }
}
```

Unlike attributes, some block types **can** appear multiple times in the same scope. For example,
you can have multiple `locals` blocks, multiple `resource` blocks, or multiple `composite status`
blocks.

## Attributes

An attribute assigns a value to a name. The value can be a literal, an expression, or a complex
object.

```hcl
# Simple attribute assignments
name    = "my-bucket"
count   = 3
enabled = true

# Object value
tags = {
  env  = "production"
  team = "platform"
}

# Expression value
full_name = "${prefix}-${name}"
```

Attributes use `=` for assignment:

```hcl
body = {
  apiVersion = "s3.aws.upbound.io/v1beta1"
  kind       = "Bucket"
  metadata = {
    name = "my-bucket"
  }
}
```

{{% alert title="Note" color="info" %}}
HCL also allows `:` as a delimiter inside object literals (e.g. `kind: "Bucket"`). Both `=` and
`:` are valid, but this documentation uses `=` consistently for clarity.
{{% /alert %}}

An attribute can only be set **once** in a given scope. Setting the same attribute name twice
in the same block is an error.

## Expressions and Functions

An expression is anything that produces a value. Expressions appear on the right-hand side of
attribute assignments.

### Literals

```hcl
"hello"       # string
42            # number
true          # bool
["a", "b"]   # list
{ x = 1 }    # object
```

### String Interpolation

Strings enclosed in double quotes can contain interpolation sequences using `${ }`:

```hcl
locals {
  greeting = "Hello, ${name}!"
  arn      = "arn:aws:s3:::${bucketName}"
  combined = "${first}-${last}"
}
```

Any expression is valid inside `${ }`, including function calls:

```hcl
locals {
  upper-name = "PREFIX-${upper(name)}"
  safe-name  = "${try(params.name, "default")}"
}
```

When a string contains only a single interpolation with no surrounding text, you can drop the
quotes entirely. These two are equivalent:

```hcl
locals {
  # These produce the same result
  a = "${req.composite.metadata.name}"
  b = req.composite.metadata.name
}
```

Prefer the bare form when no string concatenation is needed.

### Operators

| Category | Operators |
|----------|-----------|
| Arithmetic | `+`, `-`, `*`, `/`, `%` |
| Comparison | `==`, `!=`, `<`, `>`, `<=`, `>=` |
| Logical | `&&`, `||`, `!` |

### Conditional Expressions

```hcl
locals {
  env = production ? "prod" : "dev"
}
```

### For Expressions

Transform lists and maps inline:

```hcl
locals {
  upper-names = [for s in names : upper(s)]
  tagged      = { for k, v in items : k => merge(v, { env = "prod" }) }
  filtered    = [for s in names : s if s != ""]
}
```

### Splat Expressions

Shorthand for extracting an attribute from every element in a list:

```hcl
locals {
  # These are equivalent
  ids-long  = [for r in resources : r.id]
  ids-short = resources[*].id
}
```

### Index and Attribute Access

```hcl
locals {
  first  = list[0]
  value  = map["key"]
  nested = object.child.field
}
```

### Functions

HCL includes a rich standard library. function-hcl supports all
[Terraform functions](https://developer.hashicorp.com/terraform/language/functions) as of v1.5.7,
except file I/O functions (`file()`, `templatefile()`, etc.) and impure functions (`uuid()`,
`timestamp()`, etc.). See [Built-in Functions](../reference/built-in-functions/) for the
full list.

Some commonly used ones:

```hcl
locals {
  merged  = merge(defaults, overrides)
  safe    = try(obj.field, "fallback")
  ok      = can(obj.field)
  items   = join(",", list)
  encoded = base64encode(secret)
  count   = length(names)
}
```

## Identifiers

An identifier is a name used to refer to things -- local variables, resource names, block labels,
attribute keys, and function names.

HCL identifiers follow these rules:
- Must start with a letter or underscore
- Can contain letters, digits, underscores, and **dashes**
- Are case-sensitive

Unlike most programming languages where `my-bucket` would be parsed as `my` minus `bucket`, HCL treats dashes as valid identifier
characters. This is why you'll see resource names like:

```hcl
resource my-s3-bucket {
  # ...
}

locals {
  comp-name = req.composite.metadata.name
}
```

Both `my-s3-bucket` and `comp-name` are single identifiers, not subtraction expressions.

{{% alert title="Tip" color="info" %}}
Because dashes are valid in identifiers, you cannot use `-` for subtraction without whitespace
or parentheses to disambiguate. Write `a - b` (with spaces) or `(a)-(b)`, not `a-b`.
{{% /alert %}}

This is especially relevant in function-hcl because Kubernetes resource names use dashes heavily,
and crossplane resource names follow the same convention. Being able to write
`resource my-vpc-subnet { ... }` rather than `resource my_vpc_subnet { ... }` keeps your HCL
aligned with the Kubernetes naming conventions it targets.

## Object Literals vs Blocks

HCL object literals (used in attribute values) and blocks look similar but are different:

```hcl
# This is a BLOCK -- it defines structure in the DSL
resource my-bucket {
  # This is an ATTRIBUTE with an object literal VALUE
  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    metadata = {
      name = "foo" # nested object literal
      labels = {
        env = "prod"
      }
    }
  }
}
```

The key distinction:
- **Blocks** define the structure of your function-hcl program (`resource`, `locals`, `composite status`, etc.)
- **Object literals** define data values (the Kubernetes manifests, status fields, etc.)

Inside object literals, you can use expressions, string templates, and function calls freely.

## Summary

| Construct | Purpose | Syntax | Can repeat? |
|-----------|---------|--------|-------------|
| Block | Contains attributes and blocks | `type [labels] { ... }` | Depends on type |
| Attribute | Assigns a value to a name | `name = value` | No (once per scope) |
| Identifier | Names things | `my-bucket`, `compName`, `region_1` | N/A |
| Object literal | Data value (maps/dicts) | `{ key = value }` | N/A (it's a value) |
