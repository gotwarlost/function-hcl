---
title: "Debugging Compositions"
linkTitle: "Debugging"
weight: 3
description: >
  How to get debug output from function-hcl to troubleshoot your compositions.
---

function-hcl can print the inputs and outputs of your HCL script to the function's logs,
making it easy to see exactly what data the function received and what resources it produced.
Debug output is formatted in [crossplane render](https://docs.crossplane.io/latest/cli/command-reference/#render)
style, so the printed objects can be pasted directly into render unit tests.

Inputs are pre-processed to strip noise — last-applied kubectl annotations, managed fields,
and similar metadata are removed so the output focuses on what matters.

There are two ways to enable debug output: per-XR via an annotation, or composition-wide via the
function input.

## Per-XR annotation

To debug a single XR without changing the function configuration, annotate it:

```bash
kubectl annotate <xr-type> <xr-name> hcl.fn.crossplane.io/debug=true
```

The function will emit debug output the next time it reconciles that XR. Remove the annotation
when you are done to stop the output.

## Composition-wide debug mode

To enable debug output for **all** XRs processed by a composition, set `debug: true` in
the composition's function input:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: composition-name
spec:
  mode: Pipeline
  pipeline:
    - functionRef:
        name: fn-hcl
      input:
        apiVersion: function-hcl/v1
        debug: true
        hcl: |
          # your HCL code
```

This emits a full dump for every XR that uses this composition on every reconcile. In a
cluster with many such XRs this produces a significant volume of logs and is only suitable
in development environments. Use the per-XR annotation instead when you need to debug in
production.

## Debug output for new XRs

When a new XR is created for the first time, the composition reconciles it before you have
a chance to add an annotation. The `debugNew` option emits debug output for these first-time
reconciles automatically.

A XR is considered "new" when the request contains an observed composite but no other observed
resources yet.

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: composition-name
spec:
  mode: Pipeline
  pipeline:
  - functionRef:
      name: fn-hcl
    input:
      apiVersion: function-hcl/v1
      debugNew: true
      hcl: |
        # your HCL code
```

You can combine `debugNew: true` with the per-XR annotation to cover both first-time
reconciles and subsequent ones without enabling debug for every XR using this composition.

## Reading the debug output

Debug output appears in the function-hcl pod logs. To stream them:

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/function=function-hcl -f
```

### Output format

The output is in [txtar](https://pkg.go.dev/golang.org/x/tools/txtar) format — the same
format used by `fn-hcl-tools package`. Each
reconcile produces two blocks in the logs: one for the request and one for the response,
each delimited by a `## start … ##` / `## end … ##` marker.

```
## start request ##

-- xr.yaml --
apiVersion: example.crossplane.io/v1alpha1
kind: MyComposite
metadata:
  name: my-xr
spec:
  parameters:
    region: us-east-1

-- observed.yaml --
# crossplane name: my-bucket
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
...

-- context-apiextensions.crossplane.io--environment.json --
{
  "key": "value"
}

## end request ##

## start response ##

-- rendered.yaml --
# returned composite status
apiVersion: example.crossplane.io/v1alpha1
kind: MyComposite
...
---
# desired object: my-bucket
apiVersion: s3.aws.upbound.io/v1beta1
kind: Bucket
...

## end response ##
```

**Request block files:**

| File | Contents |
|------|----------|
| `xr.yaml` | The observed composite resource (XR), cleaned of noise |
| `observed.yaml` | All currently observed composed resources, as a multi-document YAML |
| `context-<key>.json` | One JSON file per context key (e.g. the Crossplane environment) |
| `extra-resources.yaml` | Extra resources returned by a requirements step (if any) |

**Response block files:**

| File | Contents |
|------|----------|
| `rendered.yaml` | Multi-document YAML: desired composite status, all desired composed resources, context, and function results |
| `requirements.yaml` | Extra-resource requirements emitted by the function (if any) |

### Extracting files from the output

Because the output is valid txtar, you can pipe it into the `txtar` tool to split it into
individual files on disk. Install the tool with:

```bash
go install golang.org/x/tools/cmd/txtar@latest
```

Then capture the relevant block from the logs:

```bash
# Save the request block for one reconcile to a file
kubectl logs -n crossplane-system -l pkg.crossplane.io/function=function-hcl \
  | awk '/## start request ##/{found=1} found{print} /## end request ##/{found=0}' \
  > request.txtar
```

and extract it:

```bash
# Extract the individual files into the current directory
txtar -x < request.txtar
```

This produces `xr.yaml`, `observed.yaml`, and any context files as separate files that
can be dropped directly into a `crossplane render` test fixture.
