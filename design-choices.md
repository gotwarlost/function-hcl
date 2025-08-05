design choices
---

This page documents some design choices we made when designing the DSL for function-hcl.

* How close should it be to Terraform? We started with "as close as possible" but realized that there be dragons in this
  approach. Things looked Terraform-y but had subtly different semantics than Terraform. So the compromise is:
  * make it look different to signal that this is not Terraform
  * ensure that Terraform and function-hcl constructs have a rough mapping such that it would be possible to write a 
    converter with some heuristics in the future.
  * support all Terraform functions that can reasonably be supported
  * one issue is the number of labels for the `resource` block (1 for function-hcl, 2 for terraform) but we couldn't come
    up with a different name for `resource` :) 

* `resource` (singular) distinct from `resources` (collection) as opposed to terraform where a collection is just another
resource with a `for_each` or a `count`. Observed variables follow the same naming pattern.

* Local variables
  * are the ones accessed without a namespace unlike terraform where it is the resource
  * can always be precomputed before user code accesses it. 
  * can be specified at multiple scopes to make them truly "local"
  * can be specified in any order just like Terraform otherwise we wouldn't be able to support file-scoped locals across
    multiple files.

* Ability to access the observed version of a resource or collection easily in a resource block using `self.resource[s]`

* Make `each` a reserved word for ergonomics (although we could have done `self.each` instead)

* `condition` (boolean) in lieu of Terraform's `count` since `count` can be expressed as `for_each = range(n)`

* Hat-tip to the [cue](https://cuelang.org/) language for giving us ideas on the status unification features and use of
  the [txtar](https://pkg.go.dev/golang.org/x/tools/txtar#hdr-Txtar_format) format as input to this function.
