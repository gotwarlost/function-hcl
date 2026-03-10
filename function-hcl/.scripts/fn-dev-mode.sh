#!/bin/bash

set -euo pipefail

cat | yq '
  with(select(.metadata.name == "fn-hcl"); .metadata.annotations."render.crossplane.io/runtime" = "Development")
'
