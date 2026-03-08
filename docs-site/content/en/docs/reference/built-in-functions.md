---
title: "Built-in Functions"
linkTitle: "Built-in Functions"
weight: 2
description: >
  Complete list of available functions.
---

function-hcl supports all Terraform functions from v1.5.7, with the exceptions noted below.

## Excluded Functions

### File I/O functions (not available)

function-hcl is a memory-only system with no filesystem access:

`file`, `fileexists`, `fileset`, `filebase64`, `templatefile`, `abspath`, `pathexpand`, `basename`, `dirname`

### Impure functions (not available)

These introduce non-determinism. function-hcl is designed to be hermetic -- the same inputs always
produce the same outputs:

`uuid`, `uuidv5`, `timestamp`, `plantimestamp`, `bcrypt`

## Available Functions by Category

### Numeric

| Function | Description |
|---------|-------------|
| `abs(n)` | Absolute value |
| `ceil(n)` | Round up to nearest integer |
| `floor(n)` | Round down to nearest integer |
| `log(n, base)` | Logarithm |
| `max(n1, n2, ...)` | Maximum value |
| `min(n1, n2, ...)` | Minimum value |
| `parseint(str, base)` | Parse integer from string |
| `pow(base, exp)` | Exponentiation |
| `signum(n)` | Sign of number (-1, 0, 1) |

### String

| Function | Description |
|---------|-------------|
| `chomp(str)` | Remove trailing newlines |
| `endswith(str, suffix)` | Check suffix |
| `format(fmt, args...)` | String formatting (sprintf-style) |
| `formatlist(fmt, list...)` | Format each element |
| `indent(spaces, str)` | Indent all lines |
| `join(sep, list)` | Join list elements |
| `lower(str)` | Lowercase |
| `regex(pattern, str)` | Regex match |
| `regexall(pattern, str)` | All regex matches |
| `replace(str, old, new)` | String replacement |
| `split(sep, str)` | Split string |
| `startswith(str, prefix)` | Check prefix |
| `strrev(str)` | Reverse string |
| `substr(str, offset, length)` | Substring |
| `title(str)` | Title case |
| `trim(str, chars)` | Trim characters |
| `trimprefix(str, prefix)` | Trim prefix |
| `trimsuffix(str, suffix)` | Trim suffix |
| `trimspace(str)` | Trim whitespace |
| `upper(str)` | Uppercase |

### Collection

| Function | Description |
|---------|-------------|
| `alltrue(list)` | All elements are true |
| `anytrue(list)` | Any element is true |
| `chunklist(list, size)` | Split list into chunks |
| `coalesce(vals...)` | First non-null, non-empty value |
| `coalescelist(lists...)` | First non-empty list |
| `compact(list)` | Remove empty strings |
| `concat(lists...)` | Concatenate lists |
| `contains(list, val)` | Check membership |
| `distinct(list)` | Remove duplicates |
| `element(list, idx)` | Get element by index (wraps) |
| `flatten(list)` | Flatten nested lists |
| `index(list, val)` | Find index of value |
| `keys(map)` | Map keys |
| `length(val)` | Length of string, list, or map |
| `list(vals...)` | Create a list |
| `lookup(map, key, default)` | Safe map lookup |
| `map(k1, v1, k2, v2, ...)` | Create a map |
| `matchkeys(vals, keys, search)` | Filter by matching keys |
| `merge(maps...)` | Merge maps |
| `one(list)` | Extract single element or null |
| `range(start, limit, step)` | Generate number sequence |
| `reverse(list)` | Reverse list |
| `setintersection(sets...)` | Set intersection |
| `setproduct(sets...)` | Cartesian product |
| `setsubtract(a, b)` | Set difference |
| `setunion(sets...)` | Set union |
| `slice(list, start, end)` | List slice |
| `sort(list)` | Sort strings lexicographically |
| `sum(list)` | Sum numbers |
| `transpose(map)` | Transpose map of lists |
| `values(map)` | Map values |
| `zipmap(keys, values)` | Create map from key/value lists |

### Encoding

| Function | Description |
|---------|-------------|
| `base64decode(str)` | Decode base64 |
| `base64encode(str)` | Encode to base64 |
| `base64gzip(str)` | Gzip then base64 encode |
| `csvdecode(str)` | Parse CSV |
| `jsondecode(str)` | Parse JSON |
| `jsonencode(val)` | Encode to JSON |
| `textdecodebase64(str, enc)` | Decode base64 with encoding |
| `textencodebase64(str, enc)` | Encode with encoding then base64 |
| `urlencode(str)` | URL encode |
| `yamldecode(str)` | Parse YAML |
| `yamlencode(val)` | Encode to YAML |

### Hash and Crypto

| Function | Description |
|---------|-------------|
| `base64sha256(str)` | Base64-encoded SHA256 |
| `base64sha512(str)` | Base64-encoded SHA512 |
| `md5(str)` | MD5 hash |
| `sha1(str)` | SHA1 hash |
| `sha256(str)` | SHA256 hash |
| `sha512(str)` | SHA512 hash |

### IP Network

| Function | Description |
|---------|-------------|
| `cidrhost(prefix, hostnum)` | Calculate host IP |
| `cidrnetmask(prefix)` | Network mask |
| `cidrsubnet(prefix, newbits, netnum)` | Calculate subnet |
| `cidrsubnets(prefix, newbits...)` | Calculate multiple subnets |

### Type Conversion

| Function | Description |
|---------|-------------|
| `can(expr)` | Test if expression evaluates without error |
| `nonsensitive(val)` | Remove sensitive marking |
| `sensitive(val)` | Mark as sensitive |
| `tobool(val)` | Convert to bool |
| `tolist(val)` | Convert to list |
| `tomap(val)` | Convert to map |
| `tonumber(val)` | Convert to number |
| `toset(val)` | Convert to set |
| `tostring(val)` | Convert to string |
| `try(exprs...)` | First expression that doesn't error |
| `type(val)` | Get type of value |

## Custom Functions

### `invoke`

```hcl
invoke("functionName", { arg1: value1, arg2: value2 })
```

Calls a [user-defined function](../../language-guide/user-functions/). The first argument must be
a static string. See the language guide for details.
