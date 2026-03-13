---
title: "Built-in Functions"
linkTitle: "Built-in Functions"
weight: 2
description: >
  Complete list of available functions.
---

function-hcl supports most Terraform functions as of v1.5.7. Exceptions are noted at the end of the page.

## Available Functions by Category

### Numeric

| Function                                                                                              | Description                   |
|-------------------------------------------------------------------------------------------------------|-------------------------------|
| [`abs(n)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/abs)                   | Absolute value                |
| [`ceil(n)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/ceil)                 | Round up to nearest integer   |
| [`floor(n)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/floor)               | Round down to nearest integer |
| [`log(n, base)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/log)             | Logarithm                     |
| [`max(n1, n2, ...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/max)         | Maximum value                 |
| [`min(n1, n2, ...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/min)         | Minimum value                 |
| [`parseint(str, base)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/parseint) | Parse integer from string     |
| [`pow(base, exp)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/pow)           | Exponentiation                |
| [`signum(n)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/signum)             | Sign of number (-1, 0, 1)     |

### String

| Function                                                                                                     | Description                       |
|--------------------------------------------------------------------------------------------------------------|-----------------------------------|
| [`chomp(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/chomp)                    | Remove trailing newlines          |
| [`endswith(str, suffix)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/endswith)      | Check suffix                      |
| [`format(fmt, args...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/format)         | String formatting (sprintf-style) |
| [`formatlist(fmt, list...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/formatlist) | Format each element               |
| [`indent(spaces, str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/indent)          | Indent all lines                  |
| [`join(sep, list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/join)                | Join list elements                |
| [`lower(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/lower)                    | Lowercase                         |
| [`regex(pattern, str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/regex)           | Regex match                       |
| [`regexall(pattern, str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/regexall)     | All regex matches                 |
| [`replace(str, old, new)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/replace)      | String replacement                |
| [`split(sep, str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/split)               | Split string                      |
| [`startswith(str, prefix)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/startswith)  | Check prefix                      |
| [`strrev(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/strrev)                  | Reverse string                    |
| [`substr(str, offset, length)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/substr)  | Substring                         |
| [`title(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/title)                    | Title case                        |
| [`trim(str, chars)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/trim)               | Trim characters                   |
| [`trimprefix(str, prefix)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/trimprefix)  | Trim prefix                       |
| [`trimsuffix(str, suffix)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/trimsuffix)  | Trim suffix                       |
| [`trimspace(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/trimspace)            | Trim whitespace                   |
| [`upper(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/upper)                    | Uppercase                         |

### Collection

| Function                                                                                                          | Description                     |
|-------------------------------------------------------------------------------------------------------------------|---------------------------------|
| [`alltrue(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/alltrue)                    | All elements are true           |
| [`anytrue(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/anytrue)                    | Any element is true             |
| [`chunklist(list, size)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/chunklist)          | Split list into chunks          |
| [`coalesce(vals...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/coalesce)               | First non-null, non-empty value |
| [`coalescelist(lists...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/coalescelist)      | First non-empty list            |
| [`compact(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/compact)                    | Remove empty strings            |
| [`concat(lists...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/concat)                  | Concatenate lists               |
| [`contains(list, val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/contains)             | Check membership                |
| [`distinct(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/distinct)                  | Remove duplicates               |
| [`element(list, idx)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/element)               | Get element by index (wraps)    |
| [`flatten(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/flatten)                    | Flatten nested lists            |
| [`index(list, val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/index)                   | Find index of value             |
| [`keys(map)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/keys)                           | Map keys                        |
| [`length(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/length)                       | Length of string, list, or map  |
| [`list(vals...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/list)                       | Create a list                   |
| [`lookup(map, key, default)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/lookup)         | Safe map lookup                 |
| [`map(k1, v1, k2, v2, ...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/map)             | Create a map                    |
| [`matchkeys(vals, keys, search)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/matchkeys)  | Filter by matching keys         |
| [`merge(maps...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/merge)                     | Merge maps                      |
| [`one(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/one)                            | Extract single element or null  |
| [`range(start, limit, step)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/range)          | Generate number sequence        |
| [`reverse(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/reverse)                    | Reverse list                    |
| [`setintersection(sets...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/setintersection) | Set intersection                |
| [`setproduct(sets...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/setproduct)           | Cartesian product               |
| [`setsubtract(a, b)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/setsubtract)            | Set difference                  |
| [`setunion(sets...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/setunion)               | Set union                       |
| [`slice(list, start, end)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/slice)            | List slice                      |
| [`sort(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sort)                          | Sort strings lexicographically  |
| [`sum(list)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sum)                            | Sum numbers                     |
| [`transpose(map)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/transpose)                 | Transpose map of lists          |
| [`values(map)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/values)                       | Map values                      |
| [`zipmap(keys, values)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/zipmap)              | Create map from key/value lists |

### Encoding

| Function | Description |
|---------|-------------|
| [`base64decode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/base64decode) | Decode base64 |
| [`base64encode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/base64encode) | Encode to base64 |
| [`base64gzip(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/base64gzip) | Gzip then base64 encode |
| [`csvdecode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/csvdecode) | Parse CSV |
| [`jsondecode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/jsondecode) | Parse JSON |
| [`jsonencode(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/jsonencode) | Encode to JSON |
| [`textdecodebase64(str, enc)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/textdecodebase64) | Decode base64 with encoding |
| [`textencodebase64(str, enc)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/textencodebase64) | Encode with encoding then base64 |
| [`urlencode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/urlencode) | URL encode |
| [`yamldecode(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/yamldecode) | Parse YAML |
| [`yamlencode(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/yamlencode) | Encode to YAML |

### Hash and Crypto

| Function | Description |
|---------|-------------|
| [`base64sha256(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/base64sha256) | Base64-encoded SHA256 |
| [`base64sha512(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/base64sha512) | Base64-encoded SHA512 |
| [`md5(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/md5) | MD5 hash |
| [`sha1(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sha1) | SHA1 hash |
| [`sha256(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sha256) | SHA256 hash |
| [`sha512(str)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sha512) | SHA512 hash |

### IP Network

| Function | Description |
|---------|-------------|
| [`cidrhost(prefix, hostnum)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/cidrhost) | Calculate host IP |
| [`cidrnetmask(prefix)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/cidrnetmask) | Network mask |
| [`cidrsubnet(prefix, newbits, netnum)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/cidrsubnet) | Calculate subnet |
| [`cidrsubnets(prefix, newbits...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/cidrsubnets) | Calculate multiple subnets |

### Type Conversion

| Function | Description |
|---------|-------------|
| [`can(expr)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/can) | Test if expression evaluates without error |
| [`nonsensitive(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/nonsensitive) | Remove sensitive marking |
| [`sensitive(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/sensitive) | Mark as sensitive |
| [`tobool(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/tobool) | Convert to bool |
| [`tolist(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/tolist) | Convert to list |
| [`tomap(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/tomap) | Convert to map |
| [`tonumber(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/tonumber) | Convert to number |
| [`toset(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/toset) | Convert to set |
| [`tostring(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/tostring) | Convert to string |
| [`try(exprs...)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/try) | First expression that doesn't error |
| [`type(val)`](https://developer.hashicorp.com/terraform/language/v1.5.x/functions/type) | Get type of value |

## Custom Functions

### `invoke`

```hcl
invoke("functionName", { arg1: value1, arg2: value2 })
```

Calls a [user-defined function](../../language-guide/user-functions/). The first argument must be
a static string. See the language guide for details.

## Excluded Functions

### File I/O functions (not available)

function-hcl is a memory-only system with no filesystem access:

`file`, `fileexists`, `fileset`, `filebase64`, `templatefile`, `abspath`, `pathexpand`, `basename`, `dirname`

### Impure functions (not available)

These introduce non-determinism. function-hcl is designed to be hermetic -- the same inputs always
produce the same outputs:

`uuid`, `uuidv5`, `timestamp`, `plantimestamp`, `bcrypt`

