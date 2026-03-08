---
title: "function-hcl"
linkTitle: "function-hcl"
---

{{< blocks/cover title="function-hcl" image_anchor="top" height="full" >}}
<a class="btn btn-lg btn-primary me-3 mb-4" href="/docs/getting-started/">
  Get Started <i class="fas fa-arrow-alt-circle-right ms-2"></i>
</a>
<a class="btn btn-lg btn-secondary me-3 mb-4" href="https://github.com/crossplane-contrib/function-hcl">
  GitHub <i class="fab fa-github ms-2 "></i>
</a>
<p class="lead mt-5">A Crossplane composition function that uses an opinionated HCL-based DSL to model desired resources — with a familiar feel for anyone coming from Terraform.</p>
{{< blocks/link-down color="info" >}}
{{< /blocks/cover >}}

{{% blocks/lead color="primary" %}}
**function-hcl** lets you write Crossplane Compositions using HCL — the same language as Terraform — giving you
locals, resource blocks, expressions, and automatic dependency resolution without any YAML gymnastics.
{{% /blocks/lead %}}

{{% blocks/section color="dark" type="row" %}}

{{% blocks/feature icon="fa-code" title="HCL DSL" %}}
Write compositions in a clean, expressive DSL built on HashiCorp HCL. Locals, resource blocks, and expressions
work the way Terraform users expect.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-magic" title="Automatic Dependency Resolution" %}}
Resources that depend on values not yet available are automatically deferred — and will never drop a resource
that already exists.
{{% /blocks/feature %}}

{{% blocks/feature icon="fa-file-archive" title="txtar Multi-file Support" %}}
Bundle multiple HCL files into a single input using the txtar format, preserving filenames and line numbers
in all error messages.
{{% /blocks/feature %}}

{{% /blocks/section %}}

{{% blocks/section %}}

## Quick Example

```hcl
-- main.hcl --

locals {
  comp     = req.composite
  compName = comp.metadata.name
  params   = comp.spec.parameters
}

resource my-bucket {
  locals {
    region = params.region
  }

  body = {
    apiVersion = "s3.aws.upbound.io/v1beta1"
    kind       = "Bucket"
    metadata = {
      name = "${compName}-bucket"
    }
    spec = {
      forProvider = {
        region = region
      }
    }
  }

  composite status {
    body = {
      bucketArn = self.resource.status.atProvider.arn
    }
  }
}
```

{{% /blocks/section %}}
