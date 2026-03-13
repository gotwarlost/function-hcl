---
title: "function-hcl"
linkTitle: "function-hcl"
---

{{< blocks/cover title="function-hcl" image_anchor="top" height="min" >}}
<a class="btn btn-lg btn-primary me-3 mb-4" href="/function-hcl/docs/getting-started/">
  Get Started <i class="fas fa-arrow-alt-circle-right ms-2"></i>
</a>
<a class="btn btn-lg btn-secondary me-3 mb-4" href="https://github.com/crossplane-contrib/function-hcl">
  GitHub <i class="fab fa-github ms-2 "></i>
</a>
<p class="lead">A Crossplane composition function that uses an opinionated HCL-based DSL to model desired resources — with a familiar feel for anyone coming from Terraform.</p>

```hcl
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

{{< /blocks/cover >}}
