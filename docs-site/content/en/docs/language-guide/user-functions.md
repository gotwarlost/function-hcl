---
title: "User-Defined Functions"
linkTitle: "User Functions"
weight: 14
description: >
  Writing and invoking custom functions.
---

function-hcl lets you define reusable functions in your HCL code.

## Defining Functions

Functions are defined with the `function` block. They **must** be defined at the top level
(not inside a `group` or other block).

```hcl
function addNumbers {
  arg a {}
  arg b { default = 1 }

  locals {
    output = a + b
  }

  body = output
}
```

### Structure

- **`arg` blocks** define the function's parameters. Each arg can have an optional `default` value
  and an optional `description`.
- **`locals` blocks** can be used for temporary calculations.
- **`body`** is the return value of the function.

### Scoping Rules

Functions do **not** have access to external state. You cannot use `req.composite`, `self`, or
any other variable from the calling context inside a function. Only the declared arguments and
locals within the function are accessible.

A function **can** call other standard built-in functions and invoke other user-defined functions.

## Invoking Functions

Use the built-in `invoke` function:

```hcl
locals {
  c = invoke("addNumbers", { a = 2, b = 3 }) # returns 5
}
```

- The first parameter is the function name and **must be a static string** (no variables).
- The second parameter is an object providing values for the function's arguments.
- Arguments with defaults may be omitted.

## A More Practical Example

```hcl
function toProviderK8sObject {
  arg name {
    description = "metadata name of the return object"
  }
  arg manifest {
    description = "the inner manifest to be wrapped"
  }
  arg providerName {
    description = "name of the K8s provider"
    default     = "default"
  }

  locals {
    objectName = "foo-${name}"
  }

  body = {
    apiVersion = "kubernetes.crossplane.io/v1alpha1"
    kind       = "Object"
    metadata = {
      name = objectName
    }
    spec = {
      forProvider = {
        manifest = manifest
      }
      providerConfigRef = {
        name = providerName
      }
    }
  }
}

resource local-provider-config {
  locals {
    manifest = {
      apiVersion = "kubernetes.crossplane.io/v1alpha1"
      kind       = "ProviderConfig"
    }
  }

  body = invoke("toProviderK8sObject", {
    name     = "foobar"
    manifest = manifest
  })
}
```

## Recursion

Self-recursive and mutually-recursive functions are possible but not encouraged:

```hcl
function factorial {
  arg n {}
  body = n < 1 ? 1 : n * invoke("factorial", { n = n - 1 })
}
```

Infinite recursion is prevented by a call stack limit of **100**. Exceeding this limit produces an error.

## Error Conditions

The function returns an error if:
- A function or arg has a name that is not a valid identifier
- `invoke` references a non-existent function
- `invoke` is called with missing required arguments or unrecognized argument names
- The call stack exceeds 100 frames
