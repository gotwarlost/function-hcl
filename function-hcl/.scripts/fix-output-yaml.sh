#!/bin/bash

set -euo pipefail

# script to assign predictable names to pseudo-objects like context, result etc.

# sort all Result objects to the top such that they can always be indexed using 0,1,2 etc.
# done using sort order = 0 if Result, 1 otherwise
# original sort order is captured to be the document index
#
# then: eval all docs sorting by sort order, assign predictable names, and re-sort to match
# original document and remove temp vars.

cat | yq '
    .originalSortOrder = document_index |
    with(select(.kind == "Result"); .sortOrder = 0) |
    with(select(.kind != "Result"); .sortOrder = 1)
  ' | \
  yq  eval-all '
    [.] | sort_by(.sortOrder) | .[] | split_doc |
   (with(select(.kind == "Context"); .metadata.name = "context")) |
   (with(select(.kind == "Result"); .metadata.name = "r-" + document_index )) |
   del(.sortOrder) |
   [.] | sort_by(.originalSortOrder) | .[] | split_doc |
  del(.originalSortOrder)
'

