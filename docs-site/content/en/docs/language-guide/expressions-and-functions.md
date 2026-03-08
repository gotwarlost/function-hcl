---
title: "Expressions and Built-in Functions"
linkTitle: "Expressions & Functions"
weight: 15
description: >
  HCL expressions and the available function library.
---

## Expressions

function-hcl supports all HCL syntax as specified in the
[HCL syntax specification](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md):

- **String templates**: `"${name}-suffix"`, `"Hello, ${upper(name)}!"`
- **Arithmetic**: `+`, `-`, `*`, `/`, `%`
- **Comparison**: `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Logical**: `&&`, `||`, `!`
- **Conditional**: `condition ? true_val : false_val`
- **For expressions**: `[for s in list : upper(s)]`, `{for k, v in map : k => upper(v)}`
- **Splat**: `list[*].attribute`
- **Index**: `list[0]`, `map["key"]`
- **Attribute access**: `object.attribute`

## Built-in Functions

All Terraform functions as of v1.5.7 are supported, with two exceptions:

1. **No file I/O functions** -- function-hcl is a memory-only system. Functions like `file()`,
   `templatefile()`, `fileset()`, etc. are not available.
2. **No impure functions** -- functions like `uuid()`, `timestamp()`, etc. are not available.
   The intent is a hermetic system where a given set of inputs always produces the same outputs.

### Commonly Used Functions

| Function | Description | Example |
|---------|-------------|---------|
| `format(fmt, args...)` | String formatting (like sprintf) | `format("%-20s", name)` |
| `merge(map1, map2, ...)` | Merge maps (rightmost wins) | `merge(defaults, overrides)` |
| `lookup(map, key, default)` | Safe map lookup | `lookup(tags, "env", "dev")` |
| `coalesce(vals...)` | First non-null, non-empty value | `coalesce(var, "default")` |
| `try(exprs...)` | First expression that doesn't error | `try(obj.field, "fallback")` |
| `can(expr)` | True if expression evaluates without error | `can(obj.field)` |
| `tostring(val)` | Convert to string | `tostring(42)` |
| `tonumber(val)` | Convert to number | `tonumber("42")` |
| `tobool(val)` | Convert to bool | `tobool("true")` |
| `length(val)` | Length of string, list, or map | `length(list)` |
| `keys(map)` | List of keys | `keys(tags)` |
| `values(map)` | List of values | `values(tags)` |
| `contains(list, val)` | Check if list contains value | `contains(regions, "us-east-1")` |
| `range(n)` | Generate list [0..n-1] | `range(5)` |
| `flatten(list)` | Flatten nested lists | `flatten([["a"], ["b"]])` |
| `distinct(list)` | Remove duplicates | `distinct(names)` |
| `join(sep, list)` | Join strings | `join(",", tags)` |
| `split(sep, str)` | Split string | `split(",", "a,b,c")` |
| `replace(str, old, new)` | String replace | `replace(name, "_", "-")` |
| `regex(pat, str)` | Regex match | `regex("^[a-z]+", name)` |
| `base64encode(str)` | Base64 encode | `base64encode(secret)` |
| `base64decode(str)` | Base64 decode | `base64decode(encoded)` |
| `jsonencode(val)` | Encode as JSON | `jsonencode(obj)` |
| `jsondecode(str)` | Decode JSON string | `jsondecode(raw)` |
| `toset(list)` | Convert list to set | `toset(names)` |
| `tolist(set)` | Convert set to list | `tolist(names)` |
| `tomap(obj)` | Convert to map | `tomap(obj)` |

### The `invoke` Function

In addition to the standard library, function-hcl adds the `invoke` function for calling
[user-defined functions](../user-functions/):

```hcl
invoke("functionName", { arg1: value1, arg2: value2 })
```

For the complete list of available functions, see the
[Terraform function documentation](https://developer.hashicorp.com/terraform/language/functions)
(v1.5.7), keeping in mind the file I/O and impurity exclusions noted above.
