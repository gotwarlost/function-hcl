function-hcl-ls
---

A language server implementation for function-hcl.

Provides the following features:

* Completion
* Hover descriptions
* Goto Declaration/ Find references
* Semantic tokens
* Validations

Completion requires provider CRDs to be set up such that the language server can find type definitions. 

The exact mechanics of how to set this up along with IDE integration will be documented under 
[the docs page](https://crossplane-contrib.github.io/function-hcl/) when the client code is added to the repo.

## Standing on the shoulders of giants

This repo owes a lot to the [HCL language library](https://github.com/hashicorp/hcl-lang) and 
the [Terraform language server](https://github.com/hashicorp/terraform-ls) implementation.

This repo contains code copied from the above repos and modified for use. Since function-hcl is different
enough from Terraform, the borrowed code did not work well for it but gave the authors a great starting
point. A lot of refactoring has been done on both these copied codebases to make the language server
work for function-hcl.

In addition, the `go.mod` replaces the [HCL dependency](https://github.com/hashicorp/hcl) 
with a [fork](https://github.com/gotwarlost/hcl) because of a [critical fix](https://github.com/hashicorp/hcl/pull/785) 
for a [known issue](https://github.com/hashicorp/hcl/issues/597) that is needed.

You cannot build this repo using `go install github.com/...` because of this. For source builds, you will need to clone 
the repo and build it locally.
