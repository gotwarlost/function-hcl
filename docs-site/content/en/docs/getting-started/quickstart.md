---
title: "Quickstart"
linkTitle: "Quickstart"
weight: 2
description: >
  Write your first HCL-based Crossplane composition in five minutes.
---

This quickstart creates a simple composition that provisions an S3 bucket using function-hcl.

## 1. Define a Composite Resource

First, define an XRD and a Claim type for a storage bucket:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xstoragebuckets.example.crossplane.io
spec:
  group: example.crossplane.io
  names:
    kind: XStorageBucket
    plural: xstoragebuckets
  claimNames:
    kind: StorageBucket
    plural: storagebuckets
  versions:
    - name: v1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  required: [region]
                  properties:
                    region:
                      type: string
                      description: AWS region for the bucket
            status:
              type: object
              properties:
                bucketArn:
                  type: string
```

Apply it:

```bash
kubectl apply -f xrd.yaml
```

## 2. Write the Composition

Create a Composition that uses function-hcl:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: storage-bucket
spec:
  compositeTypeRef:
    apiVersion: example.crossplane.io/v1
    kind: XStorageBucket
  mode: Pipeline
  pipeline:
    - step: create-bucket
      functionRef:
        name: function-hcl
      input: |
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

Apply it:

```bash
kubectl apply -f composition.yaml
```

## 3. Create a Claim

```yaml
apiVersion: example.crossplane.io/v1
kind: StorageBucket
metadata:
  name: my-first-bucket
  namespace: default
spec:
  parameters:
    region: us-east-1
  compositionRef:
    name: storage-bucket
```

Apply it:

```bash
kubectl apply -f claim.yaml
```

## 4. Observe the Result

```bash
kubectl get storagebucket my-first-bucket
kubectl get bucket my-first-bucket-bucket
```

Once the bucket is provisioned, the `bucketArn` will be written back to the claim's status automatically —
because of the `composite status` block in the HCL.

## Next Steps

- Learn about [Concepts](../concepts/) like locals, resource blocks, and dependency resolution
- Read the full [Reference](../reference/)
