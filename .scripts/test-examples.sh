#!/bin/bash

set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

"${SCRIPT_DIR}/gen-comp.sh" >/dev/null
echo ---

cd  "${SCRIPT_DIR}/../example"

for dir in *
do
  if [[ ! -d "${dir}" ]]
  then
    continue
  fi
  echo $dir>&2
  cd $dir
  cmd="crossplane render xr.yaml composition.yaml functions.yaml"
  if [[ -f observed.yaml ]]
  then
    cmd="${cmd} -o observed.yaml"
  fi
  dyff between --omit-header --additional-identifier type --set-exit-code src/expected.yaml  <(${cmd})
  cd ..
done
